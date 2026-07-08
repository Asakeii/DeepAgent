# Tasks

- [x] 将提醒连接注册表改为同一 thread 多 channel。
- [x] 调整 SSE stream detach 逻辑，按具体 channel 注销。
- [x] 保持离线 pending 投递逻辑不变。
- [x] 增加多连接广播和单连接注销测试。
- [ ] 后续：用 Redis Pub/Sub 或 Streams 支持跨 pod 实时广播。
