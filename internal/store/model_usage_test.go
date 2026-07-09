package store

import (
	"context"
	"testing"
	"time"
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

func TestSumUserModelTokensSinceWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "usage-sum-user-" + randomSuffix()
	otherUserID := "usage-sum-other-" + randomSuffix()
	runID1 := "usage-sum-run-1-" + randomSuffix()
	runID2 := "usage-sum-run-2-" + randomSuffix()
	otherRunID := "usage-sum-run-other-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM model_usage_logs WHERE run_id IN (?, ?, ?)", runID1, runID2, otherRunID)
	})

	for _, record := range []ModelUsageRecord{
		{RunID: runID1, ThreadID: "thread-1", UserID: userID, TotalTokens: 40},
		{RunID: runID2, ThreadID: "thread-2", UserID: userID, PromptTokens: 20, CompletionTokens: 30},
		{RunID: otherRunID, ThreadID: "thread-3", UserID: otherUserID, TotalTokens: 1000},
	} {
		if err := AppendModelUsage(ctx, db, record); err != nil {
			t.Fatal(err)
		}
	}

	got, err := SumUserModelTokensSince(ctx, db, userID, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got != 90 {
		t.Fatalf("tokens=%d, want 90", got)
	}
}

func TestModelUsageCostMicros(t *testing.T) {
	got := ModelUsageCostMicros(1000, 500, 200, 10, ModelPrice{
		InputPerMillion:       2,
		OutputPerMillion:      8,
		CachedInputPerMillion: 1,
		ReasoningPerMillion:   12,
	})
	want := int64(5920)
	if got != want {
		t.Fatalf("cost=%d, want %d", got, want)
	}
}

func TestSumUserModelCostSinceWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "usage-cost-user-" + randomSuffix()
	runID := "usage-cost-run-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM model_usage_logs WHERE run_id=?", runID)
	})
	if err := AppendModelUsage(ctx, db, ModelUsageRecord{
		RunID:            runID,
		ThreadID:         "thread-cost",
		UserID:           userID,
		Model:            "priced-model",
		PromptTokens:     1000,
		CompletionTokens: 100,
	}); err != nil {
		t.Fatal(err)
	}
	summary, err := SumUserModelCostSince(ctx, db, userID, time.Now().Add(-time.Hour), map[string]ModelPrice{
		"priced-model": {InputPerMillion: 2, OutputPerMillion: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.CostMicros != 3000 || len(summary.UnknownModels) != 0 {
		t.Fatalf("summary=%+v, want cost 3000 no unknown", summary)
	}
}
