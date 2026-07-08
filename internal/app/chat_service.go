package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
	if err := store.EnsureThread(ctx, infra.DB, req.ThreadID, req.UserID, firstUserMessage(req), mode); err != nil {
		runLog.ErrorContext(ctx, "ensure thread failed", slog.Any("error", err))
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
	toolCtx := toolruntime.WithAuditContext(runCtx, toolruntime.AuditContext{
		RunID:    req.RunID,
		ThreadID: req.ThreadID,
		UserID:   req.UserID,
	})
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

func persistResearchArtifact(ctx context.Context, req model.ChatRequest, final string) error {
	if final == "" {
		return nil
	}
	metadata, _ := json.Marshal(map[string]any{
		"run_id":    req.RunID,
		"thread_id": req.ThreadID,
		"mode":      "research",
	})
	_, err := store.CreateArtifact(ctx, infra.DB, store.ArtifactRecord{
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
	return err
}
