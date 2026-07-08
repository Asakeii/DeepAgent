package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPercentileInt64(t *testing.T) {
	values := []int64{10, 50, 20, 40, 30}
	if got := percentileInt64(values, 0.95); got != 50 {
		t.Fatalf("p95=%d want 50", got)
	}
	if got := averageInt64(values); got != 30 {
		t.Fatalf("avg=%d want 30", got)
	}
}

func TestGetRunMetricsWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "metrics-user-" + randomSuffix()
	threadID := "metrics-thread-" + randomSuffix()
	runOK := "metrics-run-ok-" + randomSuffix()
	runFailed := "metrics-run-failed-" + randomSuffix()
	args, _ := json.Marshal(map[string]any{"x": 1})

	if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, user_id, thread_id, mode, status, started_at, ended_at)
		VALUES (?, ?, ?, 'chat', 'succeeded', DATE_SUB(NOW(), INTERVAL 2 SECOND), NOW())`,
		runOK, userID, threadID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, user_id, thread_id, mode, status, started_at, ended_at, error)
		VALUES (?, ?, ?, 'chat', 'failed', DATE_SUB(NOW(), INTERVAL 1 SECOND), NOW(), 'boom')`,
		runFailed, userID, threadID); err != nil {
		t.Fatal(err)
	}
	if _, err := StartToolAudit(ctx, db, ToolAuditRecord{
		RunID:     runOK,
		ThreadID:  threadID,
		UserID:    userID,
		ToolName:  "web_search",
		Risk:      "external",
		Arguments: args,
	}); err != nil {
		t.Fatal(err)
	}
	toolID, err := StartToolAudit(ctx, db, ToolAuditRecord{
		RunID:     runFailed,
		ThreadID:  threadID,
		UserID:    userID,
		ToolName:  "record_checkin",
		Risk:      "write",
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := CompleteToolAudit(ctx, db, toolID, ToolStatusFailed, "", "boom", 25); err != nil {
		t.Fatal(err)
	}

	metrics, err := GetRunMetrics(ctx, db, userID, 24)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.RunsTotal != 2 || metrics.RunsSucceeded != 1 || metrics.RunsFailed != 1 {
		t.Fatalf("unexpected run metrics: %+v", metrics)
	}
	if metrics.RunSuccessRate != 0.5 {
		t.Fatalf("success rate=%v want 0.5", metrics.RunSuccessRate)
	}
	if metrics.ToolsTotal != 2 || metrics.ToolsFailed != 1 {
		t.Fatalf("unexpected tool metrics: %+v", metrics)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM tool_audit_logs WHERE user_id=?", userID)
	_, _ = db.ExecContext(ctx, "DELETE FROM runs WHERE user_id=?", userID)
}
