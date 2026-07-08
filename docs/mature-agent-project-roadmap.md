# deepAgent 成熟化路线图

## 1. 当前判断

`deepAgent` 现在已经具备成熟 Agent 项目的核心雏形：

- 有多 Agent 研究工作流：Coordinator、Planner、Human Feedback、Researcher、Coder、Reporter。
- 有结构化计划契约：`Plan` / `Step`。
- 有工具调用能力：搜索、Python、打卡、提醒、图片识别。
- 有可恢复能力：MySQL checkpoint。
- 有长期业务记忆：messages / checkins。
- 有提醒调度：Redis ZSET + SSE 推送。
- 有多入口：Web SSE、CLI、WeChat、OpenAI-compatible。
- 有前端活动面板：计划、工具、提醒、会话历史。
- 已开始服务入口瘦身：`internal/app` 应用服务层。
- 有 OpenSpec 规格意识。

但它距离“成熟可落地 Agent 产品”还差一圈工程化能力。成熟 Agent 项目不是只靠模型能力，而是靠 **可控、可测、可观测、可恢复、可治理、可运营**。

## 2. 成熟 Agent 的标准

一个成熟 Agent 项目至少要回答这些问题：

| 维度 | 成熟标准 |
|---|---|
| 产品边界 | 用户知道它能做什么、不能做什么，失败时知道如何继续 |
| Agent 编排 | 多 Agent 分工清晰，状态可追踪，计划可确认，可取消，可恢复 |
| 工具治理 | 工具有权限、超时、重试、审计、错误分类、危险动作确认 |
| 记忆系统 | 短期上下文、长期记忆、业务记录、用户画像分层清晰 |
| 评测体系 | Prompt、模型、工具、路由变化都有回归评测 |
| 安全合规 | 身份、权限、数据隔离、敏感信息、注入攻击都有防护 |
| 可观测性 | 每次 run 的事件、耗时、token、成本、工具调用、错误可追踪 |
| 可靠性 | 断线、重试、超时、并发、多副本、队列积压都有方案 |
| 部署运维 | 配置、密钥、迁移、健康检查、灰度、回滚完整 |
| 商业化/运营 | 用户、用量、留存、成本、反馈、质量分析可运营 |

## 3. 当前最需要补的能力

### 3.1 用户与权限体系

当前问题：

- 主要以 `thread_id` 作为会话身份。
- `/api/messages?thread_id=...`、`/api/reminders?thread_id=...` 这类接口缺少用户所有权校验。
- CORS 仍偏开发态。
- WeChat 用户、Web 用户、OpenAI-compatible 调用方之间没有统一身份模型。

需要补：

- `users` 表。
- `sessions / threads` 表，绑定 `user_id`。
- API 鉴权 middleware。
- Web session / API key / WeChat openid 映射。
- 所有 thread/message/reminder/checkin 查询都加 user ownership check。
- CORS 白名单。
- rate limit。

建议优先级：P0。

推荐表：

```sql
CREATE TABLE users (
    id          VARCHAR(128) PRIMARY KEY,
    provider    VARCHAR(32) NOT NULL,
    provider_id VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_provider_user (provider, provider_id)
);

CREATE TABLE threads (
    id          VARCHAR(128) PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    source      VARCHAR(32) NOT NULL DEFAULT 'web',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_updated (user_id, updated_at)
);
```

### 3.2 Run / Event 运行记录

当前问题：

- SSE 事件能发给前端，但运行过程没有完整落库。
- 出错后难以回放一次 Agent run。
- 无法统计每个 Agent、工具、模型的耗时和失败率。
- 后续做评测、审计、成本分析缺少基础数据。

需要补：

- `runs` 表：记录一次 Agent 运行。
- `run_events` 表：记录事件流。
- 每次请求生成 `run_id`。
- 所有事件统一带 `run_id`、`thread_id`、`user_id`。
- 支持 run replay。

建议优先级：P0。

推荐表：

