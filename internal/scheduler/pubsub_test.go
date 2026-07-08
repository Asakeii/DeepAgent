package scheduler

import (
	"encoding/json"
	"testing"
)

func TestPendingPayloadRoundTrip(t *testing.T) {
	event := ReminderEvent{
		ID:        "rem-1",
		ThreadID:  "thread-1",
		Message:   "drink water",
		FireAt:    123,
		Cron:      "0 9 * * *",
		Recurring: true,
		Status:    "fired",
	}
	payload, err := pendingPayload(event)
	if err != nil {
		t.Fatal(err)
	}
	var got ReminderEvent
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	if got != event {
		t.Fatalf("round trip got %+v, want %+v", got, event)
	}
}
