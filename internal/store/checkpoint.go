package store

import (
	"context"
	"database/sql"

	"github.com/cloudwego/eino/compose"
)

// MySQLCheckPoint 把 eino 图中断现场存 MySQL，使服务无状态可跨进程恢复。
// []byte 是 eino 序列化器产物，直接当不透明 blob 存，不二次序列化。
type MySQLCheckPoint struct {
	db *sql.DB
}

func NewMySQLCheckPoint(db *sql.DB) compose.CheckPointStore {
	if db == nil {
		panic("NewMySQLCheckPoint: db must not be nil")
	}
	return &MySQLCheckPoint{db: db}
}

func (c *MySQLCheckPoint) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	var data []byte
	err := c.db.QueryRowContext(ctx,
		`SELECT data FROM graph_checkpoint WHERE thread_id = ?`, checkPointID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (c *MySQLCheckPoint) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO graph_checkpoint (thread_id, data) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE data = VALUES(data)`,
		checkPointID, checkPoint)
	return err
}
