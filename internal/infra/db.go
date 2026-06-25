package infra

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"deepAgent/conf"
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
	DB = db
	return nil
}
