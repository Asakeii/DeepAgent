package app

import (
	"context"
	"log"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

// ChatService orchestrates one chat turn while keeping transport handlers thin.
type ChatService struct {
	Research  *ResearchService
	Checkin   *CheckinService
	Reminders *ReminderService
}

func NewChatService() *ChatService {
	return &ChatService{
		Research:  NewResearchService(),
		Checkin:   NewCheckinService(),
		Reminders: NewReminderService(),
	}
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
	if err := store.EnsureThread(ctx, infra.DB, req.ThreadID, req.UserID, firstUserMessage(req), mode); err != nil {
		log.Printf("[thread] ensure thread=%s user=%s: %v", req.ThreadID, req.UserID, err)
	}
	if ok, err := store.ThreadBelongsToUser(ctx, infra.DB, req.ThreadID, req.UserID); err != nil {
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "thread ownership check failed: " + err.Error()})
		return
	} else if !ok {
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "thread forbidden"})
		return
	}
	if err := store.CreateRun(ctx, infra.DB, store.RunRecord{
		ID:       req.RunID,
		UserID:   req.UserID,
		ThreadID: req.ThreadID,
		Mode:     mode,
	}); err != nil {
		log.Printf("[run] create run=%s thread=%s: %v", req.RunID, req.ThreadID, err)
	}
	runWriter := NewRunEventWriter(infra.DB, req.RunID, req.ThreadID, req.UserID, writer)
	defer func() {
		status := store.RunStatusSucceeded
		errText := ""
		if failed, captured := runWriter.Failed(); failed {
			status = store.RunStatusFailed
			errText = captured
		}
		if err := store.CompleteRun(context.Background(), infra.DB, req.RunID, status, errText); err != nil {
			log.Printf("[run] complete run=%s status=%s: %v", req.RunID, status, err)
		}
	}()

	detach := s.Reminders.AttachStream(ctx, req.ThreadID, runWriter)
	defer detach()

	if req.ImageBase64 != "" {
		resp, err := s.Checkin.AnalyzeImage(ctx, req)
		if err != nil {
			_ = runWriter.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "分析失败: " + err.Error()})
			return
		}
		_ = runWriter.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp})
		return
	}

	result, err := s.Research.Run(ctx, req, runWriter)
	if err != nil {
		return
	}
	if result.RouteToCheckin {
		checkinResult, err := s.Checkin.RunTurn(ctx, CheckinTurnRequest{
			UserID:   req.UserID,
			ThreadID: req.ThreadID,
			Messages: req.Messages,
		})
		if err != nil {
			return
		}
		s.Checkin.EmitResult(runWriter, req.ThreadID, checkinResult)
		return
	}
	persistResearchMessages(ctx, req.ThreadID, req.Messages, result.Final)
}

func (s *ChatService) RunToText(ctx context.Context, req model.ChatRequest) string {
	writer := NewCaptureWriter()
	s.RunStream(ctx, req, writer)
	return writer.FinalContent()
}
