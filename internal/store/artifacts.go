package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ArtifactKindReport   = "report"
	ArtifactFormatMD     = "markdown"
	ArtifactSourceAgent  = "agent"
	defaultArtifactLimit = 50
)

type ArtifactRecord struct {
	ID        int64
	UserID    string
	ThreadID  string
	RunID     string
	Kind      string
	Title     string
	Format    string
	Content   string
	Metadata  json.RawMessage
	Version   int64
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func EnsureArtifactTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS artifacts (
		id          BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id     VARCHAR(128) NOT NULL DEFAULT '',
		thread_id   VARCHAR(128) NOT NULL DEFAULT '',
		run_id      VARCHAR(128) NOT NULL DEFAULT '',
		kind        VARCHAR(32) NOT NULL,
		title       VARCHAR(255) NOT NULL DEFAULT '',
		format      VARCHAR(32) NOT NULL DEFAULT 'markdown',
		content     MEDIUMTEXT NOT NULL,
		metadata    JSON,
		version     BIGINT NOT NULL DEFAULT 1,
		source      VARCHAR(64) NOT NULL DEFAULT 'agent',
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		KEY idx_user_created (user_id, created_at),
		KEY idx_thread_kind (thread_id, kind, created_at),
		KEY idx_run (run_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure artifact tables: %w", err)
	}
	return nil
}

func CreateArtifact(ctx context.Context, db *sql.DB, record ArtifactRecord) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	normalizeArtifact(&record)
	if record.Content == "" {
		return 0, fmt.Errorf("artifact content is required")
	}
	if len(record.Metadata) == 0 {
		record.Metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(record.Metadata) {
		return 0, fmt.Errorf("artifact metadata must be valid json")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin create artifact: %w", err)
	}
	defer tx.Rollback()

	if record.Version <= 0 {
		if err := tx.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(version), 0) + 1
			 FROM artifacts
			 WHERE thread_id=? AND kind=? AND title=?`,
			record.ThreadID, record.Kind, record.Title,
		).Scan(&record.Version); err != nil {
			return 0, fmt.Errorf("next artifact version: %w", err)
		}
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO artifacts (user_id, thread_id, run_id, kind, title, format, content, metadata, version, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.UserID, record.ThreadID, record.RunID, record.Kind, record.Title, record.Format, record.Content, string(record.Metadata), record.Version, record.Source,
	)
	if err != nil {
		return 0, fmt.Errorf("create artifact: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("artifact id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit create artifact: %w", err)
	}
	return id, nil
}

func ListArtifacts(ctx context.Context, db *sql.DB, userID, threadID, kind string, limit int) ([]ArtifactRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	kind = strings.TrimSpace(kind)
	if limit <= 0 {
		limit = defaultArtifactLimit
	} else if limit > 200 {
		limit = 200
	}
	query := `SELECT a.id, a.user_id, a.thread_id, a.run_id, a.kind, a.title, a.format, a.content,
		COALESCE(a.metadata, JSON_OBJECT()), a.version, a.source, a.created_at, a.updated_at
		FROM artifacts a
		LEFT JOIN threads t ON t.id=a.thread_id
		WHERE (a.user_id=? OR (COALESCE(t.team_id, '')<>'' AND EXISTS (
			SELECT 1 FROM team_members tm WHERE tm.team_id=t.team_id AND tm.user_id=?
		)))`
	args := []any{userID, userID}
	if threadID != "" {
		query += ` AND a.thread_id=?`
		args = append(args, threadID)
	}
	if kind != "" {
		query += ` AND a.kind=?`
		args = append(args, kind)
	}
	query += ` ORDER BY a.created_at DESC, a.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	defer rows.Close()

	var out []ArtifactRecord
	for rows.Next() {
		var record ArtifactRecord
		if err := rows.Scan(&record.ID, &record.UserID, &record.ThreadID, &record.RunID, &record.Kind, &record.Title, &record.Format, &record.Content, &record.Metadata, &record.Version, &record.Source, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func GetArtifact(ctx context.Context, db *sql.DB, artifactID int64, userID string) (ArtifactRecord, error) {
	if db == nil {
		return ArtifactRecord{}, fmt.Errorf("db is nil")
	}
	if artifactID <= 0 {
		return ArtifactRecord{}, fmt.Errorf("artifact id is required")
	}
	userID = NormalizeUserID(userID)
	var record ArtifactRecord
	err := db.QueryRowContext(ctx,
		`SELECT a.id, a.user_id, a.thread_id, a.run_id, a.kind, a.title, a.format, a.content,
		 COALESCE(a.metadata, JSON_OBJECT()), a.version, a.source, a.created_at, a.updated_at
		 FROM artifacts a
		 LEFT JOIN threads t ON t.id=a.thread_id
		 WHERE a.id=? AND (a.user_id=? OR (COALESCE(t.team_id, '')<>'' AND EXISTS (
			SELECT 1 FROM team_members tm WHERE tm.team_id=t.team_id AND tm.user_id=?
		 )))`,
		artifactID, userID, userID,
	).Scan(&record.ID, &record.UserID, &record.ThreadID, &record.RunID, &record.Kind, &record.Title, &record.Format, &record.Content, &record.Metadata, &record.Version, &record.Source, &record.CreatedAt, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return ArtifactRecord{}, nil
	}
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("get artifact: %w", err)
	}
	return record, nil
}

func normalizeArtifact(record *ArtifactRecord) {
	record.UserID = NormalizeUserID(record.UserID)
	record.ThreadID = strings.TrimSpace(record.ThreadID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.Kind = strings.TrimSpace(record.Kind)
	if record.Kind == "" {
		record.Kind = ArtifactKindReport
	}
	record.Title = strings.TrimSpace(record.Title)
	if record.Title == "" {
		record.Title = "Untitled artifact"
	}
	record.Format = strings.TrimSpace(record.Format)
	if record.Format == "" {
		record.Format = ArtifactFormatMD
	}
	record.Content = strings.TrimSpace(record.Content)
	record.Source = strings.TrimSpace(record.Source)
	if record.Source == "" {
		record.Source = ArtifactSourceAgent
	}
}
