# Tasks

- [x] 增加 OIDC 与具名 API key principal 配置。
- [x] 基于 `go-oidc` 实现 Discovery、JWKS、issuer、audience、签名和有效期校验。
- [x] 将可信 Principal 注入 request context，并覆盖不可信用户字段。
- [x] 基于 role/group claim 保护管理员 API。
- [x] 使用稳定主体执行多 Pod 共享限流和 users 幂等建档。
- [x] 增加 `/api/me` 身份探测接口。
- [x] 增加 API key、OIDC 签名/audience、身份优先级和 middleware 测试。
- [x] 更新配置示例、成熟化路线图和历史 OpenSpec 状态。
- [ ] 后续：前端 Authorization Code + PKCE 登录、刷新和退出流程。
- [ ] 后续：生产环境空认证配置启动门禁和 Secret Manager 集成。
- [ ] 后续：opaque access token introspection adapter。
