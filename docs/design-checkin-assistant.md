# 自律打卡 AI 助手 设计方案 (v3 — 调研+无状态+docker+Coordinator分类)

基于 deepAgent（deer-go 复现）现有实现，新增「自律打卡」业务场景。覆盖学习/饮食/运动。
原则：**不重复造轮子（用 eino 原生能力）、服务无状态、docker 统一启动**。

## v3 相对 v2 的变更
- 意图分类不再用独立 intent 节点，改为 **Coordinator 兼任分类器（多 handoff tool）**，对齐 deer-go。
- 部署统一 docker compose：**MySQL + deepAgent 一起容器化**，一次 `up` 全起。
- deepAgent 镜像化，调试改动需 rebuild。

## 一、需求锁定

| 维度 | 决策 |
|---|---|
| 场景 | 学习/饮食/运动 自律打卡 |
| 图的复用 | 一张总图 + 状态驱动分支：研究类走现有研究子图；打卡/识图走新增子图 |
| 识图 | 独立识图 agent，agnes-2.0-flash 多模态，**独立可切换配置** |
| 识图输入 | console 阶段本地路径模拟（`打卡早餐 /tmp/food.jpg`） |
| 存储 | 全 MySQL（业务表 + 图 checkpoint） |
| 定时提醒 | 调度做成 agent tool（`create_reminder`）+ 无状态 ticker 扫表触发 |
| 多模态模型 | agnes-2.0-flash 本身支持图片理解，单独配置项 |
| 入口 | 先 console，后接微信（`@tencent-weixin/openclaw-weixin-cli`） |
| **服务形态** | **无状态**：任何进程可接管任意 thread_id 会话；状态全部外置 MySQL |

## 二、eino 能力调研结论（决定设计的关键事实）

| 能力点 | eino 现状 | 对设计的影响 |
|---|---|---|
| CheckPointStore 接口 | ✅ 已提供（`core.CheckPointStore`，`Get/Set([]byte)`）。**仅在中断时落盘**，正常轮次不落 | checkpoint 只管中断现场；对话历史必须单独建表 |
| 中断恢复链路 | ✅ `InterruptAndRerun` + `WithCheckPointID` + `WithStateModifier` + `RegisterSerializableType`，天然支持跨进程恢复 | 直接复用现有 human_feedback + handler 范式 |
| 官方 checkpoint 后端实现 | ❌ 未提供（map 内存实现是 sample） | 自己实现 MySQL 后端的 CheckPointStore |
| 状态驱动路由 | ✅ `NewGraphBranch` + `ProcessState` 读 `state.Goto`，eino 自身 ReAct 也用这范式 | 意图判定做成图内节点写 Goto，零自造路由层 |
| 子图/agent 混挂 | ✅ `AddGraphNode`(AnyGraph) 与 `AddLambdaNode`(AnyLambda) 可同图；ReAct 还有 `ExportGraph()` 直接当节点 | 研究子图用 GraphNode，轻交互/识图用 LambdaNode 或直接 Generate |
| 自定义 tool | ✅ `utils.InferTool(func, struct)` 自动推导 schema | 用 InferTool，不手写 BaseTool 接口 |
| 多模态消息 | ✅ `schema.Message.UserInputMultiContent` + `MessageInputImage`（旧 `MultiContent` 已废弃） | 识图用此构造图片消息 |
| ReAct 直接调用 | ✅ `agent.Generate(ctx, msgs)`，单 agent 场景不必建子图 | 轻交互/识图可直接调，不进总图 |
| 定时调用 | ❌ eino 无 cron（已确认） | 自己用 robfig/cron 做扫表 ticker |

## 三、整体架构（无状态）

