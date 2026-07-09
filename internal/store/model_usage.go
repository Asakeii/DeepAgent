package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
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

type ModelPrice struct {
	InputPerMillion       float64
	OutputPerMillion      float64
	CachedInputPerMillion float64
	ReasoningPerMillion   float64
}

type ModelCostSummary struct {
	CostMicros    int64
	UnknownModels []string
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

func SumUserModelTokensSince(ctx context.Context, db *sql.DB, userID string, since time.Time) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	var total int64
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(total_tokens), 0)
		 FROM model_usage_logs
		 WHERE user_id=? AND created_at>=?`,
		userID, since,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum user model tokens: %w", err)
	}
	return total, nil
}

func SumUserModelCostSince(ctx context.Context, db *sql.DB, userID string, since time.Time, prices map[string]ModelPrice) (ModelCostSummary, error) {
	if db == nil {
		return ModelCostSummary{}, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	rows, err := db.QueryContext(ctx,
		`SELECT model,
		 COALESCE(SUM(prompt_tokens), 0),
		 COALESCE(SUM(completion_tokens), 0),
		 COALESCE(SUM(cached_tokens), 0),
		 COALESCE(SUM(reasoning_tokens), 0)
		 FROM model_usage_logs
		 WHERE user_id=? AND created_at>=?
		 GROUP BY model`,
		userID, since,
	)
	if err != nil {
		return ModelCostSummary{}, fmt.Errorf("sum user model cost: %w", err)
	}
	defer rows.Close()
	return scanModelCostRows(rows, prices)
}

func SumTeamModelCostSince(ctx context.Context, db *sql.DB, teamID string, since time.Time, prices map[string]ModelPrice) (ModelCostSummary, error) {
	if db == nil {
		return ModelCostSummary{}, fmt.Errorf("db is nil")
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return ModelCostSummary{}, fmt.Errorf("%w: team id is required", ErrValidation)
	}
	rows, err := db.QueryContext(ctx,
		`SELECT m.model,
		 COALESCE(SUM(m.prompt_tokens), 0),
		 COALESCE(SUM(m.completion_tokens), 0),
		 COALESCE(SUM(m.cached_tokens), 0),
		 COALESCE(SUM(m.reasoning_tokens), 0)
		 FROM model_usage_logs m
		 JOIN threads t ON t.id=m.thread_id
		 WHERE t.team_id=? AND m.created_at>=?
		 GROUP BY m.model`,
		teamID, since,
	)
	if err != nil {
		return ModelCostSummary{}, fmt.Errorf("sum team model cost: %w", err)
	}
	defer rows.Close()
	return scanModelCostRows(rows, prices)
}

func scanModelCostRows(rows *sql.Rows, prices map[string]ModelPrice) (ModelCostSummary, error) {
	out := ModelCostSummary{}
	unknown := map[string]bool{}
	for rows.Next() {
		var model string
		var promptTokens, completionTokens, cachedTokens, reasoningTokens int64
		if err := rows.Scan(&model, &promptTokens, &completionTokens, &cachedTokens, &reasoningTokens); err != nil {
			return out, err
		}
		price, ok := prices[model]
		if !ok {
			unknown[model] = true
			continue
		}
		out.CostMicros += ModelUsageCostMicros(promptTokens, completionTokens, cachedTokens, reasoningTokens, price)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	for model := range unknown {
		out.UnknownModels = append(out.UnknownModels, model)
	}
	sort.Strings(out.UnknownModels)
	return out, nil
}

func ModelUsageCostMicros(promptTokens, completionTokens, cachedTokens, reasoningTokens int64, price ModelPrice) int64 {
	if cachedTokens < 0 {
		cachedTokens = 0
	}
	if promptTokens < 0 {
		promptTokens = 0
	}
	if cachedTokens > promptTokens {
		cachedTokens = promptTokens
	}
	uncachedPrompt := promptTokens - cachedTokens
	cachedPrice := price.CachedInputPerMillion
	if cachedPrice <= 0 {
		cachedPrice = price.InputPerMillion
	}
	total := float64(uncachedPrompt)*price.InputPerMillion +
		float64(cachedTokens)*cachedPrice +
		float64(completionTokens)*price.OutputPerMillion +
		float64(reasoningTokens)*price.ReasoningPerMillion
	if total <= 0 {
		return 0
	}
	return int64(math.Round(total))
}
