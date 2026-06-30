package scheduler

import "context"

// ReminderEvent is the structured payload sent to the frontend for reminder cards.
type ReminderEvent struct {
	ID        string `json:"id"`
	ThreadID  string `json:"thread_id"`
	Message   string `json:"message"`
	FireAt    int64  `json:"fire_at"`
	Cron      string `json:"cron,omitempty"`
	Recurring bool   `json:"recurring"`
	Status    string `json:"status"`
}

type eventSinkKey struct{}

type EventSink func(ReminderEvent)

// WithEventSink lets tool calls report structured reminder events to the caller.
func WithEventSink(ctx context.Context, sink EventSink) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, eventSinkKey{}, sink)
}

func EmitEvent(ctx context.Context, event ReminderEvent) {
	if sink, ok := ctx.Value(eventSinkKey{}).(EventSink); ok && sink != nil {
		sink(event)
	}
}

func eventFromReminder(r Reminder, status string) ReminderEvent {
	return ReminderEvent{
		ID:        r.ID,
		ThreadID:  r.ThreadID,
		Message:   r.Message,
		FireAt:    r.FireAt,
		Cron:      r.Cron,
		Recurring: r.Recurring,
		Status:    status,
	}
}