```
                        ┌──────────── MySQL (共享外置状态) ────────────┐
                        │  checkins / plans / reminders / messages     │
                        │  graph_checkpoint(thread_id BLOB)            │
                        │  intent_classifier 缓存(可选)                │
                        └──────────────────────────────────────────────┘
                                   ▲                          ▲
                                   │读写                      │扫表
        console/微信请求            │                          │
用户 ──────────────────▶ [Service 进程 A]            [Service 进程 N]
                        │  1. 载入/创建 thread_id                          │
                        │  2. runnable.Stream(ctx, START,                 │ 各进程都跑一个
                        │       WithCheckPointID(thread_id),             │ 无身份 ticker：
                        │       WithStateModifier(...))                  │ 每分钟扫 reminders
                        │  3. 中断 → 存 checkpoint → 返回 interrupt 事件   │ 表到点的记录，
                        │  4. 正常 → 读模型流式输出                         │ 抢锁处理 + 推送
                        ▼
                     Response (SSE/打印)
```

**无状态的三个保证**：
1. 进程无任何会话级内存态（除 eino 单次请求内 ctx 透传）。
2. thread_id 是会话身份的唯一标识，checkpoint 与业务表都以它为 key。
3. 任意进程挂掉重启，用 thread_id 可完整恢复中断现场与历史。

### 总图编排（一张图，状态驱动分流，Coordinator 兼任意图分类）

```
START → [Coordinator 子图] ──(router 写 state.Goto)──▶ branch(agentHandOff 读 Goto)
                                                            ├── research  → 现有研究子图(不动) → END
                                                            ├── checkin   → 轻交互 ReAct agent    → END
                                                            ├── vision    → 识图 ReAct agent       → END
                                                            └── END       → 闲聊直接答
```
- **Coordinator 兼任意图分类器**（对齐 deer-go 范式）：Coordinator 是绑定多个 handoff 工具的 ChatModel：
  - `hand_to_researcher` → 调它则 `state.Goto = research`
  - `hand_to_checkin`    → 调它则 `state.Goto = checkin`
  - `hand_to_vision`     → 调它则 `state.Goto = vision`
  - **不调任何 tool** → 闲聊/拒绝/澄清，router `state.Goto = END` 直接结束
- 分类信号 = "调了哪个 tool"（复用 tool call 机制，零额外解析，天然结构化）。
- Coordinator 的 prompt 仿 deer-go 的 coordinator.md「Request Classification」结构，把三路 handoff 规则写清。
- router 读第一个 tool call 名前缀映射 Goto；**图片兜底**：若用户消息含 `UserInputMultiContent` 但模型没调 `hand_to_vision`，router 强制 `Goto = vision`（稳）。
- branch 复用现有 `agentHandOff`（builder.go 已有），**不新增路由层/intent 节点**。
- 现有研究子图整套保留不动，作为 `research` 分支的 GraphNode。
- 轻交互/识图：用 `agent.Generate` 包成 LambdaNode，或 `ExportGraph()` 当 GraphNode。
- 总图编译带 `WithCheckPointStore(mysqlStore)` + `WithNodeTriggerMode(AnyPredecessor)`。

## 四、关键设计决策

### 1. 无状态 checkpoint：自实现 MySQL CheckPointStore
- 实现 `core.CheckPointStore`：`Get/Set(thread_id, []byte)`，后端表 `graph_checkpoint(thread_id VARCHAR PK, data LONGBLOB, updated_at)`。
- `[]byte` 是 eino 序列化器产物，**直接当不透明 blob 存**，不二次序列化。
- 替换 deepAgent 现有 `model.NewDeepAgentCheckPoint`（包级单例 map，重启即丢）为这个注入实例。
- `model.State` 的 `init()` 已 `RegisterSerializableType`，保持（跨进程反序列化必须）。

### 2. 对话历史独立建表（不能依赖 checkpoint）
- eino 只在中断时落 checkpoint，正常跑完不落 → 靠 checkpoint 存历史会丢轮次。
- `messages(thread_id, turn_idx, role, content, created_at)` append 写。
- 每轮：用户消息先入 messages 表，再通过 `WithStateModifier` 注入 checkpoint 的 State 让图续跑。

