package tool

import (
	"context"
	"os"
	"testing"

	"database/sql"
	"github.com/cloudwego/eino/schema"
	_ "github.com/go-sql-driver/mysql"

	"deepAgent/internal/store"
	"deepAgent/internal/toolruntime"
)

func storeDBForTest(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRecordCheckinWritesRow(t *testing.T) {
	db := storeDBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	tid := "tool-test-" + "0002"

	in := checkinInput{
		ThreadID: tid,
		Category: "sport",
		Content:  "跑步5km",
		Value:    5.0,
	}
	out, err := recordCheckin(ctx, in, db)
	if err != nil {
		t.Fatalf("recordCheckin: %v", err)
	}
	if out.Message == "" {
		t.Fatal("empty output")
	}

	// 验证写入：query 一条 checkins
	var got string
	err = db.QueryRowContext(ctx,
		`SELECT content FROM checkins WHERE thread_id=? ORDER BY id DESC LIMIT 1`, tid).Scan(&got)
	if err != nil || got != "跑步5km" {
		t.Fatalf("expect 跑步5km, got %q err=%v", got, err)
	}
	_, _ = db.ExecContext(ctx, "DELETE FROM checkins WHERE thread_id=?", tid)
}

func TestQueryCheckinReturnsRows(t *testing.T) {
	db := storeDBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	tid := "tool-test-" + "0003"
	_, _ = recordCheckin(ctx, checkinInput{ThreadID: tid, Category: "study", Content: "读书1小时", Value: 1}, db)
	_, _ = recordCheckin(ctx, checkinInput{ThreadID: tid, Category: "study", Content: "做题2小时", Value: 2}, db)

	out, err := queryCheckin(ctx, queryCheckinInput{ThreadID: tid, Category: "study", Limit: 10}, db)
	if err != nil {
		t.Fatalf("queryCheckin: %v", err)
	}
	if len(out.Records) != 2 {
		t.Fatalf("expect 2 records, got %d: %v", len(out.Records), out)
	}
	_, _ = db.ExecContext(ctx, "DELETE FROM checkins WHERE thread_id=?", tid)
}

func TestRecordVisionModelUsage(t *testing.T) {
	db := storeDBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	runID := "vision-usage-run"
	threadID := "vision-usage-thread"
	userID := "vision-usage-user"
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM model_usage_logs WHERE run_id=?", runID)
	})

	ctx = toolruntime.WithAuditContext(ctx, toolruntime.AuditContext{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
	})
	recordVisionModelUsage(ctx, db, &schema.Message{
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     11,
				CompletionTokens: 7,
				TotalTokens:      18,
			},
		},
	})

	var agent string
	var total int
	if err := db.QueryRowContext(ctx,
		`SELECT agent, total_tokens FROM model_usage_logs WHERE run_id=?`,
		runID,
	).Scan(&agent, &total); err != nil {
		t.Fatal(err)
	}
	if agent != "vision" || total != 18 {
		t.Fatalf("agent=%q total=%d", agent, total)
	}
}
