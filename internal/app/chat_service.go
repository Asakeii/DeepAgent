package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/observability"
	"deepAgent/internal/store"
	"deepAgent/internal/toolruntime"
)

// ChatService orchestrates one chat turn while keeping transport handlers thin.
type ChatService struct {
	Research  ResearchRunner
	Checkin   CheckinRunner
	Reminders ReminderStreamer
}

func NewChatService() *ChatService {
	return NewChatServiceWithDeps(NewResearchService(), NewCheckinService(), NewReminderService())
}

func NewChatServiceWithDeps(research ResearchRunner, checkin CheckinRunner, reminders ReminderStreamer) *ChatService {
	if research == nil {
		research = NewResearchService()
	}
	if checkin == nil {
		checkin = NewCheckinService()
	}
	if reminders == nil {
		reminders = NewReminderService()
	}
	return &ChatService{Research: research, Checkin: checkin, Reminders: reminders}
}

// RunStream preserves the existing /chat/stream event contract.
func (s *ChatService) RunStream(ctx context.Context, req model.ChatRequest, writer EventWriter) {
	req.InterruptFeedback = NormalizeInterruptFeedback(req.InterruptFeedback)
	if req.RunID == "" {
		req.RunID = newRunID()
	}
	if req.ThreadID == "" {
		req.ThreadID = req.RunID
	}
	req.UserID = store.NormalizeUserID(req.UserID)
	mode := "chat"
	if req.ImageBase64 != "" {
		mode = "image"
	}
	startedAt := time.Now()
	runLog := observability.RunLogger(req.RunID, req.ThreadID, req.UserID).With(slog.String("mode", mode))
	runLog.InfoContext(ctx, "chat run accepted")
	if err := store.EnsureThreadWithTeam(ctx, infra.DB, req.ThreadID, req.UserID, req.TeamID, firstUserMessage(req), mode); err != nil {
		runLog.ErrorContext(ctx, "ensure thread failed", slog.Any("error", err))
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "ensure thread failed: " + err.Error()})
		return
	}
	if ok, err := store.ThreadBelongsToUser(ctx, infra.DB, req.ThreadID, req.UserID); err != nil {
		runLog.ErrorContext(ctx, "thread ownership check failed", slog.Any("error", err))
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "thread ownership check failed: " + err.Error()})
		return
	} else if !ok {
		runLog.WarnContext(ctx, "thread ownership rejected")
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "thread forbidden"})
		return
	}
	settings, err := applyUserSettingsDefaults(ctx, &req)
	if err != nil {
		runLog.ErrorContext(ctx, "apply user settings failed", slog.Any("error", err))
	}
	persistExplicitMemories(ctx, req.UserID, req.ThreadID, req.Messages)
	originalMessages := append([]*schema.Message(nil), req.Messages...)
	agentMessages, err := messagesWithUserMemories(ctx, req.UserID, req.Messages)
	if err != nil {
		runLog.ErrorContext(ctx, "inject user memories failed", slog.Any("error", err))
	} else {
		req.Messages = agentMessages
	}
	if err := store.CreateRun(ctx, infra.DB, store.RunRecord{
		ID:       req.RunID,
		UserID:   req.UserID,
		ThreadID: req.ThreadID,
		Mode:     mode,
	}); err != nil {
		runLog.ErrorContext(ctx, "create run failed", slog.Any("error", err))
	}
	runCtx, stopCancellationWatcher := withRunCancellation(ctx, req.RunID)
	defer stopCancellationWatcher()
	runCtx, stopRunTimeout := withRunTimeout(runCtx)
	defer stopRunTimeout()
	runWriter := NewRunEventWriter(infra.DB, req.RunID, req.ThreadID, req.UserID, writer)
	req.ModelProfile = infra.NormalizeModelProfile(req.ModelProfile)
	if !infra.HasModelProfile(req.ModelProfile) {
		_ = runWriter.WriteEvent("error", &model.ChatResp{
			Role:         "assistant",
			Content:      "未知模型配置: " + req.ModelProfile,
			FinishReason: "invalid_model_profile",
		})
		return
	}
	runCtx = infra.WithModelProfile(runCtx, req.ModelProfile)
	toolCtx := toolruntime.WithAuditContext(runCtx, toolruntime.AuditContext{
		RunID:    req.RunID,
		ThreadID: req.ThreadID,
		UserID:   req.UserID,
	})
	toolCtx = infra.WithPluginScope(toolCtx, infra.PluginScope{UserID: req.UserID, TeamID: req.TeamID})
	defer func() {
		status := store.RunStatusSucceeded
		errText := ""
		if isRunCancelled(req.RunID) {
			status = store.RunStatusCancelled
		} else if failed, captured := runWriter.Failed(); failed {
			status = store.RunStatusFailed
			errText = captured
		}
		if err := store.CompleteRun(context.Background(), infra.DB, req.RunID, status, errText); err != nil {
			runLog.ErrorContext(context.Background(), "complete run failed", slog.String("status", status), slog.Any("error", err))
			return
		}
		runLog.InfoContext(context.Background(), "chat run completed",
			slog.String("status", status),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
	}()
	if exceeded, used, err := dailyTokenBudgetExceeded(ctx, req.UserID, settings); err != nil {
		runLog.ErrorContext(ctx, "daily token budget check failed", slog.Any("error", err))
		_ = runWriter.WriteEvent("error", &model.ChatResp{
			Role:         "assistant",
			Content:      "模型用量预算检查失败，请稍后重试",
			FinishReason: "token_budget_check_failed",
		})
		return
	} else if exceeded {
		_ = runWriter.WriteEvent("error", &model.ChatResp{
			Role:         "assistant",
			Content:      tokenBudgetExceededMessage(used, settings.DailyTokenBudget.Int64),
			FinishReason: "token_budget_exceeded",
		})
		return
	}
	if req.ImageBase64 != "" {
		if err := validateImageInput(req.ImageBase64); err != nil {
			_ = runWriter.WriteEvent("error", &model.ChatResp{
				Role:         "assistant",
				Content:      "图片输入不符合安全要求: " + err.Error(),
				FinishReason: "invalid_image",
			})
			return
		}
	}

	detach := s.Reminders.AttachStream(runCtx, req.ThreadID, runWriter)
	defer detach()

	if req.ImageBase64 != "" {
		resp, err := s.Checkin.AnalyzeImage(toolCtx, req)
		if err != nil {
			if isRunCancelled(req.RunID) {
				writeRunCancelled(runWriter, req.RunID, req.ThreadID)
				return
			}
			if isRunTimedOut(toolCtx, err) {
				writeRunTimedOut(runWriter, req.RunID, req.ThreadID)
				return
			}
			_ = runWriter.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "分析失败: " + err.Error()})
			return
		}
		_ = runWriter.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp})
		return
	}

	result, err := s.Research.Run(toolCtx, req, runWriter)
	if err != nil {
		if isRunCancelled(req.RunID) {
			writeRunCancelled(runWriter, req.RunID, req.ThreadID)
		} else if isRunTimedOut(toolCtx, err) {
			writeRunTimedOut(runWriter, req.RunID, req.ThreadID)
		}
		return
	}
	if result.RouteToCheckin {
		checkinResult, err := s.Checkin.RunTurn(toolCtx, CheckinTurnRequest{
			RunID:    req.RunID,
			UserID:   req.UserID,
			ThreadID: req.ThreadID,
			Messages: req.Messages,
		})
		if err != nil {
			if isRunCancelled(req.RunID) {
				writeRunCancelled(runWriter, req.RunID, req.ThreadID)
			} else if isRunTimedOut(toolCtx, err) {
				writeRunTimedOut(runWriter, req.RunID, req.ThreadID)
			}
			return
		}
		s.Checkin.EmitResult(runWriter, req.ThreadID, checkinResult)
		return
	}
	persistResearchMessages(ctx, req.ThreadID, originalMessages, result.Final)
	if err := persistResearchArtifact(ctx, req, result.Final); err != nil {
		runLog.ErrorContext(ctx, "persist research artifact failed", slog.Any("error", err))
	}
}

