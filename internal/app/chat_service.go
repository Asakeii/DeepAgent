package app

import (
	"context"

	"deepAgent/internal/model"
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

	detach := s.Reminders.AttachStream(ctx, req.ThreadID, writer)
	defer detach()

	if req.ImageBase64 != "" {
		resp, err := s.Checkin.AnalyzeImage(ctx, req)
		if err != nil {
			_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "分析失败: " + err.Error()})
			return
		}
		_ = writer.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp})
		return
	}

	result, err := s.Research.Run(ctx, req, writer)
	if err != nil {
		return
	}
	if result.RouteToCheckin {
		checkinResult, err := s.Checkin.RunTurn(ctx, CheckinTurnRequest{
			ThreadID: req.ThreadID,
			Messages: req.Messages,
		})
		if err != nil {
			return
		}
		s.Checkin.EmitResult(writer, req.ThreadID, checkinResult)
		return
	}
	persistResearchMessages(ctx, req.ThreadID, req.Messages, result.Final)
}

func (s *ChatService) RunToText(ctx context.Context, req model.ChatRequest) string {
	writer := NewCaptureWriter()
	s.RunStream(ctx, req, writer)
	return writer.FinalContent()
}
