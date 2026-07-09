package store

import (
	"context"
	"database/sql"
	"fmt"
)

type ModelUsageRecord struct {
	ID               int64
	RunID            string
	ThreadID         string
	UserID           string
	Agent            string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CachedTokens     int
	ReasoningTokens  int
}

func EnsureModelUsageTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS model_usage_logs (
		id                BIGINT AUTO_INCREMENT PRIMARY KEY,
		run_id            VARCHAR(128) NOT NULL DEFAULT '',
		thread_id         VARCHAR(128) NOT NULL DEFAULT '',
		user_id           VARCHAR(128) NOT NULL DEFAULT '',
		agent             VARCHAR(64) NOT NULL DEFAULT '',
		model             VARCHAR(128) NOT NULL DEFAULT '',
		prompt_tokens     INT NOT NULL DEFAULT 0,
		completion_tokens INT NOT NULL DEFAULT 0,
		total_tokens      INT NOT NULL DEFAULT 0,
		cached_tokens     INT NOT NULL DEFAULT 0,
		reasoning_tokens  INT NOT NULL DEFAULT 0,
		created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		KEY idx_run_model (run_id, id),
		KEY idx_user_created (user_id, created_at),
		KEY idx_agent_created (agent, created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure model usage table: %w", err)
	}
	return nil
}

func AppendModelUsage(ctx context.Context, db *sql.DB, record ModelUsageRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if record.RunID == "" {
		return fmt.Errorf("run id is required")
	}
	if record.ThreadID == "" {
		return fmt.Errorf("thread id is required")
	}
	if record.UserID == "" {
		record.UserID = AnonymousUserID
	}
	if record.TotalTokens <= 0 && record.PromptTokens+record.CompletionTokens > 0 {
		record.TotalTokens = record.PromptTokens + record.CompletionTokens
	}
	if record.TotalTokens <= 0 {
		return nil
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO model_usage_logs
		 (run_id, thread_id, user_id, agent, model, prompt_tokens, completion_tokens, total_tokens, cached_tokens, reasoning_tokens)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.ThreadID, record.UserID, record.Agent, record.Model, record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.CachedTokens, record.ReasoningTokens,
	)
	if err != nil {
		return fmt.Errorf("append model usage: %w", err)
	}
	return nil
}
