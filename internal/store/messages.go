package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func EnsureMessageTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS messages (
		id          BIGINT AUTO_INCREMENT PRIMARY KEY,
		thread_id   VARCHAR(128) NOT NULL,
		turn_idx    BIGINT NOT NULL,
		role        VARCHAR(32) NOT NULL,
		content     MEDIUMTEXT NOT NULL,
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		KEY idx_thread_turn (thread_id, turn_idx)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure messages table: %w", err)
	}
	needsAlter, err := messageTurnIdxNeedsAlter(ctx, db)
	if err != nil {
		return err
	}
	if needsAlter {
		if _, err := db.ExecContext(ctx, `ALTER TABLE messages MODIFY turn_idx BIGINT NOT NULL`); err != nil {
			return fmt.Errorf("ensure messages turn_idx: %w", err)
		}
	}
	return nil
}

func messageTurnIdxNeedsAlter(ctx context.Context, db *sql.DB) (bool, error) {
	var dataType string
	err := db.QueryRowContext(ctx, `SELECT DATA_TYPE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='messages' AND COLUMN_NAME='turn_idx'`).Scan(&dataType)
	if err != nil {
		return false, fmt.Errorf("inspect messages turn_idx: %w", err)
	}
	return strings.ToLower(dataType) != "bigint", nil
}

// AppendMessage 追加一条消息到 messages 表。
// turn_idx 使用该行自增 id，避免多 pod 并发写同一 thread 时出现 MAX(turn_idx)+1 竞争。
func AppendMessage(ctx context.Context, db *sql.DB, threadID, role, content string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin append message: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO messages (thread_id, turn_idx, role, content) VALUES (?, 0, ?, ?)`,
		threadID, role, content)
	if err != nil {
		return fmt.Errorf("append message: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("message insert id: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE messages SET turn_idx=? WHERE id=?`, id, id)
	if err != nil {
		return fmt.Errorf("set message turn_idx: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit append message: %w", err)
	}
	return nil
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

// ThreadInfo 会话摘要信息。
type ThreadInfo struct {
	ThreadID string `json:"thread_id"`
	TeamID   string `json:"team_id,omitempty"`
	FirstMsg string `json:"first_msg"` // 第一条用户消息作为标题
	LastAt   string `json:"last_at"`   // 最后消息时间
	MsgCount int    `json:"msg_count"`
}

// ListThreads 返回所有 thread 的基本信息，按最近活动排序。
func ListThreads(ctx context.Context, db *sql.DB, limit int) ([]ThreadInfo, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT thread_id, 
		 (SELECT content FROM messages m2 WHERE m2.thread_id=m1.thread_id AND role='user' ORDER BY turn_idx ASC LIMIT 1) as first_msg,
		 MAX(created_at) as last_at,
		 COUNT(*) as msg_count
		 FROM messages m1 GROUP BY thread_id ORDER BY last_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()
	var out []ThreadInfo
	for rows.Next() {
		var t ThreadInfo
		if err := rows.Scan(&t.ThreadID, &t.FirstMsg, &t.LastAt, &t.MsgCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func ListThreadsForUser(ctx context.Context, db *sql.DB, userID string, limit int) ([]ThreadInfo, error) {
	userID = NormalizeUserID(userID)
	return SearchThreadsForUser(ctx, db, userID, "", limit)
}

func SearchThreadsForUser(ctx context.Context, db *sql.DB, userID, query string, limit int) ([]ThreadInfo, error) {
	return SearchThreadsForUserInScope(ctx, db, userID, query, nil, limit)
}

func SearchThreadsForUserInScope(ctx context.Context, db *sql.DB, userID, query string, teamID *string, limit int) ([]ThreadInfo, error) {
	userID = NormalizeUserID(userID)
	query = strings.TrimSpace(query)
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	where := `WHERE (t.user_id=? OR (t.team_id<>'' AND EXISTS (
		SELECT 1 FROM team_members tm WHERE tm.team_id=t.team_id AND tm.user_id=?
	)))`
	args := []any{userID, userID}
	if teamID != nil {
		scopeTeamID := strings.TrimSpace(*teamID)
		where += ` AND t.team_id=?`
		args = append(args, scopeTeamID)
	}
	if query != "" {
		like := "%" + strings.ReplaceAll(query, "%", `\%`) + "%"
		where += ` AND (t.id LIKE ? OR t.title LIKE ? OR EXISTS (
			SELECT 1 FROM messages sm WHERE sm.thread_id=t.id AND sm.content LIKE ?
		))`
		args = append(args, like, like, like)
	}
	args = append(args, limit)
	rows, err := db.QueryContext(ctx,
		`SELECT t.id, COALESCE(t.team_id, ''),
		 COALESCE(NULLIF(t.title, ''), (SELECT content FROM messages m2 WHERE m2.thread_id=t.id AND role='user' ORDER BY turn_idx ASC LIMIT 1), '') AS first_msg,
		 COALESCE(MAX(m.created_at), t.updated_at) AS last_at,
		 COUNT(m.id) AS msg_count
		 FROM threads t
		 LEFT JOIN messages m ON m.thread_id=t.id
		 `+where+`
		 GROUP BY t.id, t.team_id, t.title, t.updated_at
		 ORDER BY last_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list user threads: %w", err)
	}
	defer rows.Close()
	var out []ThreadInfo
	for rows.Next() {
		var t ThreadInfo
		if err := rows.Scan(&t.ThreadID, &t.TeamID, &t.FirstMsg, &t.LastAt, &t.MsgCount); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
