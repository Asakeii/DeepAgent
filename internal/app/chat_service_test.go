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
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
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
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
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
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
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

func TestChatServicePersistsResearchArtifact(t *testing.T) {
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
	runID := "artifact-run-" + suffix
	threadID := "artifact-thread-" + suffix
	userID := "artifact-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})

	service := NewChatServiceWithDeps(&fakeResearchRunner{result: ResearchRunResult{Final: "# 最终报告\n\n参考 [资料](https://example.com/report)"}}, &fakeCheckinRunner{}, fakeReminderStreamer{})
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("研究成熟 Agent 项目"),
		},
	}, NewCaptureWriter())

	records, err := store.ListArtifacts(context.Background(), db, userID, threadID, store.ArtifactKindReport, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("len=%d, want 1", len(records))
	}
	if records[0].RunID != runID || records[0].Content == "" {
		t.Fatalf("unexpected artifact: %+v", records[0])
	}
	citations, err := store.ListArtifactCitations(context.Background(), db, userID, records[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(citations) != 1 || citations[0].URL != "https://example.com/report" {
		t.Fatalf("unexpected citations: %+v", citations)
	}
}

func TestChatServiceAppliesUserSettingsDefaults(t *testing.T) {
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
	runID := "settings-run-" + suffix
	threadID := "settings-thread-" + suffix
	userID := "settings-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM user_settings WHERE user_id=?", userID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})
	enableBackground := true
	if err := store.UpsertUserSettings(context.Background(), db, store.UserSettingsRecord{
		UserID:                        userID,
		Locale:                        "en-US",
		MaxPlanIterations:             sql.NullInt64{Int64: 4, Valid: true},
		MaxStepNum:                    sql.NullInt64{Int64: 6, Valid: true},
		EnableBackgroundInvestigation: sql.NullBool{Bool: enableBackground, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	research := &fakeResearchRunner{result: ResearchRunResult{Final: "ok"}}
	service := NewChatServiceWithDeps(research, &fakeCheckinRunner{}, fakeReminderStreamer{})
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("研究用户设置默认值"),
		},
	}, NewCaptureWriter())

	if !research.called {
		t.Fatal("research runner was not called")
	}
	if research.req.Locale != "en-US" {
		t.Fatalf("locale=%q, want en-US", research.req.Locale)
	}
	if research.req.MaxPlanIterations != 4 || research.req.MaxStepNum != 6 {
		t.Fatalf("unexpected limits: maxPlan=%d maxStep=%d", research.req.MaxPlanIterations, research.req.MaxStepNum)
	}
	if research.req.EnableBackgroundInvestigation == nil || !*research.req.EnableBackgroundInvestigation {
		t.Fatalf("background setting not applied: %+v", research.req.EnableBackgroundInvestigation)
	}
}

