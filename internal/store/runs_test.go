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

func TestRunCancellationLifecycle(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureRunTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	runID := "run-cancel-" + randomSuffix()
	threadID := "thread-cancel-" + randomSuffix()
	userID := "user-cancel-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(ctx, "DELETE FROM runs WHERE id=?", runID)
	})

	if err := CreateRun(ctx, db, RunRecord{
		ID:       runID,
		UserID:   userID,
		ThreadID: threadID,
		Mode:     "chat",
	}); err != nil {
		t.Fatal(err)
	}
	if cancelled, err := CancelRun(ctx, db, runID, "other-user"); err != nil || cancelled {
		t.Fatalf("other user cancelled=%v err=%v, want false nil", cancelled, err)
	}
	cancelled, err := CancelRun(ctx, db, runID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !cancelled {
		t.Fatal("cancelled=false, want true")
	}
	isCancelled, err := IsRunCancelled(ctx, db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if !isCancelled {
		t.Fatal("run was not marked cancelled")
	}
	if err := CompleteRun(ctx, db, runID, RunStatusSucceeded, ""); err != nil {
		t.Fatal(err)
	}
	run, err := GetRun(ctx, db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunStatusCancelled {
		t.Fatalf("status=%s, want %s", run.Status, RunStatusCancelled)
	}
	if !run.CancelRequestedAt.Valid {
		t.Fatal("cancel_requested_at is not set")
	}
	if err := AppendRunCancelledEvent(ctx, db, run); err != nil {
		t.Fatal(err)
	}
	events, err := ListRunEvents(ctx, db, runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventName != "run_cancelled" {
		t.Fatalf("unexpected cancellation events: %+v", events)
	}
}
