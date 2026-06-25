# 自律打卡助手 Stage 1: Docker 基建 + MySQL 存储 + 无状态 Checkpoint 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 docker compose 统一启动 MySQL + deepAgent，实现 MySQL 后端的无状态 CheckPointStore，替换现有内存 checkpoint，使现有研究报告流程可在容器内跨进程中断恢复。

**Architecture:** docker compose 起 mysql:8（自动建表）与 deepAgent 容器（depends_on mysql，DSN 指向容器名）。新增 `internal/store` 包：`mysql.go` 管 DB 连接与业务表 CRUD，`checkpoint.go` 实现 eino `compose.CheckPointStore` 接口（后端 MySQL `graph_checkpoint` 表，thread_id→[]byte blob）。替换 builder 里的内存 `NewDeepAgentCheckPoint` 为 MySQL 版。验证：现有研究流程在新 checkpoint 下仍能触发 human_feedback 中断并用同 thread_id 恢复。

**Tech Stack:** Go 1.25.5, eino v0.9.7, github.com/go-sql-driver/mysql, database/sql, docker compose, mysql:8

## Global Constraints

- 所有 deepAgent 文件用绝对路径操作，工作目录在 /Users/gd-npc-1029/Documents/CodeBase/deepAgent。
- Go module 名为 `deepAgent`（见 go.mod 顶部，import 前缀 `deepAgent/...`）。
- eino 版本固定 v0.9.7，不升级；CheckPointStore 接口来自 `github.com/cloudwego/eino/compose`（是 `core.CheckPointStore` 的别名）。
- MySQL 驱动：`github.com/go-sql-driver/mysql`，当前 go.mod 未引入，需 `go get`。
- 服务无状态：进程零会话级内存态；checkpoint 与业务表都以 thread_id 为 key 存 MySQL。`[]byte` 是 eino 序列化器产物，直接当不透明 blob 存，不二次序列化。
- 配置文件 `conf/deep-agent.yaml` 被 .gitignore，不进 git；改配置结构同时改 `deep-agent.yaml.example`。
- 每个任务结束都要 `go build ./...` 通过 + 提交一次。
- 不改现有研究图节点逻辑（coordinator/planner/researcher 等），只替换 checkpoint store 来源。

---

## File Structure

（本阶段涉及的文件，按职责单一原则拆分）

- `internal/store/mysql.go` 【新建】DB 连接初始化 + 业务表（checkins/plans/reminders/messages）CRUD。本阶段只实现 schema 建表辅助与连接，业务 CRUD 留空壳供后续阶段填。
- `internal/store/checkpoint.go` 【新建】MySQL 后端的 `compose.CheckPointStore` 实现（Get/Set），表 `graph_checkpoint(thread_id, data, updated_at)`。
- `internal/store/schema.sql` 【新建】所有建表 DDL，docker 启动时自动执行。
- `conf/config.go` 【改】加 `DatabaseConfig` 与 `ModelConfig.VisionModel` 段。
- `conf/deep-agent.yaml.example` 【改】加 `database.dsn` 与 `model.vision_model` 示例。
- `internal/model/state.go` 【改】删除内存 `DeepAgentCheckPoint`/`NewDeepAgentCheckPoint`，改为返回注入 store 的工厂（保持 `NewDeepAgentCheckPoint` 签名不变以免动 builder，内部改为取 infra 全局 MySQL store）。
- `internal/infra/db.go` 【新建】`InitDB(ctx)` 初始化全局 `*sql.DB`；`infra.DB` 全局变量。
- `internal/infra/llm.go` 【改】`InitModel` 增加 `VisionModel` 初始化（缺失回退 ChatModel）。
- `internal/agent/builder.go` 【改】`WithCheckPointStore` 改为传 MySQL store（通过 model.NewDeepAgentCheckPoint 间接拿到）。
- `main.go` 【改】`InitMCP` 前加 `infra.InitDB`；启动顺序：Load→InitDB→InitModel→InitMCP。
- `deploy/docker-compose.yml` 【新建】mysql + deepagent 两服务。
- `deploy/Dockerfile` 【新建】多阶段构建 deepAgent 二进制。
- `internal/store/checkpoint_test.go` 【新建】测 CheckPointStore Get/Set/覆盖写。
- `conf/config_test.go` 【新建】测 database 段解析。

