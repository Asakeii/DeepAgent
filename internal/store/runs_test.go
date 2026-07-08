package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRunEventLifecycle(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureRunTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	runID := "run-test-" + randomSuffix()
	threadID := "thread-test-" + randomSuffix()

	if err := CreateRun(ctx, db, RunRecord{
		ID:       runID,
		UserID:   "user-test",
		ThreadID: threadID,
		Mode:     "chat",
	}); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"content": "hello"})
	if err := AppendRunEvent(ctx, db, RunEventRecord{
		RunID:     runID,
		ThreadID:  threadID,
		UserID:    "user-test",
		EventName: "message",
		Agent:     "checkin",
		Payload:   payload,
	}); err != nil {
		t.Fatal(err)
	}
	events, err := ListRunEvents(ctx, db, runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventName != "message" || events[0].Agent != "checkin" {
		t.Fatalf("unexpected events: %+v", events)
	}
	if err := CompleteRun(ctx, db, runID, RunStatusSucceeded, ""); err != nil {
		t.Fatal(err)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM run_events WHERE run_id=?", runID)
	_, _ = db.ExecContext(ctx, "DELETE FROM runs WHERE id=?", runID)
}
