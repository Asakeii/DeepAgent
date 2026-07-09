package infra

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	ecmodel "github.com/cloudwego/eino/components/model"

	"deepAgent/internal/store"
)

func TestLoggerCallbackRecordsModelUsage(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Skip("mysql not available")
	}
	if err := store.RunMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	prevDB := DB
	DB = db
	t.Cleanup(func() {
		DB = prevDB
		_ = db.Close()
	})

	runID := "logger-usage-run-" + storeTestSuffix()
	threadID := "logger-usage-thread-" + storeTestSuffix()
	userID := "logger-usage-user-" + storeTestSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM model_usage_logs WHERE run_id=?", runID)
	})

	cb := &LoggerCallback{ID: threadID, RunID: runID, UserID: userID}
	cb.recordModelUsage("checkin", &ecmodel.CallbackOutput{
		Config: &ecmodel.Config{Model: "test-model"},
		TokenUsage: &ecmodel.TokenUsage{
			PromptTokens:     7,
			CompletionTokens: 8,
			TotalTokens:      15,
		},
	})

	var agent, modelName string
	var total int
	if err := db.QueryRowContext(context.Background(),
		`SELECT agent, model, total_tokens FROM model_usage_logs WHERE run_id=?`,
		runID,
	).Scan(&agent, &modelName, &total); err != nil {
		t.Fatal(err)
	}
	if agent != "checkin" || modelName != "test-model" || total != 15 {
		t.Fatalf("agent=%q model=%q total=%d", agent, modelName, total)
	}
}

func storeTestSuffix() string {
	return time.Now().Format("20060102150405.000000000")
}