---

## Task 1: 引入 MySQL 驱动并加 database 配置段

**Files:**
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/go.mod`
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/conf/config.go`
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/conf/deep-agent.yaml.example`
- Test: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/conf/config_test.go`

**Interfaces:**
- Consumes: 现有 `conf.Config` 结构
- Produces: `conf.DatabaseConfig{ DSN string }`；`conf.App.Database.DSN` 可读；`ModelConfig.VisionModel *ModelConfig`

- [ ] **Step 1: 加 mysql 驱动依赖**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
go get github.com/go-sql-driver/mysql
go mod tidy
```
Expected: go.mod 出现 `github.com/go-sql-driver/mysql vX.X.X`，无报错。

- [ ] **Step 2: 写 config 解析失败测试**

创建 `conf/config_test.go`：
```go
package conf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesDatabaseAndVision(t *testing.T) {
	dir := t.TempDir()
	ys := []byte(`
database:
  dsn: "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true"
model:
  default_model: "m"
  api_key: "k"
  base_url: "u"
  vision_model:
    default_model: "vm"
    api_key: "vk"
    base_url: "vu"
setting:
  max_step_num: 3
`)
	if err := os.WriteFile(filepath.Join(dir, "deep-agent.yaml"), ys, 0644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	_ = os.Chdir(dir)

	cfg, err := Load(t.Context())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Database.DSN == "" {
		t.Fatal("DSN empty")
	}
	if cfg.Model.VisionModel == nil || cfg.Model.VisionModel.DefaultModel != "vm" {
		t.Fatalf("vision model not parsed: %+v", cfg.Model.VisionModel)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && go test ./conf/ -run TestLoadParsesDatabaseAndVision -v`
Expected: FAIL（Config 无 Database/VisionModel 字段）。

- [ ] **Step 4: 改 config.go 加字段**

在 `conf/config.go` 的 `Config` struct 加 `Database DatabaseConfig \`yaml:"database"\``；在 `ModelConfig` 加 `VisionModel *ModelConfig \`yaml:"vision_model,omitempty"\``；新增：
```go
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}
```
注意 `Load` 里 `App` 赋值前不要因 DSN 空而报错（本阶段允许空，InitDB 时才校验）。

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && go test ./conf/ -run TestLoadParsesDatabaseAndVision -v`
Expected: PASS。

- [ ] **Step 6: 更新 example yaml**

在 `conf/deep-agent.yaml.example` 的 `model:` 段下加：
```yaml
  vision_model:              # 识图 agent 专用，初值复用上方；缺失回退主模型
    default_model: "<your vision model>"
    api_key: "<your api key>"
    base_url: "<your base url>"
```
在文件末尾加：
```yaml
database:
  dsn: "root:deepagent@tcp(mysql:3306)/deepagent?parseTime=true"
```

- [ ] **Step 7: build + vet + 提交**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
go build ./... && go vet ./...
git add go.mod go.sum conf/config.go conf/config_test.go conf/deep-agent.yaml.example
git commit -m "feat(conf): 加 database 与 vision_model 配置段"
```

## Task 2: schema.sql + MySQL 连接初始化

**Files:**
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/store/schema.sql`
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/infra/db.go`
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/main.go`（启动调 InitDB）

**Interfaces:**
- Consumes: `conf.App.Database.DSN`
- Produces: `infra.DB *sql.DB`（全局）；`infra.InitDB(ctx context.Context) error`

- [ ] **Step 1: 写 schema.sql**

创建 `internal/store/schema.sql`：
```sql
CREATE TABLE IF NOT EXISTS graph_checkpoint (
    thread_id   VARCHAR(128) NOT NULL PRIMARY KEY,
    data        LONGBLOB NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id   VARCHAR(128) NOT NULL,
    turn_idx    INT NOT NULL,
    role        VARCHAR(32) NOT NULL,
    content     MEDIUMTEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_thread_turn (thread_id, turn_idx)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS checkins (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id   VARCHAR(128) NOT NULL,
    category    VARCHAR(32) NOT NULL,
    content     TEXT NOT NULL,
    value       DOUBLE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_thread_cat (thread_id, category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS reminders (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id     VARCHAR(128) NOT NULL,
    cron          VARCHAR(64) NOT NULL,
    content       TEXT NOT NULL,
    next_fire_at  TIMESTAMP NOT NULL,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    KEY idx_fire (status, next_fire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- [ ] **Step 2: 写 InitDB**

创建 `internal/infra/db.go`：
```go
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
```

- [ ] **Step 3: main.go 启动顺序加 InitDB**

在 `main.go` 的 `main()` 里，`infra.InitModel(ctx)` 之前加：
```go
	if err := infra.InitDB(ctx); err != nil {
		log.Fatal(err)
	}
```
（放在 Load 之后、InitModel 之前。）

- [ ] **Step 4: build 确认**

Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && go build ./...`
Expected: 编译通过（此时还没有 MySQL 在跑，InitDB 只在运行时调，编译不受影响）。

- [ ] **Step 5: 提交**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
git add internal/store/schema.sql internal/infra/db.go main.go
git commit -m "feat(store): MySQL 连接初始化与建表 schema"
```

## Task 3: MySQL CheckPointStore 实现 + 测试

**Files:**
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/store/checkpoint.go`
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/store/checkpoint_test.go`
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/model/state.go`（替换内存 store）
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/internal/agent/builder.go`（确认用 store）

**Interfaces:**
- Consumes: `infra.DB *sql.DB`
- Produces: `store.NewMySQLCheckPoint(db *sql.DB) compose.CheckPointStore`；`model.NewDeepAgentCheckPoint(ctx) compose.CheckPointStore`（改为返回 MySQL 版，签名不变）

- [ ] **Step 1: 写 checkpoint 失败测试**

创建 `internal/store/checkpoint_test.go`：
```go
package store

import (
	"context"
	"testing"
)

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
```
并在 `checkpoint_test.go` 顶部加辅助（用环境变量 DSN 起连）：
```go
import (
	"database/sql"
	"os"
	_ "github.com/go-sql-driver/mysql"
)

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
	t.Cleanup(func() { db.Close() })
	return db
}
```

- [ ] **Step 2: 运行测试确认行为**

Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && go test ./internal/store/ -run TestMySQLCheckPointGetSet -v`
Expected: 编译失败（NewMySQLCheckPoint 未定义）。设环境变量后跳过（无 DB）也可。

- [ ] **Step 3: 实现 checkpoint.go**

创建 `internal/store/checkpoint.go`：
```go
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
```

- [ ] **Step 4: 运行测试（无 MySQL 时 SKIP，视为通过）**

Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && go test ./internal/store/ -run TestMySQLCheckPointGetSet -v`
Expected: PASS（SKIP 也算通过，因无 TEST_MYSQL_DSN）。

- [ ] **Step 5: 替换 model/state.go 的内存 store**

`internal/model/state.go`：删除 `DeepAgentCheckPoint` struct、`deepAgentCheckPoint` 全局变量、`NewDeepAgentCheckPoint` 旧实现。改为：
```go
import (
	"deepAgent/internal/infra"
	"deepAgent/internal/store"
)

// NewDeepAgentCheckPoint 返回 MySQL 后端的 checkpoint store。
// 保持原签名，builder 不用改。
func NewDeepAgentCheckPoint(ctx context.Context) compose.CheckPointStore {
	return store.NewMySQLCheckPoint(infra.DB)
}
```
注意：如果 `infra.DB` 为 nil（DB 未初始化），会在运行期 Set/Get 时报错。main.go 已保证先 InitDB 再 Builder，可接受。为稳妥，builder 编译期不要求 DB 非空。

- [ ] **Step 6: 确认 builder.go 无需改**

`builder.go:79` 已是 `WithCheckPointStore(model.NewDeepAgentCheckPoint(ctx))`，签名不变，无需修改。手动 `grep` 确认：
Run: `cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent && grep -n "NewDeepAgentCheckPoint\|WithCheckPointStore" internal/agent/builder.go`
Expected: 一行 `NewDeepAgentCheckPoint` 调用，未动。

- [ ] **Step 7: build + vet + 提交**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
go build ./... && go vet ./...
git add internal/store/checkpoint.go internal/store/checkpoint_test.go internal/model/state.go
git commit -m "feat(store): MySQL 后端的无状态 CheckPointStore 替换内存版"
```

## Task 4: docker-compose + Dockerfile

**Files:**
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/deploy/docker-compose.yml`
- Create: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/deploy/Dockerfile`
- Modify: `/Users/gd-npc-1029/Documents/CodeBase/deepAgent/.dockerignore`（新建，排除 .venv/output 等）

**Interfaces:**
- Consumes: `internal/store/schema.sql`（mysql 自动执行）、deepAgent 二进制
- Produces: `docker compose up` 一键起 mysql + deepAgent，互连同网段

- [ ] **Step 1: 写 Dockerfile**

创建 `deploy/Dockerfile`（多阶段，先 build 再 slim 运行）：
```dockerfile
# build stage
FROM golang:1.25-alpine AS builder
WORKDIR /src
# 缓存依赖
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/deep-agent .

# runtime stage：需 node(用于 npx tavily-mcp) + uv(用于 python mcp)
FROM node:20-alpine
RUN apk add --no-cache python3 py3-pip bash
    
# python mcp 依赖
WORKDIR /app/mcps/python
COPY mcps/python/pyproject.toml mcps/python/uv.lock ./
RUN pip install --no-cache-dir mcp==1.8.1 pydantic pydantic-settings python-dotenv
WORKDIR /app
COPY --from=builder /out/deep-agent /app/deep-agent
COPY conf/ /app/conf/
COPY internal/prompts/ /app/internal/prompts/
COPY internal/store/schema.sql /app/internal/store/schema.sql
COPY mcps/python/server.py /app/mcps/python/server.py
EXPOSE 8080
CMD ["/app/deep-agent"]
```
注意：python MCP 的 `--directory` arg 在 yaml 里指向宿主路径，容器内要改。本阶段先保证编译/起服务，python MCP 路径在 Task 5 验证时调整 `deep-agent.yaml`（容器内用 `/app/mcps/python`）。tavily-mcp 走 npx 需要 node，已在基础镜像。

- [ ] **Step 2: 写 docker-compose.yml**

创建 `deploy/docker-compose.yml`：
```yaml
services:
  mysql:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: deepagent
      MYSQL_DATABASE: deepagent
    ports:
      - "3306:3306"
    volumes:
      - ../internal/store/schema.sql:/docker-entrypoint-initdb.d/schema.sql
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-uroot", "-pdeepagent"]
      interval: 5s
      timeout: 3s
      retries: 20

  deepagent:
    build:
      context: ..
      dockerfile: deploy/Dockerfile
    depends_on:
      mysql:
        condition: service_healthy
    environment:
      DEEPAGENT_MODE: server
    ports:
      - "8080:8080"
    # 容器内 DSN 用服务名 mysql；yaml 通过挂载或内联配置
    volumes:
      - ../conf/deep-agent.yaml:/app/conf/deep-agent.yaml:ro

volumes:
  mysql-data:
```

- [ ] **Step 3: 写 .dockerignore**

创建 `.dockerignore`（项目根，与 Dockerfile context 对应的 `..`）：
```
.git
.venv
output
*.log
conf/deep-agent.yaml
```
（不把真实 key 的 yaml 烤进镜像，靠 volume 挂载注入。）

- [ ] **Step 4: 准备容器用的 yaml 配置**

把本地 `conf/deep-agent.yaml` 的 `database.dsn` 改为容器内可达：
```yaml
database:
  dsn: "root:deepagent@tcp(mysql:3306)/deepagent?parseTime=true"
```
python MCP 的 args `--directory` 路径改为容器内 `/app/mcps/python`（如已是绝对路径则替换为该值）。

- [ ] **Step 5: 起服务验证**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
colima start          # 若未运行
docker compose -f deploy/docker-compose.yml up -d --build
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs deepagent | tail -20
```
Expected: mysql healthy，deepagent 容器日志显示监听 8080 且无 panic（InitDB 成功连上 mysql，MCP 加载成功）。若 MCP 因路径失败，按日志调整 yaml 里 `--directory` 路径。

- [ ] **Step 6: 提交**

Run:
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
git add deploy/docker-compose.yml deploy/Dockerfile .dockerignore
git commit -m "feat(deploy): docker compose 起 mysql + deepAgent"
```

## Task 5: 端到端验证无状态 checkpoint 中断恢复

**Files:**
- 无新文件，仅验证现有 human_feedback 流程在 MySQL checkpoint 下可跨"进程"恢复

**Interfaces:**
- Consumes: Task 1-4 全部产出
- Produces: 验证结论（无代码产物）

- [ ] **Step 1: 用 curl 触发一次研究请求（AutoAcceptedPlan=false 触发中断）**

```bash
curl -N -X POST http://localhost:8080/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"研究一下2026年主流AI编程助手"}],"auto_accepted_plan":false,"thread_id":"test-thread-1"}'
```
Expected: SSE 流里先有 planner 输出，然后收到 `event: interrupt` 事件（Human Feedback 中断）。此时 MySQL `graph_checkpoint` 表应有一条 `thread_id='test-thread-1'` 记录。验证：
```bash
docker compose -f deploy/docker-compose.yml exec mysql \
  mysql -uroot -pdeepagent deepagent -e "SELECT thread_id, LENGTH(data), updated_at FROM graph_checkpoint;"
```
Expected: 一行 thread_id=test-thread-1，data 非空。

- [ ] **Step 2: 重启 deepagent 容器模拟进程重启** 

```bash
docker compose -f deploy/docker-compose.yml restart deepagent
sleep 3
```

- [ ] **Step 3: 用同 thread_id + accepted 恢复执行**

```bash
curl -N -X POST http://localhost:8080/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"开始执行"}],"auto_accepted_plan":false,"thread_id":"test-thread-1","interrupt_feedback":"accepted"}'
```
Expected: 流不再返回 interrupt，而是继续 researcher/reporter 直到出报告。证明进程重启后用同 thread_id 完整恢复了中断现场——无状态达成。

- [ ] **Step 4: 若失败则用 DEEPAGENT_DEBUG_MCP 排查**

若恢复失败（如 state 字段反序列化错），检查：
- `model/state.go` 的 `init()` 仍调用 `compose.RegisterSerializableType[State]("DeepAgentState")`（跨进程恢复必须）。
- State 结构改动过会让老 checkpoint 反序列化失败——本阶段未改 State 字段，应无碍。

- [ ] **Step 5: 记录验证结论并提交（可选）**

本任务无代码改动，验证通过即 Stage 1 完成。若过程中修改了配置或路径，提交：
```bash
cd /Users/gd-npc-1029/Documents/CodeBase/deepAgent
git add -A
git commit -m "chore: stage1 验证 checkpoint 无状态恢复通过" || echo "无改动跳过"
```

---

## Self-Review（写作后自查）

**1. Spec 覆盖（对照设计 v3 Stage 1）:**
- docker 起 mysql + deepAgent → Task 4 ✓
- MySQL 连接 + 建表 → Task 2 ✓
- MySQL CheckPointStore → Task 3 ✓
- 替换内存 checkpoint → Task 3 Step 5 ✓
- 配置 database/vision_model 段 → Task 1 ✓
- 验证无状态中断恢复 → Task 5 ✓
- 未覆盖：VisionModel 初始化（llm.go）属 Stage 4 识图阶段用，本阶段只加配置，不初始化——符合阶段划分。

**2. 占位符扫描:** 无 TBD/TODO/"implement later"。每个代码 step 含完整代码。

**3. 类型一致性:**
- `NewMySQLCheckPoint(db *sql.DB) compose.CheckPointStore` —— Task 3 定义，Task 3 Step 5 使用，签名一致。
- `infra.InitDB(ctx) error` —— Task 2 定义，main.go 调用，一致。
- `conf.DatabaseConfig{DSN string}` —— Task 1 定义，Task 2 `conf.App.Database.DSN` 使用，一致。
- `infra.DB *sql.DB` —— Task 2 定义，Task 3 通过 `infra.DB` 引用，一致。
- `model.NewDeepAgentCheckPoint(ctx)` —— 签名保持不变，builder 不动。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-25-checkin-assistant-stage1-docker-storage-checkpoint.md`. Two execution options:

1. **Subagent-Driven (recommended)** - 每个 Task 派一个新 subagent 实现，任务间两阶段 review，迭代快。
2. **Inline Execution** - 当前会话内按 executing-plans 批量执行，带 checkpoint 复审。

Which approach?

后续阶段（Stage 2-6：checkin 工具/识图 agent/Coordinator 意图分类/定时提醒/微信）各自续写计划，不塞进本计划。
