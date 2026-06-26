package tool

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// ---- create_reminder ----

type createReminderInput struct {
	Content string `json:"content" jsonschema:"required" jsonschema_description:"提醒内容，如'喝水'"`
	Cron    string `json:"cron" jsonschema:"required" jsonschema_description:"cron 表达式，如'0 8 * * *'表示每天8点，'*/30 * * * *'表示每30分钟"`
}

type createReminderOutput struct {
	ID         int64  `json:"id"`
	NextFireAt string `json:"next_fire_at"`
	Message    string `json:"message"`
}

func createReminder(ctx context.Context, in createReminderInput, threadID string, db *sql.DB) (createReminderOutput, error) {
	if in.Content == "" || in.Cron == "" {
		return createReminderOutput{}, fmt.Errorf("content and cron required")
	}

	// 解析 cron 表达式
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(in.Cron)
	if err != nil {
		return createReminderOutput{}, fmt.Errorf("invalid cron %q: %w", in.Cron, err)
	}

	nextFire := sched.Next(time.Now())
	if nextFire.IsZero() {
		return createReminderOutput{}, fmt.Errorf("cron %q has no future fire time", in.Cron)
	}

	result, err := db.ExecContext(ctx,
		`INSERT INTO reminders (thread_id, cron, content, next_fire_at, status) VALUES (?, ?, ?, ?, 'pending')`,
		threadID, in.Cron, in.Content, nextFire)
	if err != nil {
		return createReminderOutput{}, fmt.Errorf("insert reminder: %w", err)
	}

	id, _ := result.LastInsertId()
	return createReminderOutput{
		ID:         id,
		NextFireAt: nextFire.Format("2006-01-02 15:04:05"),
		Message:    fmt.Sprintf("已设置提醒：%s（%s，下次触发 %s）", in.Content, in.Cron, nextFire.Format("15:04")),
	}, nil
}

// ---- list_reminders ----

type listRemindersInput struct {
	Limit int `json:"limit" jsonschema_description:"返回条数上限，默认10"`
}

type reminderRecord struct {
	ID         int64  `json:"id"`
	Cron       string `json:"cron"`
	Content    string `json:"content"`
	NextFireAt string `json:"next_fire_at"`
	Status     string `json:"status"`
}

type listRemindersOutput struct {
	Reminders []reminderRecord `json:"reminders"`
}

func listReminders(ctx context.Context, in listRemindersInput, threadID string, db *sql.DB) (listRemindersOutput, error) {
	if in.Limit <= 0 {
		in.Limit = 10
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, cron, content, next_fire_at, status FROM reminders WHERE thread_id=? ORDER BY id DESC LIMIT ?`,
		threadID, in.Limit)
	if err != nil {
		return listRemindersOutput{}, fmt.Errorf("query reminders: %w", err)
	}
	defer rows.Close()

	var out listRemindersOutput
	for rows.Next() {
		var r reminderRecord
		var nf time.Time
		if err := rows.Scan(&r.ID, &r.Cron, &r.Content, &nf, &r.Status); err != nil {
			return listRemindersOutput{}, err
		}
		r.NextFireAt = nf.Format("2006-01-02 15:04:05")
		out.Reminders = append(out.Reminders, r)
	}
	return out, rows.Err()
}

// ---- delete_reminder ----

type deleteReminderInput struct {
	ID int64 `json:"id" jsonschema:"required" jsonschema_description:"提醒记录 ID（从 list_reminders 获取）"`
}

type deleteReminderOutput struct {
	Message string `json:"message"`
}

func deleteReminder(ctx context.Context, in deleteReminderInput, threadID string, db *sql.DB) (deleteReminderOutput, error) {
	result, err := db.ExecContext(ctx,
		`DELETE FROM reminders WHERE id=? AND thread_id=?`, in.ID, threadID)
	if err != nil {
		return deleteReminderOutput{}, fmt.Errorf("delete reminder: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return deleteReminderOutput{}, fmt.Errorf("reminder %d not found or not yours", in.ID)
	}
	return deleteReminderOutput{Message: "已删除提醒"}, nil
}

// ---- tool registration helpers (used by checkin_tools.go) ----

func makeCreateReminder(db *sql.DB) func(ctx context.Context, in createReminderInput) (createReminderOutput, error) {
	return func(ctx context.Context, in createReminderInput) (createReminderOutput, error) {
		tid := ThreadIDFromCtx(ctx)
		return createReminder(ctx, in, tid, db)
	}
}

func makeListReminders(db *sql.DB) func(ctx context.Context, in listRemindersInput) (listRemindersOutput, error) {
	return func(ctx context.Context, in listRemindersInput) (listRemindersOutput, error) {
		tid := ThreadIDFromCtx(ctx)
		return listReminders(ctx, in, tid, db)
	}
}

func makeDeleteReminder(db *sql.DB) func(ctx context.Context, in deleteReminderInput) (deleteReminderOutput, error) {
	return func(ctx context.Context, in deleteReminderInput) (deleteReminderOutput, error) {
		tid := ThreadIDFromCtx(ctx)
		return deleteReminder(ctx, in, tid, db)
	}
}
