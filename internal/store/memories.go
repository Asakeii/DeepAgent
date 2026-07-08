package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const MemoryKindPreference = "preference"

type MemoryRecord struct {
	ID         int64
	UserID     string
	ThreadID   string
	Kind       string
	Content    string
	Importance int
	Source     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func CreateMemory(ctx context.Context, db *sql.DB, record MemoryRecord) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	record.UserID = NormalizeUserID(record.UserID)
	record.Kind = strings.TrimSpace(record.Kind)
	record.Content = strings.TrimSpace(record.Content)
	if record.Kind == "" {
		record.Kind = MemoryKindPreference
	}
	if record.Content == "" {
		return 0, fmt.Errorf("memory content is required")
	}
	if len(record.Kind) > 32 {
		record.Kind = record.Kind[:32]
	}
	if len(record.Source) > 64 {
		record.Source = record.Source[:64]
	}
	if len(record.Content) > 4096 {
		record.Content = record.Content[:4096]
	}
	res, err := db.ExecContext(ctx,
		`INSERT INTO memories (user_id, thread_id, kind, content, importance, source)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		record.UserID, record.ThreadID, record.Kind, record.Content, record.Importance, record.Source,
	)
	if err != nil {
		return 0, fmt.Errorf("create memory: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("memory id: %w", err)
	}
	return id, nil
}

func ListMemories(ctx context.Context, db *sql.DB, userID, kind string, limit int) ([]MemoryRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `SELECT id, user_id, thread_id, kind, content, importance, source, created_at, updated_at
		FROM memories WHERE user_id=?`
	args := []any{userID}
	if strings.TrimSpace(kind) != "" {
		query += ` AND kind=?`
		args = append(args, strings.TrimSpace(kind))
	}
	query += ` ORDER BY importance DESC, updated_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var out []MemoryRecord
	for rows.Next() {
		var record MemoryRecord
		if err := rows.Scan(&record.ID, &record.UserID, &record.ThreadID, &record.Kind, &record.Content, &record.Importance, &record.Source, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}