### 3. 轻交互场景：ReAct agent + `InferTool` 工具，直接 Generate
- 不为它建子图（调研确认单 agent 可直接 `agent.Generate`）。若纳总图则用 `ExportGraph()`/`AnyLambda`。
- 工具用 `utils.InferTool` 从 Go 函数+struct 生成：`record_checkin`/`query_checkin`/`create_reminder`/`list_reminders`/`delete_reminder`/`get_summary`。
- 工具内部读写 MySQL。工具与现有 MCP tool 同为 `tool.BaseTool`，可在同一 `ToolsConfig` 混用。

### 4. 识图 agent：多模态 ReAct，独立配置
- 用 `schema.Message.UserInputMultiContent` + `MessageInputImage`(URL 或 Base64) 构造图片消息（**不用废弃的 MultiContent**）。
- 模型从 `conf` 的 `vision_model` 段读（可单独换），初值复用主模型。
- console 阶段：本地图片读为 Base64 data url 注入。
- 把 `record_checkin(category=diet)` 作为 tool 给识图 agent，让它自主写库。
- StreamToolCallChecker 沿用 coder.go 的"扫到任意 chunk 有 ToolCalls 即 true"版本（默认只看首 chunk 会误判）。

### 5. 定时提醒：agent tool 建提醒 + 无状态 ticker 触发
- **创建**：用户"每天8点提醒喝水"→ checkin agent 调 `create_reminder(cron, content)` tool → 写 `reminders(user_id, cron, content, next_fire_at, status)` 表，算好下次触发时间。
- **触发**：每个 Service 进程跑一个**无身份 ticker**（robfig/cron 每分钟触发），扫 `next_fire_at <= now AND status=pending`，用 `UPDATE ... WHERE status=pending`（或 SELECT FOR UPDATE）**抢锁**，抢到的进程负责推送（console 打印 / 后续微信消息），并按 cron 算下次 next_fire_at。
- 抢锁保证多副本不重复触发；进程可随时重启，状态全在表里。
- cron 解析用 robfig/cron 库（eino 无此能力，需自引，调研已确认）。

### 6. 配置独立化（conf 改造）
```yaml
model:
  default_model: "agnes-2.0-flash"
  api_key: "..."
  base_url: "..."
  vision_model:              # 识图 agent 专用，初值复用上方；缺失回退到主模型
    model: "agnes-2.0-flash"
    api_key: "..."
    base_url: "..."
database:
  dsn: "user:pass@tcp(127.0.0.1:3306)/deepagent?parseTime=true"
setting:
  max_plan_iterations: 1
  max_step_num: 3
  enable_background_investigation: false
```
`infra.InitModel` 增加 `VisionModel` 初始化（缺失回退 ChatModel）；新增 `infra.InitDB`。

## 五、目录结构变化

```
deepAgent/
  internal/
    agent/
      builder.go            # 改：总图加 Coordinator 多 handoff tool + 三分支；研究子图不动
      coordinator.go        # 改：hand_to_planner 扩展为 hand_to_researcher/checkin/vision
      checkin_agent.go      # 新增：轻交互 ReAct agent 构造
      vision_agent.go       # 新增：识图 ReAct agent 构造
    tool/
      checkin_tools.go      # 新增：InferTool 注册 record_checkin/query_checkin 等
      reminder_tools.go     # 新增：InferTool 注册 create_reminder/list/delete
    store/
      mysql.go              # 新增：DB 连接 + 业务表 CRUD
      checkpoint.go         # 新增：MySQL 后端的 CheckPointStore 实现
      reminder_ticker.go    # 新增：无状态扫表 ticker
      schema.sql            # 新增：建表 DDL
    model/
      state.go              # 改：扩展 State(加 UserID 等)，注册保持
    prompts/
      coordinator.md        # 改：三路 handoff 分类规则（仿 deer-go）
      checkin_coach.md      # 新增：自律教练 system prompt
      vision_diet.md        # 新增：识图 agent system prompt
  conf/
    config.go               # 改：加 vision_model / database 段
    deep-agent.yaml.example # 改：加对应示例；DSN 指向容器名 mysql
  deploy/
    docker-compose.yml      # 新增：mysql + deepAgent 两服务
    Dockerfile              # 新增：deepAgent 镜像（含 mcps/python 依赖）
  main.go                   # 改：console 入口走总图(含 Coordinator 分流)
```

