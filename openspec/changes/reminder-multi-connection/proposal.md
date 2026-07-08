# 提醒 SSE 多连接支持

## 背景

提醒投递使用 Redis 作为队列和 pending 存储，但活跃 SSE 连接注册表仍是进程内结构。旧实现每个 `thread_id` 只保存一个 channel，同一用户打开多个页面或重复连接时，后注册的连接会覆盖旧连接，导致提醒只投给最后一个连接。

## 方案

- 将 `ConnRegistry` 从 `thread_id -> channel` 改为 `thread_id -> channel set`。
- `Register` 为同一 thread 保留多个活跃连接。
- `Unregister` 改为按 `thread_id + channel` 移除指定连接。
- `Push` 对同一 thread 的所有活跃连接广播，至少一个连接投递成功则返回 true。

## 方案反思

- 这解决的是单 pod 内多 SSE 连接问题。
- 多 pod 跨实例实时广播已由后续 `reminder-pubsub-delivery` 使用 Redis Pub/Sub 继续推进。
- 现有 Redis pending 机制仍保留，用于离线或当前 pod 无连接时的兜底投递。
