package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"deepAgent/internal/security"
)

const (
	ToolStatusStarted   = "started"
	ToolStatusSucceeded = "succeeded"
	ToolStatusFailed    = "failed"
	ToolStatusBlocked   = "blocked"
)

type ToolAuditRecord struct {
	ID         int64
	RunID      string
	ThreadID   string
	UserID     string
	ToolName   string
	Risk       string
	Status     string
	Arguments  json.RawMessage
	Result     string
	Error      string
	DurationMS int64
}

func EnsureToolAuditTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS tool_audit_logs (
		id            BIGINT AUTO_INCREMENT PRIMARY KEY,
		run_id        VARCHAR(128) NOT NULL DEFAULT '',
		thread_id     VARCHAR(128) NOT NULL DEFAULT '',
		user_id       VARCHAR(128) NOT NULL DEFAULT '',
		tool_name     VARCHAR(128) NOT NULL,
		risk          VARCHAR(32) NOT NULL,
		status        VARCHAR(32) NOT NULL,
		arguments     JSON,
		result        MEDIUMTEXT,
		error         TEXT,
		duration_ms   BIGINT NOT NULL DEFAULT 0,
		created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at  TIMESTAMP NULL,
		KEY idx_run_tool (run_id, id),
		KEY idx_thread_created (thread_id, created_at),
		KEY idx_tool_status (tool_name, status)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure tool audit tables: %w", err)
	}
	return nil
}

func StartToolAudit(ctx context.Context, db *sql.DB, record ToolAuditRecord) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	if record.ToolName == "" {
		return 0, fmt.Errorf("tool name is required")
	}
	if record.Status == "" {
		record.Status = ToolStatusStarted
	}
	if len(record.Arguments) == 0 {
		record.Arguments = json.RawMessage(`{}`)
	}
	record.Arguments = security.RedactJSON(record.Arguments)
	res, err := db.ExecContext(ctx,
		`INSERT INTO tool_audit_logs (run_id, thread_id, user_id, tool_name, risk, status, arguments)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.RunID, record.ThreadID, record.UserID, record.ToolName, record.Risk, record.Status, string(record.Arguments),
	)
	if err != nil {
		return 0, fmt.Errorf("start tool audit: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("tool audit id: %w", err)
	}
	return id, nil
}

func CompleteToolAudit(ctx context.Context, db *sql.DB, id int64, status, result, errorText string, durationMS int64) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if id <= 0 {
		return nil
	}
	if status == "" {
		status = ToolStatusSucceeded
	}
	_, err := db.ExecContext(ctx,
		`UPDATE tool_audit_logs
		 SET status=?, result=?, error=?, duration_ms=?, completed_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		status, trimAuditText(security.RedactString(result)), nullableString(security.RedactString(errorText)), durationMS, id,
	)
	if err != nil {
		return fmt.Errorf("complete tool audit: %w", err)
	}
	return nil
}

func ListToolAuditsByRun(ctx context.Context, db *sql.DB, runID string, limit int) ([]ToolAuditRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, run_id, thread_id, user_id, tool_name, risk, status, COALESCE(arguments, JSON_OBJECT()), COALESCE(result, ''), COALESCE(error, ''), duration_ms
		 FROM tool_audit_logs WHERE run_id=? ORDER BY id ASC LIMIT ?`, runID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tool audits: %w", err)
	}
	defer rows.Close()

	var out []ToolAuditRecord
	for rows.Next() {
		var record ToolAuditRecord
		if err := rows.Scan(&record.ID, &record.RunID, &record.ThreadID, &record.UserID, &record.ToolName, &record.Risk, &record.Status, &record.Arguments, &record.Result, &record.Error, &record.DurationMS); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func trimAuditText(v string) string {
	const max = 8192
	if len(v) <= max {
		return v
	}
	return v[:max] + "...[truncated]"
}
