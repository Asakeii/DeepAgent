# deepAgent 服务入口瘦身架构优化设计

## 1. 背景

当前 `deepAgent` 已经具备较完整的 Agent 产品雏形：研究工作流、打卡助手、图片识别、提醒调度、微信接入、OpenAI-compatible API、SSE 前端流式展示等能力都已经跑通。

但服务入口开始变胖，核心集中在：

- `main.go` 负责初始化、HTTP 路由、CORS、静态文件、CLI 调试入口。
- `internal/handler/deer.go` 同时承担请求解析、SSE 连接管理、提醒连接注册、图片分支、图构建、中断恢复、checkin 路由、消息持久化。
- `handler/wechat.go`、`handler/openai.go` 和 `/chat/stream` 存在重复的 Agent 调用路径。
- `agent.CheckinThreads` 使用进程内 `sync.Map` 作为 Coordinator 到 handler 的跨层信号，单机可用，但不利于多副本、测试和故障恢复。

这类问题在大型后端系统中通常被称为 Fat Controller / Fat Handler。它短期开发快，但长期会导致：

- 协议层和业务层耦合，SSE、微信、OpenAI API 难以共享逻辑。
- 新增入口时重复拼装 Agent、消息、持久化、错误处理。
- 单元测试困难，只能做端到端测试。
- 多实例部署时进程内状态失效。
- 业务行为散落在 handler 中，后续产品化和权限治理成本变高。

## 2. 优化目标

本次设计不推翻现有 Eino Graph 方案，而是围绕服务入口瘦身，补齐一层应用架构。

目标：

1. Handler 只做协议适配，不写业务编排。
2. Agent 运行、checkin 分流、提醒事件、消息持久化由 Service 层接管。
3. SSE、微信、OpenAI-compatible API 复用同一套应用用例。
4. 消除跨层 `sync.Map` 路由信号，改为显式返回运行结果。
5. 为后续鉴权、多租户、审计、评测、观测打基础。

非目标：

- 不重写现有 Eino Graph。
- 不替换 React 前端。
- 不一次性引入复杂微服务。
- 不改变当前 MySQL + Redis 基础设施选择。

## 3. 主流方案参考

主流大厂 Agent / AI 应用后端通常会采用类似分层：

```text
Transport / Adapter Layer
  HTTP, SSE, WebSocket, OpenAI-compatible API, WeChat, CLI

Application Service Layer
  ResearchService, CheckinService, ReminderService, RunService

Domain / Agent Runtime Layer
  Eino Graph, ReAct Agent, Planner, Tool Calling, State Router

Infrastructure Layer
  MySQL, Redis, LLM Client, MCP Client, Object Storage, Logger, Metrics
```

核心思想：

- 入口层不懂业务，只负责把外部协议转换为内部命令。
- 应用层表达业务用例，例如 `StartChatRun`、`ResumeResearchPlan`、`RunCheckinTurn`。
- Agent Runtime 层只关心图和工具，不关心 HTTP/SSE/微信。
- 基础设施通过接口注入，减少全局变量对测试和多环境的影响。
- 所有运行事件归一化，前端流、微信回复、审计日志都来自同一份事件。

## 4. 目标架构

### 4.1 总体结构

```text
                      ┌─────────────────────────────┐
                      │         frontend            │
                      │       SSE / HTTP API        │
                      └──────────────┬──────────────┘
                                     │
┌────────────────────────────────────▼────────────────────────────────────┐
│                         Transport Layer                                  │
│                                                                          │
│  handler.ChatStream  handler.Wechat  handler.OpenAI  CLI                 │
│                                                                          │
│  职责：解析请求、鉴权、协议响应、错误码映射、流式写出                     │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │ App Request / App Event
┌────────────────────────────────────▼────────────────────────────────────┐
│                       Application Service Layer                           │
│                                                                          │
│  ChatService                                                              │
│    - RunTurn                                                              │
│    - ResumeTurn                                                           │
│    - AnalyzeImage                                                         │
│                                                                          │
│  ResearchService                                                          │
│    - BuildGraph                                                           │
│    - RunResearch                                                          │
│    - ResumePlan                                                           │
│                                                                          │
│  CheckinService                                                           │
│    - RunCheckin                                                           │
│    - RecordImageAnalysis                                                  │
│                                                                          │
│  ReminderService                                                          │
│    - RegisterConnection                                                   │
│    - EmitScheduledEvents                                                  │
│    - List / Cancel / Toggle                                               │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │ Domain Command
┌────────────────────────────────────▼────────────────────────────────────┐
│                         Agent Runtime Layer                              │
│                                                                          │
│  Eino Graph: Coordinator / Planner / Human / ResearchTeam / Reporter      │
│  ReAct Agent: Checkin Agent                                               │
│  Tools: Search / Python / Checkin / Reminder / Vision                     │
│                                                                          │
│  职责：Agent 决策、工具调用、状态转移、结果生成                           │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │ Repository / Client Interface
┌────────────────────────────────────▼────────────────────────────────────┐
│                         Infrastructure Layer                             │
│                                                                          │
│  MySQL: messages / checkpoint / checkins / runs / events                  │
│  Redis: reminder queue / pending events                                   │
│  LLM: ChatModel / PlanModel / VisionModel                                 │
│  MCP: Tavily/SearXNG / Python                                             │
│  Logger / Metrics / Tracing                                               │
└──────────────────────────────────────────────────────────────────────────┘
```

