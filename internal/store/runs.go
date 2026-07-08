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
	RunStatusCancelled = "cancelled"
)

type RunRecord struct {
	ID                string
	UserID            string
	ThreadID          string
	Mode              string
	Status            string
	Error             string
	CancelRequestedAt sql.NullTime
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
			cancel_requested_at TIMESTAMP NULL,
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
	if err := ensureColumn(ctx, db, "runs", "cancel_requested_at", "TIMESTAMP NULL"); err != nil {
		return fmt.Errorf("ensure run cancel column: %w", err)
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
		 ON DUPLICATE KEY UPDATE status=VALUES(status), error=NULL, ended_at=NULL, cancel_requested_at=NULL`,
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
		`UPDATE runs
		 SET status=?, error=?, ended_at=CURRENT_TIMESTAMP
		 WHERE id=? AND status<>?`,
		status, nullableString(errorText), runID, RunStatusCancelled,
	)
	if err != nil {
		return fmt.Errorf("complete run: %w", err)
	}
	return nil
}

func CancelRun(ctx context.Context, db *sql.DB, runID, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	if runID == "" {
		return false, fmt.Errorf("run id is required")
	}
	userID = NormalizeUserID(userID)
	res, err := db.ExecContext(ctx,
		`UPDATE runs
		 SET status=?, error=?, ended_at=CURRENT_TIMESTAMP, cancel_requested_at=CURRENT_TIMESTAMP
		 WHERE id=? AND user_id=? AND status=?`,
		RunStatusCancelled, "cancelled by user", runID, userID, RunStatusRunning,
	)
	if err != nil {
		return false, fmt.Errorf("cancel run: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("cancel run rows affected: %w", err)
	}
	return n > 0, nil
}

func IsRunCancelled(ctx context.Context, db *sql.DB, runID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	if runID == "" {
		return false, fmt.Errorf("run id is required")
	}
	var status string
	err := db.QueryRowContext(ctx, `SELECT status FROM runs WHERE id=?`, runID).Scan(&status)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query run status: %w", err)
	}
	return status == RunStatusCancelled, nil
}

func GetRun(ctx context.Context, db *sql.DB, runID string) (RunRecord, error) {
	if db == nil {
		return RunRecord{}, fmt.Errorf("db is nil")
	}
	var run RunRecord
	err := db.QueryRowContext(ctx,
		`SELECT id, user_id, thread_id, mode, status, COALESCE(error, ''), cancel_requested_at
		 FROM runs WHERE id=?`,
		runID,
	).Scan(&run.ID, &run.UserID, &run.ThreadID, &run.Mode, &run.Status, &run.Error, &run.CancelRequestedAt)
	if err == sql.ErrNoRows {
		return RunRecord{}, nil
	}
	if err != nil {
		return RunRecord{}, fmt.Errorf("get run: %w", err)
	}
	return run, nil
}

func AppendRunCancelledEvent(ctx context.Context, db *sql.DB, run RunRecord) error {
	if run.ID == "" {
		return fmt.Errorf("run id is required")
	}
	payload, _ := json.Marshal(map[string]any{
		"run_id":    run.ID,
		"thread_id": run.ThreadID,
		"status":    RunStatusCancelled,
	})
	return AppendRunEvent(ctx, db, RunEventRecord{
		RunID:     run.ID,
		ThreadID:  run.ThreadID,
		UserID:    run.UserID,
		EventName: "run_cancelled",
		Payload:   payload,
	})
}

func ensureColumn(ctx context.Context, db *sql.DB, tableName, columnName, definition string) error {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM information_schema.columns
		 WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`,
		tableName, columnName,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition))
	return err
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
