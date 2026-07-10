# Tasks

- [x] 新增 `server.api_keys` 配置。
- [x] 新增 `server.rate_limit_per_minute` 配置。
- [x] 保护 `/chat/stream`、`/v1/chat/completions` 和 `/api/*`。
- [x] 支持 `Authorization: Bearer <key>` 和 `X-DeepAgent-API-Key`。
- [x] 使用 Redis 计数实现多 pod 共享限流。
- [x] 更新示例配置。
- [x] 补充配置解析和 HTTP middleware 单元测试。
- [x] 后续：支持无状态 OIDC/JWKS Bearer JWT 校验和可信用户主体（浏览器登录流程待补）。
- [ ] 后续：Redis Lua token bucket。
- [ ] 后续：限流命中率和 Redis fail-open 告警指标。
