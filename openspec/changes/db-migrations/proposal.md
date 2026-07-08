# 数据库迁移基座

## 背景

当前项目依赖 `schema.sql` 和启动时散落的 `Ensure*Tables` 做建表/修表。这个方式适合早期开发，但在多 pod 部署中存在几个问题：

- 每个 pod 启动都会执行多处 DDL，缺少统一版本记录。
- 无法判断某个 schema 变更是否已经应用。
- 后续回滚、审计和发布前检查缺少依据。

## 方案

- 新增嵌入式 SQL migration runner，不引入外部服务。
- SQL 文件放在 `internal/store/migrations/*.sql`，随 Go binary embed。
- 使用 `schema_migrations` 表记录 `version/name/checksum/applied_at`。
- 使用 MySQL `GET_LOCK` 做迁移互斥，避免多 pod 同时执行迁移。
- `InitDB` 改为执行 `store.RunMigrations`，不再散落调用多个 `Ensure*Tables`。
- 保留 `schema.sql`，供 docker init 和人工初始化使用；运行时以 migrations 为准。

## 方案反思

- MySQL DDL 会隐式提交，因此迁移不是严格事务化；当前迁移 SQL 需要保持幂等。
- 这是项目内轻量 runner，不是 goose/atlas 这类完整工具；后续如果需要 down migration、dry run、schema diff，可迁移到成熟工具。
- checksum 校验能防止已应用迁移被悄悄修改。
