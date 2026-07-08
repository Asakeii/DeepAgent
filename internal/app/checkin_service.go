package app

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/agent"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
)

type CheckinService struct{}

type CheckinTurnRequest struct {
	ThreadID string
	Messages []*schema.Message
}

type CheckinTurnResult struct {
	Response       *schema.Message
	ReminderEvents []scheduler.ReminderEvent
}

func NewCheckinService() *CheckinService {
	return &CheckinService{}
}

func (s *CheckinService) RunTurn(ctx context.Context, req CheckinTurnRequest) (CheckinTurnResult, error) {
	var reminderEvents []scheduler.ReminderEvent
	var reminderMu sync.Mutex
	checkinCtx := scheduler.WithEventSink(ctx, func(event scheduler.ReminderEvent) {
		reminderMu.Lock()
		defer reminderMu.Unlock()
		reminderEvents = append(reminderEvents, event)
	})

	resp, err := agent.RunCheckin(checkinCtx, req.Messages, req.ThreadID)
	if err != nil {
		return CheckinTurnResult{}, err
	}

	reminderMu.Lock()
	events := append([]scheduler.ReminderEvent(nil), reminderEvents...)
	reminderMu.Unlock()
	return CheckinTurnResult{Response: resp, ReminderEvents: events}, nil
}

func (s *CheckinService) AnalyzeImage(ctx context.Context, req model.ChatRequest) (string, error) {
	txt := ""
	if len(req.Messages) > 0 {
		txt = req.Messages[len(req.Messages)-1].Content
	}
	return agent.AnalyzeFoodImage(ctx, req.ImageBase64, txt, req.ThreadID)
}

func (s *CheckinService) EmitResult(writer EventWriter, threadID string, result CheckinTurnResult) {
	for _, event := range result.ReminderEvents {
		_ = writer.WriteEvent("reminder_scheduled", &model.ChatResp{
			ThreadID: threadID,
			Role:     "assistant",
			Content:  event.Message,
			Reminder: reminderResp(event),
		})
	}
	content := ""
	if result.Response != nil {
		content = result.Response.Content
	}
	_ = writer.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: content})
}
