package infra

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"deepAgent/conf"
	"deepAgent/internal/store"
	"github.com/cloudwego/eino/schema"
)

// DB 全局 MySQL 连接，无状态服务共享。所有持久化走它。
var DB *sql.DB

// InitDB 初始化全局 MySQL 连接。DSN 在 conf.database.dsn。
func InitDB(ctx context.Context) error {
	if conf.App == nil {
		return fmt.Errorf("config is not loaded")
	}
	dsn := conf.App.Database.DSN
	if dsn == "" {
		return fmt.Errorf("database.dsn is empty")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(20)
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}
	if err := store.EnsureIdentityTables(ctx, db); err != nil {
		return err
	}
	if err := store.EnsureRunTables(ctx, db); err != nil {
		return err
	}
	DB = db
	return nil
}

// RecentMessagesForCheckin 取某 thread 最近历史（封装给 main 用，避免 main 直接依赖 store 包做类型转换）。
func RecentMessagesForCheckin(ctx context.Context, threadID string, limit int) ([]*schema.Message, error) {
	return store.RecentMessages(ctx, DB, threadID, limit)
}

// AppendMessageForCheckin 追加一条消息。
func AppendMessageForCheckin(ctx context.Context, threadID, role, content string) error {
	return store.AppendMessage(ctx, DB, threadID, role, content)
}

// ListThreads 封装 store.ListThreads。
func ListThreads(ctx context.Context, limit int) ([]store.ThreadInfo, error) {
	return store.ListThreads(ctx, DB, limit)
}
