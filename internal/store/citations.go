package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type CitationRecord struct {
	ID         int64
	ArtifactID int64
	UserID     string
	ThreadID   string
	RunID      string
	Title      string
	URL        string
	Quote      string
	Position   int
	CreatedAt  time.Time
}

func EnsureCitationTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS artifact_citations (
		id          BIGINT AUTO_INCREMENT PRIMARY KEY,
		artifact_id BIGINT NOT NULL,
		user_id     VARCHAR(128) NOT NULL DEFAULT '',
		thread_id   VARCHAR(128) NOT NULL DEFAULT '',
		run_id      VARCHAR(128) NOT NULL DEFAULT '',
		title       VARCHAR(512) NOT NULL DEFAULT '',
		url         TEXT NOT NULL,
		quote       TEXT,
		position    INT NOT NULL DEFAULT 0,
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		KEY idx_artifact_position (artifact_id, position),
		KEY idx_user_created (user_id, created_at),
		KEY idx_thread_created (thread_id, created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure citation tables: %w", err)
	}
	return nil
}

func CreateArtifactCitations(ctx context.Context, db *sql.DB, records []CitationRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if len(records) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create citations: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO artifact_citations
		(artifact_id, user_id, thread_id, run_id, title, url, quote, position)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare create citations: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		normalizeCitation(&record)
		if record.ArtifactID <= 0 {
			return fmt.Errorf("artifact id is required")
		}
		if record.URL == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, record.ArtifactID, record.UserID, record.ThreadID, record.RunID, record.Title, record.URL, nullableString(record.Quote), record.Position); err != nil {
			return fmt.Errorf("create citation: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create citations: %w", err)
	}
	return nil
}

func ListArtifactCitations(ctx context.Context, db *sql.DB, userID string, artifactID int64) ([]CitationRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if artifactID <= 0 {
		return nil, fmt.Errorf("artifact id is required")
	}
	userID = NormalizeUserID(userID)
	rows, err := db.QueryContext(ctx,
		`SELECT id, artifact_id, user_id, thread_id, run_id, title, url, COALESCE(quote, ''), position, created_at
		 FROM artifact_citations
		 WHERE user_id=? AND artifact_id=?
		 ORDER BY position ASC, id ASC`,
		userID, artifactID,
	)
	if err != nil {
		return nil, fmt.Errorf("list artifact citations: %w", err)
	}
	defer rows.Close()

	var out []CitationRecord
	for rows.Next() {
		var record CitationRecord
		if err := rows.Scan(&record.ID, &record.ArtifactID, &record.UserID, &record.ThreadID, &record.RunID, &record.Title, &record.URL, &record.Quote, &record.Position, &record.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func ArtifactBelongsToUser(ctx context.Context, db *sql.DB, artifactID int64, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	if artifactID <= 0 {
		return false, nil
	}
	userID = NormalizeUserID(userID)
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifacts WHERE id=? AND user_id=?`, artifactID, userID).Scan(&count); err != nil {
		return false, fmt.Errorf("artifact ownership: %w", err)
	}
	return count > 0, nil
}

func normalizeCitation(record *CitationRecord) {
	record.UserID = NormalizeUserID(record.UserID)
	record.ThreadID = strings.TrimSpace(record.ThreadID)
	record.RunID = strings.TrimSpace(record.RunID)
	record.Title = strings.TrimSpace(record.Title)
	record.URL = strings.TrimSpace(record.URL)
	record.Quote = strings.TrimSpace(record.Quote)
	if len(record.Title) > 512 {
		record.Title = record.Title[:512]
	}
}