func TestChatServiceRejectsRunWhenDailyTokenBudgetExceeded(t *testing.T) {
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
	runID := "budget-run-" + suffix
	threadID := "budget-thread-" + suffix
	userID := "budget-user-" + suffix
	usageRunID := "budget-usage-run-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM model_usage_logs WHERE user_id=?", userID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM user_settings WHERE user_id=?", userID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})
	if err := store.UpsertUserSettings(context.Background(), db, store.UserSettingsRecord{
		UserID:           userID,
		DailyTokenBudget: sql.NullInt64{Int64: 1000, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendModelUsage(context.Background(), db, store.ModelUsageRecord{
		RunID:       usageRunID,
		ThreadID:    threadID,
		UserID:      userID,
		TotalTokens: 1000,
	}); err != nil {
		t.Fatal(err)
	}

	research := &fakeResearchRunner{result: ResearchRunResult{Final: "should not run"}}
	service := NewChatServiceWithDeps(research, &fakeCheckinRunner{}, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("研究一个会触发预算拦截的任务"),
		},
	}, writer)

	if research.called {
		t.Fatal("research runner should not be called after token budget is exceeded")
	}
	if got := writer.FinalContent(); !strings.Contains(got, "今日模型 token 用量已达到预算") {
		t.Fatalf("final content=%q, want token budget error", got)
	}
	run, err := store.GetRun(context.Background(), db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != store.RunStatusFailed {
		t.Fatalf("run status=%s, want %s", run.Status, store.RunStatusFailed)
	}
}

func TestChatServiceRejectsRunWhenDailyCostBudgetExceeded(t *testing.T) {
	db := appDBForTest(t)
	if db == nil {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	prevDB := infra.DB
	prevConf := conf.App
	infra.DB = db
	conf.App = &conf.Config{Model: conf.ModelConfig{
		Prices: map[string]conf.ModelPriceConfig{
			"priced-model": {InputPerMillion: 2, OutputPerMillion: 10},
		},
	}}
	t.Cleanup(func() {
		infra.DB = prevDB
		conf.App = prevConf
	})

	suffix := time.Now().Format("20060102150405.000000000")
	runID := "cost-budget-run-" + suffix
	threadID := "cost-budget-thread-" + suffix
	userID := "cost-budget-user-" + suffix
	usageRunID := "cost-budget-usage-run-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM model_usage_logs WHERE user_id=?", userID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM user_settings WHERE user_id=?", userID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})
	if err := store.UpsertUserSettings(context.Background(), db, store.UserSettingsRecord{
		UserID:                userID,
		DailyCostBudgetMicros: sql.NullInt64{Int64: 3000, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendModelUsage(context.Background(), db, store.ModelUsageRecord{
		RunID:            usageRunID,
		ThreadID:         threadID,
		UserID:           userID,
		Model:            "priced-model",
		PromptTokens:     1000,
		CompletionTokens: 100,
	}); err != nil {
		t.Fatal(err)
	}

	research := &fakeResearchRunner{result: ResearchRunResult{Final: "should not run"}}
	service := NewChatServiceWithDeps(research, &fakeCheckinRunner{}, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		Messages: []*schema.Message{
			schema.UserMessage("研究一个会触发金额预算拦截的任务"),
		},
	}, writer)

	if research.called {
		t.Fatal("research runner should not be called after cost budget is exceeded")
	}
	if got := writer.FinalContent(); !strings.Contains(got, "今日模型金额用量已达到预算") {
		t.Fatalf("final content=%q, want cost budget error", got)
	}
	run, err := store.GetRun(context.Background(), db, runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != store.RunStatusFailed {
		t.Fatalf("run status=%s, want %s", run.Status, store.RunStatusFailed)
	}
}

func TestChatServiceRejectsRunWhenTeamDailyCostBudgetExceeded(t *testing.T) {
	db := appDBForTest(t)
	if db == nil {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	prevDB := infra.DB
	prevConf := conf.App
	infra.DB = db
	conf.App = &conf.Config{Model: conf.ModelConfig{
		Prices: map[string]conf.ModelPriceConfig{
			"team-priced-model": {InputPerMillion: 1, OutputPerMillion: 4},
		},
	}}
	t.Cleanup(func() {
		infra.DB = prevDB
		conf.App = prevConf
	})

	suffix := time.Now().Format("20060102150405.000000000")
	runID := "team-cost-budget-run-" + suffix
	threadID := "team-cost-budget-thread-" + suffix
	userID := "team-cost-budget-user-" + suffix
	usageRunID := "team-cost-budget-usage-run-" + suffix
	team, err := store.CreateTeam(context.Background(), db, userID, "Cost Team")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM model_usage_logs WHERE run_id=?", usageRunID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM team_settings WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM team_members WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM teams WHERE id=?", team.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})
	if err := store.EnsureThreadWithTeam(context.Background(), db, threadID, userID, team.ID, "team cost", "test"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTeamSettings(context.Background(), db, store.TeamSettingsRecord{
		TeamID:                team.ID,
		DailyCostBudgetMicros: sql.NullInt64{Int64: 1400, Valid: true},
		UpdatedBy:             userID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendModelUsage(context.Background(), db, store.ModelUsageRecord{
		RunID:            usageRunID,
		ThreadID:         threadID,
		UserID:           userID,
		Model:            "team-priced-model",
		PromptTokens:     1000,
		CompletionTokens: 100,
	}); err != nil {
		t.Fatal(err)
	}

	research := &fakeResearchRunner{result: ResearchRunResult{Final: "should not run"}}
	service := NewChatServiceWithDeps(research, &fakeCheckinRunner{}, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:    runID,
		ThreadID: threadID,
		UserID:   userID,
		TeamID:   team.ID,
		Messages: []*schema.Message{
			schema.UserMessage("研究一个会触发团队金额预算拦截的任务"),
		},
	}, writer)

	if research.called {
		t.Fatal("research runner should not be called after team cost budget is exceeded")
	}
	if got := writer.FinalContent(); !strings.Contains(got, "今日模型金额用量已达到预算") {
		t.Fatalf("final content=%q, want cost budget error", got)
	}
}

func TestChatServiceRejectsUnknownModelProfileBeforeRunner(t *testing.T) {
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
	runID := "model-profile-run-" + suffix
	threadID := "model-profile-thread-" + suffix
	userID := "model-profile-user-" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM artifacts WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM run_events WHERE run_id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM runs WHERE id=?", runID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM memories WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM users WHERE id=?", userID)
	})

	research := &fakeResearchRunner{result: ResearchRunResult{Final: "should not run"}}
	service := NewChatServiceWithDeps(research, &fakeCheckinRunner{}, fakeReminderStreamer{})
	writer := NewCaptureWriter()
	service.RunStream(context.Background(), model.ChatRequest{
		RunID:        runID,
		ThreadID:     threadID,
		UserID:       userID,
		ModelProfile: "missing-profile",
		Messages: []*schema.Message{
			schema.UserMessage("研究模型路由"),
		},
	}, writer)

	if research.called {
		t.Fatal("research runner should not be called for unknown model profile")
	}
	if got := writer.FinalContent(); !strings.Contains(got, "未知模型配置") {
		t.Fatalf("final content=%q, want model profile error", got)
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
	req    model.ChatRequest
	result ResearchRunResult
	err    error
}

func (r *fakeResearchRunner) Run(ctx context.Context, req model.ChatRequest, writer EventWriter) (ResearchRunResult, error) {
	r.called = true
	r.req = req
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
