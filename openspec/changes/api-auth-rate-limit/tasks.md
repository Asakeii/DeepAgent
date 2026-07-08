# Tasks

- [x] 新增 `server.api_keys` 配置。
- [x] 新增 `server.rate_limit_per_minute` 配置。
- [x] 保护 `/chat/stream`、`/v1/chat/completions` 和 `/api/*`。
- [x] 支持 `Authorization: Bearer <key>` 和 `X-DeepAgent-API-Key`。
- [x] 使用 Redis 计数实现多 pod 共享限流。
- [x] 更新示例配置。
- [x] 补充配置解析和 HTTP middleware 单元测试。
- [ ] 后续：用户登录态 / JWT / OAuth。
- [ ] 后续：Redis Lua token bucket。
- [ ] 后续：限流命中率和 Redis fail-open 告警指标。
