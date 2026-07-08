package app

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
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

func TestChatServiceRunTimeoutWithFakeResearchRunner(t *testing.T) {
	db := appDBForTest(t)
	if db == nil {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	prevDB := infra.DB
	prevConf := conf.App
	infra.DB = db
	conf.App = &conf.Config{Setting: conf.SettingConfig{RunTimeoutSeconds: 1}}
	t.Cleanup(func() {
		infra.DB = prevDB
		conf.App = prevConf
	})

	suffix := time.Now().Format("20060102150405.000000000")
	runID := "timeout-run-" + suffix
	threadID := "timeout-thread-" + suffix
	userID := "timeout-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})

	service := NewChatServiceWithDeps(blockingResearchRunner{}, &fakeCheckinRunner{}, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("执行一个会超时的研究任务"),
		},
	}, writer)

	if got := writer.FinalContent(); got != "运行超时，请缩小任务范围或稍后重试" {
		t.Fatalf("final content=%q, want timeout message", got)
	}
	run, err := store.GetRun(context.Background(), db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != store.RunStatusFailed {
		t.Fatalf("run status=%s, want %s", run.Status, store.RunStatusFailed)
	}
}

func TestChatServiceRejectsInvalidImageBeforeRunner(t *testing.T) {
	db := appDBForTest(t)
	if db == nil {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	prevDB := infra.DB
	prevConf := conf.App
	infra.DB = db
	conf.App = &conf.Config{
		Server: conf.ServerConfig{
			ImageMaxBytes:     1024,
			ImageAllowedTypes: []string{"image/png"},
		},
	}
	t.Cleanup(func() {
		infra.DB = prevDB
		conf.App = prevConf
	})

	suffix := time.Now().Format("20060102150405.000000000")
	runID := "bad-image-run-" + suffix
	threadID := "bad-image-thread-" + suffix
	userID := "bad-image-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})

	checkin := &fakeCheckinRunner{}
	service := NewChatServiceWithDeps(&fakeResearchRunner{}, checkin, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:       runID,
		ThreadID:    threadID,
		UserID:      userID,
		ImageBase64: "/etc/passwd",
		Messages: []*schema.Message{
			schema.UserMessage("分析这张图"),
		},
	}, writer)

	if checkin.imageCalled {
		t.Fatal("image runner should not be called for invalid image input")
	}
	if got := writer.FinalContent(); !strings.HasPrefix(got, "图片输入不符合安全要求") {
		t.Fatalf("final content=%q, want invalid image error", got)
	}
	run, err := store.GetRun(context.Background(), db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != store.RunStatusFailed {
		t.Fatalf("run status=%s, want %s", run.Status, store.RunStatusFailed)
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

type blockingResearchRunner struct{}

func (blockingResearchRunner) Run(ctx context.Context, req model.ChatRequest, writer EventWriter) (ResearchRunResult, error) {
	<-ctx.Done()
	return ResearchRunResult{}, ctx.Err()
}

type fakeCheckinRunner struct {
	called      bool
	imageCalled bool
	req         CheckinTurnRequest
}

func (r *fakeCheckinRunner) RunTurn(ctx context.Context, req CheckinTurnRequest) (CheckinTurnResult, error) {
	r.called = true
	r.req = req
	return CheckinTurnResult{Response: schema.AssistantMessage("fake checkin ok", nil)}, nil
}

func (r *fakeCheckinRunner) AnalyzeImage(ctx context.Context, req model.ChatRequest) (string, error) {
	r.imageCalled = true
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
