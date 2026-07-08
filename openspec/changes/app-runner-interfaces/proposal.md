# 应用 Runner 接口化

## 背景

`ChatService` 已经承担 `/chat/stream` 的应用编排，但它直接持有具体 `ResearchService` / `CheckinService`。这让完整编排单测必须触发真实 Eino graph、ReAct agent、数据库和外部模型，后续做取消、评测、OpenAI-compatible 入口复用时也更难替换局部能力。

## 目标

- 为 `ChatService` 增加 `ResearchRunner`、`CheckinRunner`、`ReminderStreamer` 接口边界。
- 保留默认构造函数，现有 handler 不需要改调用方式。
- 增加依赖注入构造函数，允许测试和未来入口传入 fake runner。
- 将 `CheckinService` 对 `agent.RunCheckin` / `agent.AnalyzeFoodImage` 的直接调用移动到默认 runner 适配器。
- 增加 ChatService fake runner 编排测试，覆盖研究路由到打卡的主路径。

## 非目标

- 不重写 Eino graph。
- 不改变现有 SSE 事件协议。
- 不改变真实 Checkin/ReAct agent 行为。
- 不在本阶段把 `ResearchService` 内部的 Eino builder 进一步拆成更细的 graph factory。

## 方案反思

- 这次只在应用服务层抽 runner 接口，粒度比较克制。继续往下拆 graph builder 虽然能提升单测隔离度，但会把 Eino option、checkpoint 和 callback 细节泄漏到更多接口里。
- ChatService 仍依赖数据库做 thread ownership、run 和 memory，这符合当前生产路径；fake runner 测试仍需要 `TEST_MYSQL_DSN`，后续如果要纯单测，应再抽 repository 接口。
- CheckinService 已经通过 runner 适配器隔离 `agent` 包，ResearchService 后续可以沿用同样方式逐步拆出 graph runner。

