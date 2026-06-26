package store

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// StartReminderTicker 启动后台 ticker goroutine。
// 每分钟扫一次 reminders 表，触发到期的提醒。无状态：状态全在 MySQL，
// 多副本通过 UPDATE 行锁抢触发权。返回 stop 函数用于优雅关闭。
func StartReminderTicker(db *sql.DB) (stop func()) {
	t := time.NewTicker(1 * time.Minute)
	done := make(chan struct{})
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	go func() {
		for {
			select {
			case <-t.C:
				firePending(context.Background(), db, parser)
			case <-done:
				t.Stop()
				return
			}
		}
	}()

	return func() { close(done) }
}

func firePending(ctx context.Context, db *sql.DB, parser cron.Parser) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, thread_id, cron, content FROM reminders
		 WHERE status='pending' AND next_fire_at <= NOW()`)
	if err != nil {
		log.Printf("[reminder] query: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var threadID, cronExpr, content string
		if err := rows.Scan(&id, &threadID, &cronExpr, &content); err != nil {
			log.Printf("[reminder] scan: %v", err)
			continue
		}

		// 抢锁：将状态从 pending 改为 firing，仅影响 1 行时该实例负责触发
		result, err := db.ExecContext(ctx,
			`UPDATE reminders SET status='firing' WHERE id=? AND status='pending'`, id)
		if err != nil {
			log.Printf("[reminder] lock %d: %v", id, err)
			continue
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			continue // 被其他实例抢了
		}

		// 触发提醒
		log.Printf("[reminder] FIRE | thread=%s | %s", threadID, content)

		// 计算下次触发时间，重置为 pending
		sched, err := parser.Parse(cronExpr)
		if err != nil {
			// cron 解析失败，停用这条提醒
			_, _ = db.ExecContext(ctx,
				`UPDATE reminders SET status='broken' WHERE id=?`, id)
			log.Printf("[reminder] broken cron %q for id=%d: %v", cronExpr, id, err)
			continue
		}
		nextFire := sched.Next(time.Now())
		if nextFire.IsZero() {
			_, _ = db.ExecContext(ctx,
				`UPDATE reminders SET status='expired' WHERE id=?`, id)
			continue
		}
		_, _ = db.ExecContext(ctx,
			`UPDATE reminders SET status='pending', next_fire_at=? WHERE id=?`,
			nextFire, id)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[reminder] rows: %v", err)
	}
}