func (s *ChatService) RunToText(ctx context.Context, req model.ChatRequest) string {
	writer := NewCaptureWriter()
	s.RunStream(ctx, req, writer)
	return writer.FinalContent()
}

func withRunTimeout(ctx context.Context) (context.Context, func()) {
	if conf.App == nil {
		return ctx, func() {}
	}
	timeout := conf.App.Setting.RunTimeout()
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func isRunTimedOut(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}

func writeRunTimedOut(writer EventWriter, runID, threadID string) {
	_ = writer.WriteEvent("error", &model.ChatResp{
		RunID:        runID,
		ThreadID:     threadID,
		Role:         "assistant",
		Content:      "运行超时，请缩小任务范围或稍后重试",
		FinishReason: "timeout",
	})
}

func dailyTokenBudgetExceeded(ctx context.Context, userID string, settings store.UserSettingsRecord) (bool, int64, error) {
	if !settings.DailyTokenBudget.Valid {
		return false, 0, nil
	}
	used, err := store.SumUserModelTokensSince(ctx, infra.DB, userID, userDayStart(time.Now(), settings.Timezone))
	if err != nil {
		return false, 0, err
	}
	return used >= settings.DailyTokenBudget.Int64, used, nil
}

func userDayStart(now time.Time, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc, _ = time.LoadLocation(store.DefaultTimezone)
	}
	if loc == nil {
		loc = time.Local
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func tokenBudgetExceededMessage(used, limit int64) string {
	return "今日模型 token 用量已达到预算（已用 " + formatInt64(used) + " / 预算 " + formatInt64(limit) + "），请稍后再试或调整用户设置。"
}

func formatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func persistResearchArtifact(ctx context.Context, req model.ChatRequest, final string) error {
	if final == "" {
		return nil
	}
	metadata, _ := json.Marshal(map[string]any{
		"run_id":    req.RunID,
		"thread_id": req.ThreadID,
		"mode":      "research",
	})
	artifactID, err := store.CreateArtifact(ctx, infra.DB, store.ArtifactRecord{
		UserID:   req.UserID,
		ThreadID: req.ThreadID,
		RunID:    req.RunID,
		Kind:     store.ArtifactKindReport,
		Title:    firstUserMessage(req),
		Format:   store.ArtifactFormatMD,
		Content:  final,
		Metadata: metadata,
		Source:   store.ArtifactSourceAgent,
	})
	if err != nil {
		return err
	}
	citations := citationRecordsFromMarkdown(artifactID, req, final)
	return store.CreateArtifactCitations(ctx, infra.DB, citations)
}
