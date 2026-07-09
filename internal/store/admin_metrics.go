package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AdminOverview struct {
	WindowHours      int
	UsersTotal       int
	ThreadsTotal     int
	ArtifactsTotal   int
	ArtifactShares   int
	RunsTotal        int
	RunsSucceeded    int
	RunsFailed       int
	RunsRunning      int
	RunSuccessRate   float64
	ToolsTotal       int
	ToolsFailed      int
	ToolsBlocked     int
	ToolErrorRate    float64
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CachedTokens     int64
	ReasoningTokens  int64
}

func GetAdminOverview(ctx context.Context, db *sql.DB, windowHours int) (AdminOverview, error) {
	if db == nil {
		return AdminOverview{}, fmt.Errorf("db is nil")
	}
	if windowHours <= 0 {
		windowHours = 24
	}
	if windowHours > 24*30 {
		windowHours = 24 * 30
	}
	since := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	out := AdminOverview{WindowHours: windowHours}
	if err := scanCount(ctx, db, `SELECT COUNT(*) FROM users`, &out.UsersTotal); err != nil {
		return out, err
	}
	if err := scanCount(ctx, db, `SELECT COUNT(*) FROM threads`, &out.ThreadsTotal); err != nil {
		return out, err
	}
	if err := scanCount(ctx, db, `SELECT COUNT(*) FROM artifacts`, &out.ArtifactsTotal); err != nil {
		return out, err
	}
	if err := scanCount(ctx, db, `SELECT COUNT(*) FROM artifact_shares WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at>CURRENT_TIMESTAMP)`, &out.ArtifactShares); err != nil {
		return out, err
	}
	if err := fillAdminRunMetrics(ctx, db, since, &out); err != nil {
		return out, err
	}
	if err := fillAdminToolMetrics(ctx, db, since, &out); err != nil {
		return out, err
	}
	if err := fillAdminModelUsageMetrics(ctx, db, since, &out); err != nil {
		return out, err
	}
	return out, nil
}

func scanCount(ctx context.Context, db *sql.DB, query string, out *int) error {
	if err := db.QueryRowContext(ctx, query).Scan(out); err != nil {
		return fmt.Errorf("admin count query: %w", err)
	}
	return nil
}

func fillAdminRunMetrics(ctx context.Context, db *sql.DB, since time.Time, out *AdminOverview) error {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(*) FROM runs WHERE started_at>=? GROUP BY status`, since)
	if err != nil {
		return fmt.Errorf("query admin run metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		out.RunsTotal += count
		switch status {
		case RunStatusSucceeded:
			out.RunsSucceeded = count
		case RunStatusFailed:
			out.RunsFailed = count
		case RunStatusRunning:
			out.RunsRunning = count
		}
	}
	if out.RunsTotal > 0 {
		out.RunSuccessRate = float64(out.RunsSucceeded) / float64(out.RunsTotal)
	}
	return rows.Err()
}

func fillAdminToolMetrics(ctx context.Context, db *sql.DB, since time.Time, out *AdminOverview) error {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tool_audit_logs WHERE created_at>=? GROUP BY status`, since)
	if err != nil {
		return fmt.Errorf("query admin tool metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		out.ToolsTotal += count
		switch status {
		case ToolStatusFailed:
			out.ToolsFailed = count
		case ToolStatusBlocked:
			out.ToolsBlocked = count
		}
	}
	if out.ToolsTotal > 0 {
		out.ToolErrorRate = float64(out.ToolsFailed+out.ToolsBlocked) / float64(out.ToolsTotal)
	}
	return rows.Err()
}

func fillAdminModelUsageMetrics(ctx context.Context, db *sql.DB, since time.Time, out *AdminOverview) error {
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(prompt_tokens), 0),
		 COALESCE(SUM(completion_tokens), 0),
		 COALESCE(SUM(total_tokens), 0),
		 COALESCE(SUM(cached_tokens), 0),
		 COALESCE(SUM(reasoning_tokens), 0)
		 FROM model_usage_logs
		 WHERE created_at>=?`,
		since,
	).Scan(&out.PromptTokens, &out.CompletionTokens, &out.TotalTokens, &out.CachedTokens, &out.ReasoningTokens)
	if err != nil {
		return fmt.Errorf("query admin model usage metrics: %w", err)
	}
	return nil
}
