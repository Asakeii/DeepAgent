# 消息并发写入安全

## 背景

成熟 Agent 项目需要支持多 pod 水平扩容。原消息写入逻辑使用 `SELECT MAX(turn_idx) + 1` 生成同一 thread 的下一条消息序号，在同一会话并发请求或多 pod 同时写入时会产生竞争，导致 `turn_idx` 重复或顺序不稳定。

## 方案

- 复用 MySQL `AUTO_INCREMENT id` 作为消息的稳定排序源，不引入额外分布式锁或序列服务。
- `AppendMessage` 在事务内先插入消息，再用 `LastInsertId` 回填 `turn_idx = id`。
- `turn_idx` 调整为 `BIGINT`，与 `id` 类型保持一致。
- 启动时通过 `EnsureMessageTables` 幂等创建/修正 messages 表。

## 方案反思

- `turn_idx` 不再表示 thread 内连续序号，而是全局自增顺序；对“最近消息”和“首条用户消息”查询足够稳定。
- 如果后续产品必须展示 thread 内连续轮次，应新增单独的展示层计数，不应回到 `MAX+1`。
- 当前迁移通过启动时 `ALTER TABLE` 修正列类型，后续成熟部署应引入正式 migration 工具和版本表。