### 4.2 推荐目录

```text
internal/
  app/
    chat_service.go
    research_service.go
    checkin_service.go
    reminder_service.go
    event.go
    request.go
    response.go

  handler/
    chat_stream.go
    wechat.go
    openai.go
    sessions.go
    reminders.go

  agent/
    builder.go
    coordinator.go
    planner.go
    checkin_agent.go
    ...

  repo/
    message_repo.go
    checkpoint_repo.go
    run_event_repo.go
    checkin_repo.go

  infra/
    db.go
    redis.go
    llm.go
    mcp.go
    logger.go

  scheduler/
    scheduler.go
    registry.go
    tools.go
```

说明：

- `handler`：协议适配。
- `app`：业务用例编排，是本次瘦身重点。
- `agent`：保留 Eino Graph 和 Agent 能力。
- `repo`：面向业务语义的存储接口，逐步替代 handler 直接调用 `store` 或 `infra.DB`。
- `infra`：实际基础设施连接和初始化。

## 5. 核心改造点

### 5.1 Handler 从业务编排中退出

当前 `/chat/stream` 的职责过多。目标是变成：

```go
func ChatStream(w http.ResponseWriter, r *http.Request) {
    req, err := decodeChatStreamRequest(r)
    if err != nil {
        writeSSEError(w, err)
        return
    }

    stream := sse.NewWriter(w)
    events, err := app.Chat.RunTurn(r.Context(), req)
    if err != nil {
        writeSSEError(w, err)
        return
    }

    for event := range events {
        stream.Write(event.Name, event.Payload)
    }
}
```

Handler 保留职责：

- JSON/XML 解析。
- 鉴权和用户身份注入。
- HTTP 状态码和 SSE 事件写出。
- 请求大小限制、超时设置。

Handler 移除职责：

- 直接构建 Eino Graph。
- 判断图片/checkin/research 具体业务路径。
- 直接写 messages。
- 操作 checkpoint。
- 维护提醒连接注册细节。

### 5.2 引入 ChatService 作为统一入口

新增 `internal/app/chat_service.go`：

```go
type ChatService struct {
    Research  *ResearchService
    Checkin   *CheckinService
    Reminder  *ReminderService
    Messages  MessageRepository
    EventRepo RunEventRepository
}

type RunTurnRequest struct {
    UserID                       string
    ThreadID                     string
    Messages                     []*schema.Message
    AutoAcceptedPlan             bool
    InterruptFeedback            string
    ImageBase64                  string
    MaxPlanIterations            int
    MaxStepNum                   int
    EnableBackgroundInvestigation *bool
}

type RunTurnResult struct {
    Events <-chan AppEvent
}
```

`ChatService` 负责做顶层业务分派：

```text
RunTurn
  -> validate request
  -> normalize defaults
  -> if image: CheckinService.AnalyzeImage
  -> else: ResearchService.RunGraph
       -> if route_to_checkin: CheckinService.RunTurn
       -> else: persist research final
  -> return normalized events
```

这样微信、SSE、OpenAI API 都可以调用同一个 `RunTurn`。

### 5.3 ResearchService 接管 Eino Graph

新增 `internal/app/research_service.go`：

```go
type ResearchService struct {
    GraphFactory GraphFactory
    Messages     MessageRepository
    Checkpoints  compose.CheckPointStore
    EventSink    EventSink
}
```

