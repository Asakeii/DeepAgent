# Tasks

- [x] 提醒触发时先写 pending，再发布 Redis Pub/Sub 事件。
- [x] Worker 启动时订阅 `reminders:events`。
- [x] 本地 SSE 投递成功后 ack pending。
- [x] 保留重连时 DrainPending 兜底逻辑。
- [x] 补充 pending payload 往返测试。
- [ ] 后续：前端基于 reminder id 去重。
- [ ] 后续：用 Redis Streams 或 event gateway 提升投递语义。
