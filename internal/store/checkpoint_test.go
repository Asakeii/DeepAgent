package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// DBForTest 用环境变量 TEST_MYSQL_DSN 起一个 *sql.DB，无 DSN 时返回 nil。
// 测试文件与 db.go 是不同包的导入方，_ "github.com/go-sql-driver/mysql" 不构成重复注册。
func DBForTest(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil
	}
	// 确保表存在
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS graph_checkpoint (
        thread_id VARCHAR(128) NOT NULL PRIMARY KEY,
        data LONGBLOB NOT NULL,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    )`)
	_ = EnsureIdentityTables(context.Background(), db)
	_ = EnsureMessageTables(context.Background(), db)
	_ = EnsureToolAuditTables(context.Background(), db)
	_ = EnsureArtifactTables(context.Background(), db)
	_ = EnsureCitationTables(context.Background(), db)
	_ = EnsureUserSettingsTables(context.Background(), db)
	_ = EnsureModelUsageTables(context.Background(), db)
	_ = EnsurePluginTables(context.Background(), db)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS runs (
        id VARCHAR(128) NOT NULL PRIMARY KEY,
        user_id VARCHAR(128) NOT NULL DEFAULT '',
        thread_id VARCHAR(128) NOT NULL,
        mode VARCHAR(32) NOT NULL,
        status VARCHAR(32) NOT NULL,
        started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        ended_at TIMESTAMP NULL,
        error TEXT,
        KEY idx_thread_started (thread_id, started_at),
        KEY idx_user_started (user_id, started_at)
    )`)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS run_events (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        run_id VARCHAR(128) NOT NULL,
        thread_id VARCHAR(128) NOT NULL,
        user_id VARCHAR(128) NOT NULL DEFAULT '',
        event_name VARCHAR(64) NOT NULL,
        agent VARCHAR(64) NOT NULL DEFAULT '',
        payload JSON NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        KEY idx_run_id (run_id, id),
        KEY idx_thread_created (thread_id, created_at)
    )`)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMySQLCheckPointGetSet(t *testing.T) {
	// 无 DB 时跳过，避免单测强依赖 MySQL；CI/本地起 MySQL 后跑。
	if DBForTest(t) == nil {
		t.Skip("mysql not available")
	}
	cp := NewMySQLCheckPoint(DBForTest(t))
	ctx := context.Background()

	// 不存在返回 found=false
	_, found, err := cp.Get(ctx, "tid-not-exist")
	if err != nil || found {
		t.Fatalf("expect not found, got found=%v err=%v", found, err)
	}
	// Set 后能 Get 回来
	if err := cp.Set(ctx, "tid1", []byte("blob-v1")); err != nil {
		t.Fatal(err)
	}
	got, found, err := cp.Get(ctx, "tid1")
	if err != nil || !found || string(got) != "blob-v1" {
		t.Fatalf("get tid1: got=%q found=%v err=%v", got, found, err)
	}
	// 覆盖写（同 id 只保留最新）
	if err := cp.Set(ctx, "tid1", []byte("blob-v2")); err != nil {
		t.Fatal(err)
	}
	got2, _, _ := cp.Get(ctx, "tid1")
	if string(got2) != "blob-v2" {
		t.Fatalf("expect overwrite v2, got %q", got2)
	}
}
