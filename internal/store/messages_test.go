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

func TestSearchThreadsForUser(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "session-search-user-" + randomSuffix()
	threadMatch := "session-search-match-" + randomSuffix()
	threadOther := "session-search-other-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE thread_id IN (?, ?)", threadMatch, threadOther)
		_, _ = db.ExecContext(ctx, "DELETE FROM threads WHERE id IN (?, ?)", threadMatch, threadOther)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID)
	})

	if err := EnsureThread(ctx, db, threadMatch, userID, "研究 Agent 成熟化", "test"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThread(ctx, db, threadOther, userID, "普通会话", "test"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, threadMatch, string(schema.User), "帮我分析多 Pod 部署"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, threadOther, string(schema.User), "记录今天跑步"); err != nil {
		t.Fatal(err)
	}

	threads, err := SearchThreadsForUser(ctx, db, userID, "多 Pod", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ThreadID != threadMatch {
		t.Fatalf("unexpected search result: %+v", threads)
	}
}

func TestSearchThreadsForUserInScope(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "session-scope-user-" + randomSuffix()
	personalThread := "session-scope-personal-" + randomSuffix()
	teamThread := "session-scope-team-" + randomSuffix()
	team, err := CreateTeam(ctx, db, userID, "Scope Team "+randomSuffix())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE thread_id IN (?, ?)", personalThread, teamThread)
		_, _ = db.ExecContext(ctx, "DELETE FROM threads WHERE id IN (?, ?)", personalThread, teamThread)
		_, _ = db.ExecContext(ctx, "DELETE FROM team_members WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM teams WHERE id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID)
	})

	if err := EnsureThread(ctx, db, personalThread, userID, "个人会话", "test"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThreadWithTeam(ctx, db, teamThread, userID, team.ID, "团队会话", "test"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, personalThread, string(schema.User), "个人空间研究"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, teamThread, string(schema.User), "团队空间研究"); err != nil {
		t.Fatal(err)
	}

	personalScope := ""
	personal, err := SearchThreadsForUserInScope(ctx, db, userID, "空间研究", &personalScope, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(personal) != 1 || personal[0].ThreadID != personalThread || personal[0].TeamID != "" {
		t.Fatalf("unexpected personal scope: %+v", personal)
	}

	teamScope := team.ID
	teamThreads, err := SearchThreadsForUserInScope(ctx, db, userID, "空间研究", &teamScope, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(teamThreads) != 1 || teamThreads[0].ThreadID != teamThread || teamThreads[0].TeamID != team.ID {
		t.Fatalf("unexpected team scope: %+v", teamThreads)
	}
}

// randomSuffix 给测试用唯一 thread_id；不在生产路径。
func randomSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
