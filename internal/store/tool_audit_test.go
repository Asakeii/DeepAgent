package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestToolAuditLifecycle(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureToolAuditTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	runID := "tool-run-" + randomSuffix()
	args, _ := json.Marshal(map[string]any{"query": "hello"})

	id, err := StartToolAudit(ctx, db, ToolAuditRecord{
		RunID:     runID,
		ThreadID:  "thread-tool",
		UserID:    "user-tool",
		ToolName:  "web_search",
		Risk:      "safe",
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := CompleteToolAudit(ctx, db, id, ToolStatusSucceeded, "ok", "", 12); err != nil {
		t.Fatal(err)
	}
	records, err := ListToolAuditsByRun(ctx, db, runID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Status != ToolStatusSucceeded || records[0].DurationMS != 12 {
		t.Fatalf("unexpected audits: %+v", records)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM tool_audit_logs WHERE run_id=?", runID)
}
