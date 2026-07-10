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
			team_id     VARCHAR(128) NOT NULL DEFAULT '',
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
	if err := EnsureTeamTables(ctx, db); err != nil {
		return err
	}
	return nil
}

func EnsureUser(ctx context.Context, db *sql.DB, userID, provider, providerID string) error {
	return EnsureUserProfile(ctx, db, userID, provider, providerID, "")
}

func EnsureUserProfile(ctx context.Context, db *sql.DB, userID, provider, providerID, displayName string) error {
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
	if len(providerID) > 128 {
		providerID = providerID[:128]
	}
	displayName = strings.TrimSpace(displayName)
	if len(displayName) > 128 {
		displayName = displayName[:128]
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, provider, provider_id, display_name)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 display_name=IF(VALUES(display_name)<>'', VALUES(display_name), display_name)`,
		userID, provider, providerID, displayName,
	)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}
	return nil
}

func EnsureThread(ctx context.Context, db *sql.DB, threadID, userID, title, source string) error {
	return EnsureThreadWithTeam(ctx, db, threadID, userID, "", title, source)
}

func EnsureThreadWithTeam(ctx context.Context, db *sql.DB, threadID, userID, teamID, title, source string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return fmt.Errorf("thread id is required")
	}
	userID = NormalizeUserID(userID)
	teamID = strings.TrimSpace(teamID)
	if source == "" {
		source = "web"
	}
	if len(title) > 255 {
		title = title[:255]
	}
	if err := EnsureUser(ctx, db, userID, source, userID); err != nil {
		return err
	}
	if teamID != "" {
		if ok, err := UserIsTeamMember(ctx, db, teamID, userID); err != nil {
			return err
		} else if !ok {
			return ErrTeamForbidden
		}
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO threads (id, user_id, team_id, title, source)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE updated_at=CURRENT_TIMESTAMP`,
		threadID, userID, teamID, title, source,
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
	var ownerID, teamID string
	err := db.QueryRowContext(ctx, `SELECT user_id, COALESCE(team_id, '') FROM threads WHERE id=?`, threadID).Scan(&ownerID, &teamID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query thread owner: %w", err)
	}
	if ownerID == userID {
		return true, nil
	}
	if teamID == "" {
		return false, nil
	}
	return UserIsTeamMember(ctx, db, teamID, userID)
}

func RunBelongsToUser(ctx context.Context, db *sql.DB, runID, userID string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	var ownerID, teamID string
	err := db.QueryRowContext(ctx,
		`SELECT r.user_id, COALESCE(t.team_id, '')
		 FROM runs r
		 LEFT JOIN threads t ON t.id=r.thread_id
		 WHERE r.id=?`,
		runID,
	).Scan(&ownerID, &teamID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query run owner: %w", err)
	}
	if ownerID == userID {
		return true, nil
	}
	if teamID == "" {
		return false, nil
	}
	return UserIsTeamMember(ctx, db, teamID, userID)
}