```sql
CREATE TABLE runs (
    id          VARCHAR(128) PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL,
    mode        VARCHAR(32) NOT NULL,
    status      VARCHAR(32) NOT NULL,
    started_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at    TIMESTAMP NULL,
    error       TEXT,
    KEY idx_thread_started (thread_id, started_at),
    KEY idx_user_started (user_id, started_at)
);

CREATE TABLE run_events (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    run_id      VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL,
    event_name  VARCHAR(64) NOT NULL,
    agent       VARCHAR(64) NOT NULL DEFAULT '',
    payload     JSON NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_run_id (run_id, id)
);
```

### 3.3 Agent 评测体系

当前问题：

- 没有系统化 eval。
- Prompt、模型、工具 schema、路由规则一改，很难知道有没有退化。
- 研究任务、打卡任务、提醒任务、图片识别任务的质量没有量化标准。

需要补：

- 固定测试集。
- 离线 eval runner。
- 评分维度。
- golden answer / rubric。
- 路由准确率统计。
- 工具调用正确率统计。
- 引用质量评测。

建议优先级：P0。

评测集建议：

| 类别 | 数量 | 目标 |
|---|---:|---|
| 闲聊/拒答/澄清 | 20 | 不误进研究或打卡 |
| 深度研究 | 50 | 计划合理、引用可靠、报告完整 |
| 打卡记录 | 30 | 正确调用 record/query 工具 |
| 提醒创建 | 30 | 时间解析正确、重复提醒正确 |
| 图片饮食 | 20 | 能识别、能估算、能入库 |
| 中断恢复 | 15 | plan accept/edit 后状态正确 |
| 工具失败 | 20 | 搜索/Redis/DB 失败有降级 |

评分维度：

- routing_accuracy
- plan_quality
- tool_call_accuracy
- citation_quality
- final_answer_completeness
- safety_compliance
- latency
- token_cost

### 3.4 工具治理

当前问题：

- 工具可以调用，但治理能力还薄。
- 缺少统一超时、重试、熔断、错误分类。
- 缺少危险工具审批模型。
- 缺少工具调用审计。

需要补：

- Tool Registry。
- Tool Policy。
- Tool Runtime wrapper。
- Tool audit log。
- Tool timeout / retry / circuit breaker。
- 工具参数 schema 版本化。

建议优先级：P0。

工具治理结构：

```text
Agent
  -> ToolCall
  -> ToolPolicy.Check
  -> ToolRuntime.Execute
      -> timeout
      -> retry
      -> audit
      -> redact sensitive output
  -> ToolResult
```

工具分类：

| 等级 | 类型 | 策略 |
|---|---|---|
| safe | 搜索、查询历史 | 可自动执行 |
| write | 写打卡、建提醒 | 自动执行，但要审计 |
| external | 发消息、外部 API | 需要权限 |
| dangerous | 删除、支付、群发 | 必须人工确认 |

### 3.5 记忆系统

当前问题：

- `messages` 和 `checkins` 已有，但记忆分层还不清晰。
- 研究历史、打卡历史、用户偏好、长期画像混在业务逻辑里。
- 没有摘要压缩和检索策略。

需要补：

- Conversation memory：最近对话。
- Episodic memory：历史事件和任务。
- Semantic memory：用户偏好、长期事实。
- Business memory：打卡、提醒、计划、报告。
- Memory write policy：什么时候写，写什么。
- Memory retrieval policy：什么时候取，取多少，如何排序。

建议优先级：P1。

推荐结构：

```sql
CREATE TABLE memories (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL DEFAULT '',
    kind        VARCHAR(32) NOT NULL,
    content     TEXT NOT NULL,
    importance  INT NOT NULL DEFAULT 0,
    source      VARCHAR(64) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_kind (user_id, kind, importance)
);
```

后续如果接向量库，可增加：

- embedding。
- memory chunks。
- semantic search。
- time decay。

### 3.6 可靠性与并发

当前问题：