职责：

- 构造 `model.State`。
- 调用 `agent.Builder`。
- 注入 checkpoint id。
- 处理 interrupt。
- 把 `LoggerCallback` 产生的事件转换成 `AppEvent`。
- 返回最终状态或运行结果。

重要改造：让 Coordinator 的 checkin 决策回到显式结果中。

当前：

```text
Coordinator -> agent.CheckinThreads.Store(threadID, true)
handler EOF -> LoadAndDelete(threadID)
```

目标：

```text
Coordinator -> state.RouteToCheckin = true
ResearchService -> 读取最终 State / GraphResult
ChatService -> if result.RouteToCheckin { CheckinService.RunTurn(...) }
```

如果 Eino 当前不方便直接读取最终 State，可以先过渡为：

- `LoggerCallback` 或 `ResearchService` 内部 callback 捕获 `RouteToCheckin`。
- 仍避免跨包全局 `sync.Map`。
- 最终再升级为图内 checkin 节点。

### 5.4 CheckinService 独立成长期助手用例

新增 `internal/app/checkin_service.go`：

```go
type CheckinService struct {
    AgentFactory CheckinAgentFactory
    Messages     MessageRepository
    Checkins     CheckinRepository
    EventSink    EventSink
}
```

职责：

- 加载最近历史。
- 追加用户消息。
- 调用 ReAct checkin agent。
- 收集 reminder scheduled 事件。
- 持久化助手回复。
- 返回标准 `AppEvent`。

这样 `/v1/chat/completions` 和微信不再自己手动拼 checkin 逻辑。

### 5.5 事件协议归一化

新增 `internal/app/event.go`：

```go
type AppEvent struct {
    RunID     string
    ThreadID  string
    UserID    string
    Name      EventName
    Agent     string
    Payload   any
    CreatedAt time.Time
}

type EventName string

const (
    EventAgent            EventName = "agent"
    EventPlan             EventName = "plan"
    EventInterrupt        EventName = "interrupt"
    EventToolCall         EventName = "tool_calls"
    EventToolCallChunk    EventName = "tool_call_chunks"
    EventToolCallResult   EventName = "tool_call_result"
    EventMessageChunk     EventName = "message_chunk"
    EventFinalMessage     EventName = "final_message"
    EventReminderScheduled EventName = "reminder_scheduled"
    EventReminderFired    EventName = "reminder"
    EventError            EventName = "error"
)
```

收益：

- SSE 只是把 `AppEvent` 写出。
- 微信可以消费最终 `final_message` 或 `message`。
- OpenAI API 可以消费最终文本。
- run event 可以落库做审计和回放。
- 前后端事件协议有统一版本。

### 5.6 Repository 替代 handler 直接写库

当前 `handler` 和 `agent` 通过 `infra.AppendMessageForCheckin` 间接写库，但语义混杂，研究消息也调用 `AppendMessageForCheckin`。

建议新增：

```go
type MessageRepository interface {
    Append(ctx context.Context, msg MessageRecord) error
    AppendMany(ctx context.Context, msgs []MessageRecord) error
    Recent(ctx context.Context, threadID string, limit int) ([]*schema.Message, error)
    ListThreads(ctx context.Context, userID string, limit int) ([]ThreadInfo, error)
}
```

同时改造 messages 表：

```sql
ALTER TABLE messages
  ADD COLUMN user_id VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN run_id VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN source VARCHAR(32) NOT NULL DEFAULT 'chat';

CREATE INDEX idx_user_thread ON messages(user_id, thread_id, id);
```

并逐步废弃 `MAX(turn_idx)+1` 的写法，避免同一 thread 并发请求时序号冲突。

推荐排序方式：

- 用自增 `id` 表示写入顺序。
- `turn_idx` 如果仍要保留，则在事务里维护，或改为由应用层 run/turn 模型生成。

## 6. 改造后的调用链

### 6.1 SSE 请求

```text
frontend
  -> POST /chat/stream
  -> handler.ChatStream
  -> app.ChatService.RunTurn
  -> ResearchService.RunGraph
  -> AppEvent channel
  -> handler writes SSE
```

### 6.2 微信请求

```text
WeChat callback
  -> handler.WechatCallback
  -> app.ChatService.RunTurn
  -> consume final AppEvent
  -> reply XML
```

### 6.3 OpenAI-compatible 请求

