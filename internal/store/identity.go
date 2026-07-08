package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const AnonymousUserID = "anonymous"

func NormalizeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AnonymousUserID
	}
	if len(userID) > 128 {
		return userID[:128]
	}
	return userID
}

func EnsureIdentityTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id           VARCHAR(128) NOT NULL PRIMARY KEY,
			provider     VARCHAR(32) NOT NULL,
			provider_id  VARCHAR(128) NOT NULL,
			display_name VARCHAR(128) NOT NULL DEFAULT '',
			created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_provider_user (provider, provider_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS threads (
			id          VARCHAR(128) NOT NULL PRIMARY KEY,
			user_id     VARCHAR(128) NOT NULL,
			title       VARCHAR(255) NOT NULL DEFAULT '',
			source      VARCHAR(32) NOT NULL DEFAULT 'web',
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			KEY idx_user_updated (user_id, updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure identity tables: %w", err)
		}
	}
	return nil
}

func EnsureUser(ctx context.Context, db *sql.DB, userID, provider, providerID string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	if provider == "" {
		provider = "local"
	}
	if providerID == "" {
		providerID = userID
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, provider, provider_id)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE provider=VALUES(provider), provider_id=VALUES(provider_id)`,
		userID, provider, providerID,
	)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}
	return nil
}

func EnsureThread(ctx context.Context, db *sql.DB, threadID, userID, title, source string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return fmt.Errorf("thread id is required")
	}
	userID = NormalizeUserID(userID)
	if source == "" {
		source = "web"
	}
	if len(title) > 255 {
		title = title[:255]
	}
	if err := EnsureUser(ctx, db, userID, source, userID); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO threads (id, user_id, title, source)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE updated_at=CURRENT_TIMESTAMP`,
		threadID, userID, title, source,
	)
	if err != nil {
		return fmt.Errorf("ensure thread: %w", err)
	}
	return nil
}

func ThreadBelongsToUser(ctx context.Context, db *sql.DB, threadID, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	var got string
	err := db.QueryRowContext(ctx, `SELECT user_id FROM threads WHERE id=?`, threadID).Scan(&got)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query thread owner: %w", err)
	}
	return got == userID, nil
}

func RunBelongsToUser(ctx context.Context, db *sql.DB, runID, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	var got string
	err := db.QueryRowContext(ctx, `SELECT user_id FROM runs WHERE id=?`, runID).Scan(&got)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query run owner: %w", err)
	}
	return got == userID, nil
}
