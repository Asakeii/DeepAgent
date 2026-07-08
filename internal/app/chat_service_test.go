package app

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func TestChatServiceRoutesToFakeCheckinRunner(t *testing.T) {
	db := appDBForTest(t)
	if db == nil {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	prevDB := infra.DB
	infra.DB = db
	t.Cleanup(func() {
		infra.DB = prevDB
	})

	suffix := time.Now().Format("20060102150405.000000000")
	runID := "fake-run-" + suffix
	threadID := "fake-thread-" + suffix
	userID := "fake-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})

	research := &fakeResearchRunner{result: ResearchRunResult{RouteToCheckin: true}}
	checkin := &fakeCheckinRunner{}
	service := NewChatServiceWithDeps(research, checkin, fakeReminderStreamer{})
	writer := NewCaptureWriter()

	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("帮我记录今天跑步"),
		},
	}, writer)

	if !research.called {
		t.Fatal("research runner was not called")
	}
	if !checkin.called {
		t.Fatal("checkin runner was not called")
	}
	if checkin.req.RunID != runID || checkin.req.ThreadID != threadID || checkin.req.UserID != userID {
		t.Fatalf("unexpected checkin request: %+v", checkin.req)
	}
	if got := writer.FinalContent(); got != "fake checkin ok" {
		t.Fatalf("final content=%q, want fake checkin ok", got)
	}
}

func appDBForTest(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.RunMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type fakeResearchRunner struct {
	called bool
	result ResearchRunResult
	err    error
}

func (r *fakeResearchRunner) Run(ctx context.Context, req model.ChatRequest, writer EventWriter) (ResearchRunResult, error) {
	r.called = true
	return r.result, r.err
}

type fakeCheckinRunner struct {
	called bool
	req    CheckinTurnRequest
}

func (r *fakeCheckinRunner) RunTurn(ctx context.Context, req CheckinTurnRequest) (CheckinTurnResult, error) {
	r.called = true
	r.req = req
	return CheckinTurnResult{Response: schema.AssistantMessage("fake checkin ok", nil)}, nil
}

func (r *fakeCheckinRunner) AnalyzeImage(ctx context.Context, req model.ChatRequest) (string, error) {
	return "fake image ok", nil
}

func (r *fakeCheckinRunner) EmitResult(writer EventWriter, threadID string, result CheckinTurnResult) {
	content := ""
	if result.Response != nil {
		content = result.Response.Content
	}
	_ = writer.WriteEvent("message", &model.ChatResp{Agent: "checkin", Role: "assistant", Content: content})
}

type fakeReminderStreamer struct{}

func (fakeReminderStreamer) AttachStream(ctx context.Context, threadID string, writer EventWriter) func() {
	return func() {}
}