```text
POST /v1/chat/completions
  -> handler.OpenAICompatible
  -> app.ChatService.RunTurn
  -> consume final AppEvent
  -> OpenAI response JSON
```

### 6.4 图片识别

```text
POST /chat/stream with image_base64
  -> handler.ChatStream
  -> app.ChatService.RunTurn
  -> CheckinService.AnalyzeImage
  -> persist image analysis/checkin
  -> AppEvent final message
```

## 7. 分阶段落地计划

### Phase 1：低风险抽取 Handler 逻辑

目标：不改变外部 API，只把业务逻辑搬出去。

任务：

- 新建 `internal/app/event.go`。
- 新建 `internal/app/chat_service.go`。
- 把 `/chat/stream` 中的默认值处理、图片分支、graph 运行、checkin 分流迁移到 `ChatService`。
- Handler 保持 SSE 写出逻辑。
- 保持现有前端事件协议不变。

验收：

- `go test ./...` 通过。
- `frontend npm run build` 通过。
- 原 `/chat/stream` 行为不变。

### Phase 2：拆 ResearchService / CheckinService

目标：让两个业务用例各自独立。

任务：

- 新建 `ResearchService`，接管 Eino Graph 构建与中断恢复。
- 新建 `CheckinService`，接管打卡历史加载、消息写入、ReAct 调用。
- `handler/openai.go` 改为调用 `CheckinService` 或 `ChatService`，不再直接调 `agent.RunCheckin`。
- `handler/wechat.go` 改为调用 `ChatService`，不再自己构建 graph。

验收：

- 微信、OpenAI-compatible、SSE 三个入口共享一套核心逻辑。
- handler 内不再出现 `agent.Builder`。
- handler 内不再出现 `agent.RunCheckin`。

### Phase 3：移除 CheckinThreads 全局信号

目标：消除进程内跨层状态。

任务：

- Coordinator 继续写 `state.RouteToCheckin`。
- `ResearchService` 捕获最终 state 或运行结果。
- `ChatService` 根据 `RouteToCheckin` 调用 `CheckinService`。
- 删除 `internal/agent/checkin_signal.go`。

验收：

- 多个并发 thread 不依赖全局 map。
- checkin 路由可用单元测试覆盖。

### Phase 4：事件落库与回放

目标：Agent 运行过程可审计、可回放、可评测。

新增表：

