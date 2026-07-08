# Tasks

- [x] 新增 `schema_migrations` 版本表。
- [x] 新增 embed SQL migration runner。
- [x] 使用 MySQL `GET_LOCK` 防止多 pod 并发迁移。
- [x] 新增 `0001_initial_schema.sql` 覆盖当前 schema。
- [x] 将 `InitDB` 接入 `RunMigrations`。
- [x] 补充 migration loader / splitter / MySQL 集成测试。
- [ ] 后续：down migration / dry run。
- [ ] 后续：发布前 migration 检查命令。
