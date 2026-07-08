package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/agent"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
	"deepAgent/internal/store"
	"deepAgent/internal/toolruntime"
)

type CheckinService struct{}

type CheckinTurnRequest struct {
	RunID    string
	UserID   string
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
	req.UserID = store.NormalizeUserID(req.UserID)
	if err := store.EnsureThread(ctx, infra.DB, req.ThreadID, req.UserID, firstCheckinUserMessage(req), "checkin"); err != nil {
		log.Printf("[thread] ensure checkin thread=%s user=%s: %v", req.ThreadID, req.UserID, err)
	}
	if ok, err := store.ThreadBelongsToUser(ctx, infra.DB, req.ThreadID, req.UserID); err != nil {
		return CheckinTurnResult{}, err
	} else if !ok {
		return CheckinTurnResult{}, fmt.Errorf("thread forbidden")
	}
	var reminderEvents []scheduler.ReminderEvent
	var reminderMu sync.Mutex
	checkinCtx := scheduler.WithEventSink(ctx, func(event scheduler.ReminderEvent) {
		reminderMu.Lock()
		defer reminderMu.Unlock()
		reminderEvents = append(reminderEvents, event)
	})
	checkinCtx = toolruntime.WithAuditContext(checkinCtx, toolruntime.AuditContext{
		RunID:    req.RunID,
		ThreadID: req.ThreadID,
		UserID:   req.UserID,
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

func firstCheckinUserMessage(req CheckinTurnRequest) string {
	for _, msg := range req.Messages {
		if msg != nil && msg.Role == schema.User && msg.Content != "" {
			return msg.Content
		}
	}
	return ""
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
			Agent:    "checkin",
			Role:     "assistant",
			Content:  event.Message,
			Reminder: reminderResp(event),
		})
	}
	content := ""
	if result.Response != nil {
		content = result.Response.Content
	}
	_ = writer.WriteEvent("message", &model.ChatResp{Agent: "checkin", Role: "assistant", Content: content})
}