- `messages` 的 turn_idx 生成方式容易有并发问题。
- SSE 断线后只有部分提醒事件有 pending 机制。
- Agent run 缺少取消和幂等。
- 多副本部署下仍需要更多一致性设计。

需要补：

- request id / idempotency key。
- run cancellation。
- tool timeout。
- graph timeout。
- SSE heartbeat。
- multi connection registry。
- 消息写入事务化。
- Redis pending/outbox 机制扩展到更多事件。

建议优先级：P1。

重点修复：

```text
AppendMessage:
  当前: SELECT MAX(turn_idx) + INSERT
  建议: 使用自增 id 排序，或事务内维护 turn counter
```

### 3.7 安全与 Prompt Injection 防护

当前问题：

- 搜索内容、网页内容、用户输入都会进入模型上下文。
- 工具调用还缺少明确的安全策略。
- 图片输入、外部链接、WeChat 回调都需要边界保护。

需要补：

- system/developer 指令隔离。
- web 内容引用标记。
- prompt injection detector。
- tool call allowlist。
- sensitive data redaction。
- WeChat POST 签名校验。
- request body size limit。
- image size/type limit。
- URL fetch allow/deny list。

建议优先级：P1。

### 3.8 可观测性

当前问题：

- 目前主要依赖日志和 SSE。
- 缺少 trace_id / run_id。
- 缺少模型成本、token、latency、tool error rate 统计。

需要补：

- structured logging。
- run_id/thread_id/user_id 全链路日志。
- metrics。
- tracing。
- dashboard。
- error taxonomy。

建议优先级：P1。

核心指标：

| 指标 | 含义 |
|---|---|
| run_success_rate | agent run 成功率 |
| route_accuracy | 路由准确率 |
| avg_latency | 平均响应时间 |
| p95_latency | P95 延迟 |
| tool_error_rate | 工具失败率 |
| model_error_rate | 模型失败率 |
| token_cost_per_run | 单次成本 |
| interrupt_accept_rate | 计划接受率 |
| reminder_delivery_rate | 提醒送达率 |

### 3.9 产物与报告系统

当前问题：

- 研究结果目前主要作为消息存在。
- 缺少报告版本、引用来源、导出、收藏、分享。

需要补：

- artifacts 表。
- report version。
- citation table。
- markdown/html/pdf export。
- 分享链接。
- 报告再编辑。

建议优先级：P2。

### 3.10 前端成熟度

当前问题：

- 已有 activity panel 和 reminder cards，但还缺成熟工作台能力。
- 错误恢复、取消、重试、编辑计划、报告管理仍可增强。

需要补：

- run cancel。
- retry failed step。
- plan diff / edit。
- report viewer。
- citations panel。
- tool timeline。
- session search。
- settings panel。
- user/account page。
- accessibility audit。

建议优先级：P2。

## 4. 推荐实施路线

### Phase 0：稳定当前架构

目标：把刚完成的应用服务层继续压实。

任务：

- 增加 `ResearchRunner`、`CheckinRunner` 接口，减少 service 对 `agent` 包的硬依赖。
- 为 `ChatService` 增加完整 fake runner 单测。
- 为 WeChat/OpenAI-compatible 添加 handler 测试。
- 明确 `AppEvent` 结构，替代散落的 string event。

验收：

- handler 内不出现 `agent.Builder`。
- handler 内不出现 `agent.RunCheckin`。
- service 可用 fake runner 单测。

### Phase 1：用户与运行记录

目标：进入可运营状态。

任务：

- 加 `users`、`threads`、`runs`、`run_events`。
- 引入 `run_id`。
- 所有消息、提醒、打卡绑定 `user_id`。
- 历史接口加 user ownership check。
- SSE 事件落库。

验收：

- 任意一次 Agent run 可回放。
- 不能通过猜 thread_id 读取别人的数据。
- 能统计每天 run 数、成功率、失败原因。

### Phase 2：评测与工具治理

目标：让 Agent 质量可控。

任务：

