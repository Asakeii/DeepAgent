package tool

import (
	"context"
	"os"
	"testing"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
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
