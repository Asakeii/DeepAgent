package store

import (
	"context"
	"testing"
)

func TestAppendModelUsageWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "usage-user-" + randomSuffix()
	runID := "usage-run-" + randomSuffix()
	threadID := "usage-thread-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM model_usage_logs WHERE run_id=?", runID)
	})

	if err := AppendModelUsage(ctx, db, ModelUsageRecord{
		RunID:            runID,
		ThreadID:         threadID,
		UserID:           userID,
		Agent:            "researcher",
		Model:            "test-model",
		PromptTokens:     10,
		CompletionTokens: 20,
		CachedTokens:     3,
		ReasoningTokens:  4,
	}); err != nil {
		t.Fatal(err)
	}
	var total, cached, reasoning int
	if err := db.QueryRowContext(ctx,
		`SELECT total_tokens, cached_tokens, reasoning_tokens FROM model_usage_logs WHERE run_id=?`,
		runID,
	).Scan(&total, &cached, &reasoning); err != nil {
		t.Fatal(err)
	}
	if total != 30 || cached != 3 || reasoning != 4 {
		t.Fatalf("total=%d cached=%d reasoning=%d", total, cached, reasoning)
	}
}