- 建 eval dataset。
- 实现 eval runner。
- 工具调用 wrapper。
- 工具超时、重试、审计。
- 工具风险等级。

验收：

- 每次 prompt/model/tool 改动可跑回归。
- 工具失败不会直接拖垮整个 run。
- 高风险工具需要确认。

### Phase 3：记忆与个性化

目标：提升长期助手能力。

任务：

- 建 memories 表。
- 对话摘要。
- 用户偏好提取。
- 打卡周期总结。
- 记忆检索策略。

验收：

- 用户长期偏好可被稳定复用。
- 上下文不会无限膨胀。
- 记忆写入有可解释来源。

### Phase 4：安全、观测、部署

目标：生产可用。

任务：

- API auth。
- CORS 白名单。
- rate limit。
- body/image size limit。
- structured logging。
- metrics dashboard。
- DB migration 工具。
- health/readiness check。
- 灰度和回滚。

验收：

- 可以面向真实用户灰度。
- 出问题能定位到 run、agent、tool、模型。
- 配置和密钥不进入镜像和仓库。

## 5. P0 任务清单

优先做这些：

- [ ] 用户身份与 thread ownership。
- [ ] `runs` / `run_events`。
- [ ] Agent eval dataset。
- [ ] Tool timeout/retry/audit wrapper。
- [ ] `ResearchRunner` / `CheckinRunner` 接口化。
- [ ] WeChat POST 签名校验。
- [ ] API request body limit。
- [ ] SSE heartbeat。
- [ ] 消息写入并发修复。
- [ ] 结构化日志和 run_id。

## 6. P1 任务清单

然后做这些：

- [x] memory 分层。
- [x] report artifact。
- [x] citation 存储。
- [ ] reminder multi-connection。
- [ ] run cancellation。
- [ ] tool risk policy。
- [x] prompt injection 防护。
- [ ] 用户设置。
- [ ] 前端 session search。

## 7. P2 任务清单

最后增强：

- [ ] PDF/HTML 导出。
- [ ] 分享链接。
- [ ] 多模型路由。
- [ ] 成本预算。
- [ ] 团队空间。
- [ ] 管理后台。
- [ ] A/B eval。
- [ ] 插件市场。

## 8. 建议的成熟架构形态

```text
handler
  -> auth middleware
  -> request decoder
  -> app service
  -> response encoder

app
  -> ChatService
  -> ResearchService
  -> CheckinService
  -> ReminderService
  -> RunEventService
  -> EvalService

agent
  -> Eino graph
  -> ReAct agents
  -> planner/reporter/researcher/coder

tool runtime
  -> registry
  -> policy
  -> timeout/retry
  -> audit

repo
  -> users
  -> threads
  -> messages
  -> runs
  -> run_events
  -> checkins
  -> reminders
  -> memories
  -> artifacts

infra
  -> mysql
  -> redis
  -> llm
  -> mcp
  -> object storage
  -> metrics
```

## 9. 近期最建议做的三件事

### 9.1 先做 run/event 落库

这是成熟化的地基。没有 run/event，就很难做评测、回放、成本分析、错误定位。

### 9.2 再做用户和权限

现在的 `thread_id` 足够开发调试，但真实产品必须有 user ownership。

### 9.3 同步做 eval

Agent 项目越往后，越容易出现“改了一个 prompt，另外三个场景坏了”。eval 是防止系统变玄学的关键。

## 10. 总结

`deepAgent` 当前最宝贵的是已经有了完整 Agent 工作流和真实业务场景，不需要推倒重来。

下一阶段应该把重点从“功能跑通”转向“质量可控”：

- 用 run/event 让运行过程可见。
- 用用户权限让数据边界可信。
- 用 eval 让模型行为可回归。
- 用工具治理让 Agent 行动可控。
- 用观测和部署体系让系统可运营。

做到这些以后，它才真正从一个能演示的 Agent 项目，变成一个能长期运行、能持续迭代、能服务真实用户的成熟 Agent 产品。
