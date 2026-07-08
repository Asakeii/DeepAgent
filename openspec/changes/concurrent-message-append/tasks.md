# Tasks

- [x] 移除 `AppendMessage` 中的 `MAX(turn_idx)+1` 写法。
- [x] 使用 MySQL 自增 id 作为消息稳定排序源。
- [x] 将 `messages.turn_idx` 调整为 `BIGINT`。
- [x] 增加启动时 messages 表幂等创建/修正。
- [x] 增加同 thread 并发 append 测试，验证 turn_idx 不重复。
- [ ] 后续：引入正式 DB migration 工具和 schema 版本表。