## 六、部署：docker compose 统一启动（v3 新增）

- **MySQL + deepAgent 一起容器化**，一次 `docker compose up` 全起。
- `deploy/docker-compose.yml`：
  - `mysql` 服务：`mysql:8`，挂载 `store/schema.sql` 到 `/docker-entrypoint-initdb.d/` 自动建表，数据卷持久化。
  - `deepagent` 服务：build 本地 Dockerfile，`depends_on: mysql`，DSN 指向 `mysql:3306`（容器网络），端口映射 8080。
- `deploy/Dockerfile`：多阶段构建（go build 二进制）；需把 `mcps/python` 的 uv 环境一并打入（Coder 依赖），或保留 python + tavily-npx 依赖。
- 调试改动需 `docker compose build deepagent` 重建镜像后再 up。
- 配置 `database.dsn` 走容器名；本地 `go run` 时走 `127.0.0.1`（用环境变量区分或起单独 profile）。
- colima 已就绪（v0.10.3 + docker 29.5.2），实现时确认 `docker compose` 插件可用。

## 七、复用 vs 新增（不造轮子版）

## 八、分阶段实现路线（每阶段可运行）

1. **docker 基建 + 存储**：`deploy/docker-compose.yml`(mysql+deepagent) + `deploy/Dockerfile` + `store/schema.sql`(建表) + `store/mysql.go`(连库+CRUD) + `conf` 加 database/vision_model 段。`docker compose up` 起 MySQL，程序能连库。
2. **无状态 checkpoint**：`store/checkpoint.go`(MySQL CheckPointStore) + 替换 builder 的内存 store；验证现有研究流程在新 checkpoint 下仍可中断恢复(human_feedback)。
3. **打卡工具 + 轻交互 agent**：`InferTool` 实现 checkin 工具集；`checkin_agent.go`；console"今天跑5km"写库。
4. **识图 agent**：`vision_agent.go` 多模态消息 + vision_model；console"打卡早餐 /tmp/food.jpg"识图写库。
5. **Coordinator 接意图分类**：扩展 Coordinator 为多 handoff tool（research/checkin/vision）+ 改 coordinator.md prompt + builder 接三分支；console 全场景可用。
6. **定时提醒 ticker**：reminder 工具 + 无状态扫表 ticker；console 验证到点提醒 + 多副本不重复。
7. （后续）接入微信。

## 九、风险与待定（不阻塞设计）

- **`docker compose` 插件**：目前未输出版本，实现时 `colima start` 后确认；不可用则用 `docker-compose` v1 或补插件。
- **deepAgent 镜像含 MCP 依赖**：Dockerfile 要打入 uv(python MCP) + npx(node, tavily MCP)，构造稍复杂，第1阶段细化。
- **多模态 API 输入格式**：agnes 走 OpenAI 协议，image 用 base64 data url 还是 file url，第4阶段用一张测试图确认。
- **console 提醒抢占终端**：ticker 到点用独立前缀打印，接微信后改消息通道。
- **多副本 ticker 抢锁**：用 `UPDATE reminders SET status='running' WHERE id=? AND status='pending'` 判影响行数抢锁；MySQL 行锁足够。
- **微信接入**：`openclaw-weixin-cli` 的消息格式/回调实现阶段再细究。

## 十、调研依据源码路径
- CheckPointStore 接口：eino@v0.9.7/internal/core/interrupt.go:27
- 状态驱动路由：deepAgent builder.go:15-24 agentHandOff
- Coordinator 单 handoff tool 范式（待扩展为多 tool）：deepAgent coordinator.go + deer-go coordinator.md
- InferTool：eino@v0.9.7/components/tool/utils/invokable_func.go
- ReAct 多模态/直接调用：eino@v0.9.7/flow/agent/react/react.go:480
- 多模态消息：eino@v0.9.7/schema/message.go:177/207
- ExportGraph：eino@v0.9.7/flow/agent/react/react.go:489