```sql
CREATE TABLE IF NOT EXISTS runs (
    id          VARCHAR(128) PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL,
    mode        VARCHAR(32) NOT NULL,
    status      VARCHAR(32) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_thread_created (thread_id, created_at),
    KEY idx_user_created (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS run_events (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    run_id      VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL,
    event_name  VARCHAR(64) NOT NULL,
    agent       VARCHAR(64) NOT NULL DEFAULT '',
    payload     JSON NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_run_id (run_id, id),
    KEY idx_thread_created (thread_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

验收：

- 每次 agent run 有 run_id。
- 前端 SSE 看到的关键事件可在 `run_events` 中回放。
- 后续可以做失败分析和质量评测。

### Phase 5：入口治理

目标：达到可生产部署的入口标准。

任务：

- 加鉴权 middleware。
- CORS 白名单化。
- request body size limit。
- per user / per IP rate limit。
- 微信 POST 签名校验。
- SSE 心跳和断线处理。
- 统一错误码和错误事件。

验收：

- 未授权请求不能读取任意 `thread_id`。
- 大请求、恶意请求、重复请求有明确保护。
- 入口层只剩协议逻辑。

## 8. 建议接口形态

### 8.1 ChatService

```go
type ChatService interface {
    RunTurn(ctx context.Context, req RunTurnRequest) (<-chan AppEvent, error)
}
```

### 8.2 ResearchService

```go
type ResearchService interface {
    Run(ctx context.Context, req ResearchRunRequest) (ResearchRunResult, error)
    Resume(ctx context.Context, req ResearchResumeRequest) (ResearchRunResult, error)
}
```

### 8.3 CheckinService

```go
type CheckinService interface {
    RunTurn(ctx context.Context, req CheckinTurnRequest) (CheckinTurnResult, error)
    AnalyzeImage(ctx context.Context, req ImageAnalyzeRequest) (CheckinTurnResult, error)
}
```

### 8.4 ReminderService

```go
type ReminderService interface {
    Register(ctx context.Context, threadID string) (<-chan AppEvent, func(), error)
    List(ctx context.Context, threadID string, limit int) ([]ReminderDTO, error)
    Cancel(ctx context.Context, threadID, reminderID string) (ReminderDTO, error)
    Toggle(ctx context.Context, threadID, reminderID string, active bool) (ReminderDTO, error)
}
```

## 9. 当前文件改造映射

| 当前文件 | 问题 | 目标位置 |
|---|---|---|
| `internal/handler/deer.go` | Fat Handler，业务编排过多 | 拆到 `internal/app/chat_service.go`、`research_service.go`、`checkin_service.go` |
| `internal/handler/wechat.go` | 自己构建 graph，逻辑重复 | 只做 XML 适配，调用 `ChatService` |
| `internal/handler/openai.go` | 直接调用 checkin agent | 调用 `ChatService` 或 `CheckinService` |
| `internal/agent/checkin_signal.go` | 进程内跨层状态 | 删除，改为显式结果 |
| `internal/infra/db.go` | 全局 DB，可测试性弱 | 逐步通过 repository 注入 |
| `internal/store/messages.go` | `MAX(turn_idx)+1` 并发风险 | repository + 自增 id / 事务 |
| `main.go` | 初始化和 HTTP 细节混杂 | 后续拆 `server.go`、`wire.go` 或 `bootstrap.go` |

## 10. 测试策略

### 10.1 Service 单元测试

重点覆盖：

- research 请求能走 Planner。
- checkin 请求能分流到 CheckinService。
- image 请求不进入 Research Graph。
- interrupt feedback 能恢复计划。
- final message 能持久化。
- reminder scheduled 事件能透传。

### 10.2 Handler 测试

重点覆盖：

- HTTP 请求解析。
- SSE 事件格式。
- 错误请求返回。
- 鉴权失败。
- 微信 XML 回复格式。
- OpenAI-compatible response 格式。

### 10.3 Contract 测试

为 `AppEvent` 建立快照或 schema 测试：

- `plan`
- `interrupt`
- `tool_calls`
- `tool_call_result`
- `final_message`
- `reminder_scheduled`
- `reminder`
- `error`

### 10.4 Agent 回归评测

准备固定 case：

- 研究类问题。
- 闲聊类问题。
- 打卡记录。
- 历史查询。
- 创建提醒。
- 图片饮食识别。
- 计划编辑后恢复。

每次改 prompt、工具、模型参数后跑一次。

## 11. 风险与权衡

### 11.1 不建议一步到微服务

当前项目体量适合模块化单体。直接拆微服务会引入 RPC、部署、链路追踪、分布式事务成本，不划算。

推荐路径：

```text
Fat Handler
  -> Modular Monolith
  -> Clear Service Boundary
  -> 按需拆独立服务
```

### 11.2 不建议过早抽象所有 infra

可以先从 `MessageRepository`、`RunEventRepository`、`ReminderRepository` 三个最关键接口开始。LLM 和 MCP 可以后续再做 provider 抽象。

### 11.3 保持 Eino Graph 形态

当前 `State + Goto` 的图编排是项目核心资产，不需要因为瘦入口而重写。Service 层只负责调用和治理，不替代 Agent Runtime。

## 12. 最终效果

改造完成后，代码职责会变成：

```text
handler:
  我收到了一个 HTTP / 微信 / OpenAI 请求，要怎么读写协议？

app service:
  这是一个什么业务请求？应该跑研究、打卡、图片还是恢复计划？

agent:
  给定状态和消息，下一步哪个 agent 执行？调用什么工具？

repo / infra:
  数据和外部依赖怎么读写？
```

这会带来几个直接收益：

- 新增入口不重复写业务逻辑。
- 核心 agent 行为可测试。
- 多副本部署更稳。
- 事件可审计、可回放。
- 后续加用户、权限、计费、评测、监控更顺。

## 13. 推荐下一步

优先做 Phase 1 和 Phase 2：

1. 新建 `internal/app`。
2. 把 `/chat/stream` 的业务逻辑搬到 `ChatService`。
3. 再拆 `ResearchService` 和 `CheckinService`。
4. 保持外部 API 和前端事件协议完全不变。
5. 最后删除 `agent.CheckinThreads` 这类跨层全局状态。

这一步完成后，`deepAgent` 的架构会从“功能能跑”进入“可以持续扩展”的状态。
