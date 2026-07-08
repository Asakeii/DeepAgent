package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const (
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
)

type RunRecord struct {
	ID       string
	UserID   string
	ThreadID string
	Mode     string
	Status   string
	Error    string
}

type RunEventRecord struct {
	ID        int64
	RunID     string
	ThreadID  string
	UserID    string
	EventName string
	Agent     string
	Payload   json.RawMessage
}

func EnsureRunTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS runs (
			id          VARCHAR(128) NOT NULL PRIMARY KEY,
			user_id     VARCHAR(128) NOT NULL DEFAULT '',
			thread_id   VARCHAR(128) NOT NULL,
			mode        VARCHAR(32) NOT NULL,
			status      VARCHAR(32) NOT NULL,
			started_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ended_at    TIMESTAMP NULL,
			error       TEXT,
			KEY idx_thread_started (thread_id, started_at),
			KEY idx_user_started (user_id, started_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS run_events (
			id          BIGINT AUTO_INCREMENT PRIMARY KEY,
			run_id      VARCHAR(128) NOT NULL,
			thread_id   VARCHAR(128) NOT NULL,
			user_id     VARCHAR(128) NOT NULL DEFAULT '',
			event_name  VARCHAR(64) NOT NULL,
			agent       VARCHAR(64) NOT NULL DEFAULT '',
			payload     JSON NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			KEY idx_run_id (run_id, id),
			KEY idx_thread_created (thread_id, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure run tables: %w", err)
		}
	}
	return nil
}

func CreateRun(ctx context.Context, db *sql.DB, run RunRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if run.ID == "" {
		return fmt.Errorf("run id is required")
	}
	if run.ThreadID == "" {
		return fmt.Errorf("thread id is required")
	}
	if run.Mode == "" {
		run.Mode = "chat"
	}
	if run.Status == "" {
		run.Status = RunStatusRunning
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO runs (id, user_id, thread_id, mode, status)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE status=VALUES(status), error=NULL, ended_at=NULL`,
		run.ID, run.UserID, run.ThreadID, run.Mode, run.Status,
	)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}
	return nil
}

func CompleteRun(ctx context.Context, db *sql.DB, runID, status, errorText string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if runID == "" {
		return fmt.Errorf("run id is required")
	}
	if status == "" {
		status = RunStatusSucceeded
	}
	_, err := db.ExecContext(ctx,
		`UPDATE runs SET status=?, error=?, ended_at=CURRENT_TIMESTAMP WHERE id=?`,
		status, nullableString(errorText), runID,
	)
	if err != nil {
		return fmt.Errorf("complete run: %w", err)
	}
	return nil
}

func AppendRunEvent(ctx context.Context, db *sql.DB, event RunEventRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if event.RunID == "" {
		return fmt.Errorf("run id is required")
	}
	if event.ThreadID == "" {
		return fmt.Errorf("thread id is required")
	}
	if event.EventName == "" {
		return fmt.Errorf("event name is required")
	}
	if len(event.Payload) == 0 {
		event.Payload = json.RawMessage(`{}`)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO run_events (run_id, thread_id, user_id, event_name, agent, payload)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.RunID, event.ThreadID, event.UserID, event.EventName, event.Agent, string(event.Payload),
	)
	if err != nil {
		return fmt.Errorf("append run event: %w", err)
	}
	return nil
}

func ListRunEvents(ctx context.Context, db *sql.DB, runID string, limit int) ([]RunEventRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, run_id, thread_id, user_id, event_name, agent, payload
		 FROM run_events WHERE run_id=? ORDER BY id ASC LIMIT ?`, runID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list run events: %w", err)
	}
	defer rows.Close()

	var out []RunEventRecord
	for rows.Next() {
		var event RunEventRecord
		if err := rows.Scan(&event.ID, &event.RunID, &event.ThreadID, &event.UserID, &event.EventName, &event.Agent, &event.Payload); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
