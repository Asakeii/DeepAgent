# SSE 心跳保活

## 背景

`/chat/stream` 是 Agent 运行的主交互通道。生产环境中，浏览器、反向代理、网关和负载均衡可能会关闭长时间没有字节输出的 SSE 连接，导致前端误以为 run 中断。成熟 Agent 项目需要让长任务流式连接具备可预期的保活行为。

## 目标

- 为 `/chat/stream` 增加可配置 SSE heartbeat。
- 使用 SSE comment frame（`: heartbeat`），不改变现有 `event/data` 事件协议。
- heartbeat 生命周期绑定 HTTP request context，连接关闭或请求结束后自动停止。
- 默认开启，支持通过配置调整间隔或禁用。
- 保持无状态：每条连接只在当前请求 goroutine 内维护 ticker，不保存跨 pod 连接状态。

## 非目标

- 不在本阶段实现 run cancellation。
- 不改变前端业务事件名称或 payload。
- 不引入进程级连接注册表来管理 chat stream。

## 方案反思

- SSE heartbeat 只能证明连接仍活着，不能证明 Agent 子任务仍在推进；后续仍需要 run cancellation、step retry 和更细的 tool/model latency 指标。
- 当前 heartbeat 间隔是全局配置，尚未按网关或环境分层；后续部署到多环境时可以通过配置中心覆盖。
- 写入失败时 heartbeat goroutine 退出，实际请求 context 通常也会很快结束；如果后续要精确观测断连原因，可在 writer 层增加结构化日志。

