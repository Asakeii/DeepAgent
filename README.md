# deepAgent

基于 CloudWeGo Eino 的多 Agent 深度研究工作流，复刻并演进 `deer-go` 的设计：用户提问 → 规划 → 联网检索/代码计算 → 汇总报告。在此基础上向「自律打卡 AI 助手」（学习/饮食/运动）场景扩展。

## 工作流（已实现，已端到端验证）

```text
用户输入
  -> Coordinator    判断闲聊 / 研究任务（hand_to_planner tool call）
  -> Planner        生成结构化 Plan JSON（json_schema 强约束）
  -> Human Feedback 中断确认计划（支持 checkpoint 跨进程恢复）
  -> ResearchTeam   分派未完成 step
  -> Researcher     ReAct + 搜索 MCP（tavily）
  -> Coder          ReAct + Python MCP
  -> Reporter       汇总带引用的最终报告
```

设计要点（详见 `docs/rebuild-deer-go-guide.md`）：
- 全局一份共享 `State`，节点间通过 `state.Goto` 轻量路由（声明意图而非调用，路由集中）。
- 每个节点是 `load → agent → router` 三段式子图，总图复用 `agentHandOff` 状态驱动分支。
- Planner 输出 `Plan` JSON 是后续执行的契约；Researcher/Coder 逐个填充 `Step.ExecutionRes`。
- Human Feedback 中断用 eino v0.9.7 标准 `compose.Interrupt`，配合 `CheckPointStore` 支持人机协同。

## 目录结构

- `conf/`: YAML 配置（模型/MCP/数据库/运行参数）。
- `internal/model/`: 共享 `State`、`Plan`、`Step`、请求/响应模型。
- `internal/infra/`: 模型初始化、MCP 客户端、prompt 加载、SSE 回调、**MySQL 连接（`infra.DB`）**。
- `internal/agent/`: Eino 总图与各 agent 子图节点（builder/coordinator/planner/researcher/coder/reporter/research_team/human_feedback）。
- `internal/store/`: **MySQL 持久化层**——`schema.sql` 建表 DDL、`checkpoint.go` 无状态 CheckPointStore。
- `internal/prompts/`: 各 agent 的系统提示词（与 deer-go 字节级对齐）。
- `internal/handler/`: HTTP SSE 接口（`/chat/stream`）。
- `mcps/python/`: Python REPL MCP server（Coder 用，`execute_python` 等）。
- `deploy/`: **docker-compose + Dockerfile**，一键起 MySQL + deepAgent。
- `docs/`: 复现指南、设计文档、实施计划。

## 运行

### 方式一：docker compose（推荐，含无状态 MySQL + checkpoint）

```bash
cp conf/deep-agent.yaml.example conf/deep-agent.yaml   # 填入真实 model/api_key、tavily key、database.dsn
colima start                                            # macOS 上启动 docker
docker compose -f deploy/docker-compose.yml up -d --build

# deepagent 监听 :8088（宿主，容器内 8080）；MySQL 自动建表（schema.sql 挂载到 /docker-entrypoint-initdb.d）
```

### 方式二：本地 console

```bash
cp conf/deep-agent.yaml.example conf/deep-agent.yaml
./run.sh
```

> console 模式用 `r.Stream`（非 `Invoke`）——图内 chat model 节点走流式，Coordinator 输出 tool call 后不以字符串结束，Invoke 会死锁。

## 无状态与可恢复

服务无状态，状态全部外置 MySQL：
- **图中断现场** → `graph_checkpoint(thread_id, data BLOB)`，实现 eino `compose.CheckPointStore`。`[]byte` 是 eino 序列化器产物，直接当 blob 存。
- **业务数据** → `messages` / `checkins` / `plans` / `reminders` 表。
- 中断后用同 `thread_id` + `interrupt_feedback`（`accepted`/`edit_plan`）即可从断点恢复，进程重启不丢现场。

验证：触发 interrupt → `docker compose restart deepagent` → 同 thread_id + accepted 续跑至 Researcher/Reporter 成功。

## 配置（`conf/deep-agent.yaml`）

```yaml
mcp:
  servers:
    tavily: { command: "npx", args: ["-y","tavily-mcp@0.1.3"], env: { TAVILY_API_KEY: "..." } }
    python: { command: "uv", args: ["--directory","./mcps/python","run","server.py"] }
model:
  default_model: "..."
  api_key: "..."
  base_url: "..."
  vision_model: { ... }        # 识图 agent 专用，缺失回退主模型（后续阶段启用）
setting:
  max_plan_iterations: 1
  max_step_num: 3
  enable_background_investigation: false
database:
  dsn: "root:deepagent@tcp(mysql:3306)/deepagent?parseTime=true"   # 容器内用服务名 mysql
```

## 路线（自律打卡助手扩展）

在现有研究图基础上扩展（设计见 `docs/design-checkin-assistant.md`）：
- ✅ Stage 1: docker 部署 + MySQL 存储 + 无状态 checkpoint + 中断恢复
- ✅ Stage 2: 打卡工具（`InferTool`）+ 轻交互 ReAct agent + 跨会话记忆
- ✅ Stage 3: 多模态识图 agent（食物图片→能量摄入→写库）
- ✅ Stage 4: Coordinator 多 handoff tool 统一入口路由
- ✅ Stage 5: 定时提醒（agent tool + 无状态扫表 ticker）
- ✅ Stage 6: 微信原生回调 + MCP bridge

## 微信接入

deepAgent 提供原生微信公众号回调接口（`/wechat/callback`），无需 OpenClaw 等中间层。

**前提条件：** 微信公众号（服务号/订阅号），服务有公网域名。

**配置步骤：**
1. 设置环境变量 `WECHAT_TOKEN`（docker compose 中已预留）
2. 在微信公众号后台配置服务器 URL：`https://your-domain/wechat/callback`
3. Token 填入 docker compose 中的 `WECHAT_TOKEN` 值
4. 微信服务器发送 GET 请求验证 → SHA1 签名校验 + echostr 回传
5. 验证通过后，用户消息通过 POST 回调到达，走 Coordinator 自动路由处理

**微信消息流程：** 用户发消息 → 微信服务器 POST XML → `/wechat/callback` → Coordinator 路由 → XML 回复 → 微信推送

## 外部接入（MCP bridge）

deepAgent 同时暴露 MCP bridge（`:8090`），支持通过 MCP 协议被外部 AI 网关调用：
- `GET /sse` — MCP SSE 连接端点
- `POST /message` — MCP JSON-RPC 消息端点
- 暴露 `chat` 和 `list_capabilities` 两个工具

任何支持 MCP 客户端的框架（OpenClaw、LangChain、自研 agent 等）均可通过此端口调用 deepAgent。

## 参考

- `deer-go`: `eino-examples/flow/agent/deer-go`（CloudWeGo 官方复刻 DeerFlow）。
- 更新日志与实施计划见 `docs/superpowers/plans/`。
