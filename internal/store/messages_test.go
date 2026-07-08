package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	_ "github.com/go-sql-driver/mysql"
)

func TestMessagesAppendAndRecent(t *testing.T) {
	db := DBForTest(t) // 复用 checkpoint_test.go 里已有的 DBForTest
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	tid := "msg-test-" + randomSuffix()

	// 空历史
	msgs, err := RecentMessages(ctx, db, tid, 10)
	if err != nil || len(msgs) != 0 {
		t.Fatalf("expect empty, got %v err=%v", msgs, err)
	}

	// append 两条
	// 注：brief 给的签名为 role string，而 schema.User 是 schema.RoleType（type RoleType string，
	// 具名类型不可隐式转 string），故此处显式 string() 转换。详见 task-1-report 的 concerns。
	if err := AppendMessage(ctx, db, tid, string(schema.User), "今天跑步5km"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, tid, string(schema.Assistant), "已记录，坚持得不错"); err != nil {
		t.Fatal(err)
	}

	// 取回
	msgs, err = RecentMessages(ctx, db, tid, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Role != schema.User || msgs[1].Content != "已记录，坚持得不错" {
		t.Fatalf("unexpected: %+v", msgs)
	}

	// 清理
	_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE thread_id=?", tid)
}

func TestAppendMessageConcurrentTurnIdxUnique(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	tid := "msg-concurrent-" + randomSuffix()
	const total = 24

	var wg sync.WaitGroup
	errs := make(chan error, total)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- AppendMessage(ctx, db, tid, string(schema.User), fmt.Sprintf("msg-%02d", i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	var count, distinctTurns int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(DISTINCT turn_idx) FROM messages WHERE thread_id=?`, tid).Scan(&count, &distinctTurns); err != nil {
		t.Fatal(err)
	}
	if count != total || distinctTurns != total {
		t.Fatalf("count=%d distinctTurns=%d want %d", count, distinctTurns, total)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE thread_id=?", tid)
}

// randomSuffix 给测试用唯一 thread_id；不在生产路径。
func randomSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
