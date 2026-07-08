# 提醒跨 Pod 实时投递

## 背景

提醒调度已经使用 Redis ZSET 做共享队列，因此多个 pod 中只有一个 worker 会 claim 某个到期提醒。但 SSE 连接注册表是 pod 本地内存：如果提醒在 A pod 触发，而用户连接在 B pod，旧逻辑会把提醒写入 pending，用户需要重连才能收到。

## 方案

- 复用现有 Redis，不引入新的消息系统。
- 到期提醒触发后：
  - 先写入 `pending:<thread_id>`，保留离线兜底。
  - 再发布到 Redis Pub/Sub channel `reminders:events`。
- 每个 pod 的 worker 启动时订阅该 channel。
- 任意 pod 收到广播后，如果本地 SSE registry 成功投递，就从 pending list 中 ack 掉对应 payload。

## 方案反思

- Redis Pub/Sub 不保证持久化，所以必须保留 pending list。
- 这仍不是 exactly-once 投递；网络抖动时可能出现极少量重复，前端可以基于 reminder id 去重。
- 后续如果提醒成为强 SLA，可演进到 Redis Streams consumer group 或独立 event gateway。
