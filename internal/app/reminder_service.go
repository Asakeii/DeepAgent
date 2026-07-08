package app

import (
	"context"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
)

type ReminderService struct{}

func NewReminderService() *ReminderService {
	return &ReminderService{}
}

func (s *ReminderService) AttachStream(ctx context.Context, threadID string, writer EventWriter) func() {
	if threadID == "" || infra.RDB == nil {
		return func() {}
	}
	notifCh := scheduler.DefaultRegistry.Register(threadID)
	scheduler.DrainPending(ctx, infra.RDB, threadID, notifCh)
	go func() {
		for event := range notifCh {
			_ = writer.WriteEvent("reminder", &model.ChatResp{
				ThreadID: threadID,
				Agent:    "reminder",
				Role:     "assistant",
				Content:  "提醒：" + event.Message,
				Reminder: reminderResp(event),
			})
		}
	}()
	return func() {
		scheduler.DefaultRegistry.Unregister(threadID, notifCh)
	}
}
