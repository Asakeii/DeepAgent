package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cloudwego/eino/schema"
)

// AppendMessage 追加一条消息到 messages 表。turn_idx 自增。
func AppendMessage(ctx context.Context, db *sql.DB, threadID, role, content string) error {
	idx, err := NextTurnIdx(ctx, db, threadID)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO messages (thread_id, turn_idx, role, content) VALUES (?, ?, ?, ?)`,
		threadID, idx, role, content)
	if err != nil {
		return fmt.Errorf("append message: %w", err)
	}
	return nil
}

// NextTurnIdx 返回该 thread 的下一个 turn 序号（当前最大+1，空表返回0）。
func NextTurnIdx(ctx context.Context, db *sql.DB, threadID string) (int, error) {
	var maxIdx sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT MAX(turn_idx) FROM messages WHERE thread_id = ?`, threadID).Scan(&maxIdx)
	if err != nil {
		return 0, fmt.Errorf("query max turn_idx: %w", err)
	}
	if !maxIdx.Valid {
		return 0, nil
	}
	return int(maxIdx.Int64) + 1, nil
}

// RecentMessages 取该 thread 最近 limit 条消息（按 turn_idx 升序），转为 schema.Message。
func RecentMessages(ctx context.Context, db *sql.DB, threadID string, limit int) ([]*schema.Message, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT role, content FROM messages WHERE thread_id = ? ORDER BY turn_idx DESC LIMIT ?`,
		threadID, limit)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var reversed []*schema.Message
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		reversed = append(reversed, &schema.Message{Role: schema.RoleType(role), Content: content})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	// 反转成升序（最近 limit 条，但按时间正序）
	out := make([]*schema.Message, len(reversed))
	for i, m := range reversed {
		out[len(reversed)-1-i] = m
	}
	return out, nil
}
