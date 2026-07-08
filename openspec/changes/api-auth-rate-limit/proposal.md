# API Key 认证与共享限流

## 背景

当前 Web/API 入口已经有 thread/user ownership 校验和 CORS 白名单，但 `X-DeepAgent-User` 仍由调用方传入，不能作为认证凭据。成熟 Agent 服务需要至少具备可选 API key 认证和限流能力，避免业务接口被未授权调用或被突发流量拖垮。

## 方案

- 新增 `server.api_keys` 配置。
  - 未配置时保持本地开发兼容，不启用 API key 校验。
  - 配置后，受保护接口必须传 `Authorization: Bearer <key>` 或 `X-DeepAgent-API-Key`。
- 新增 `server.rate_limit_per_minute` 配置。
  - 值为 `0` 时禁用。
  - Redis 可用时使用 Redis 计数，多个 pod 共享限流状态。
  - Redis 不可用时 fail-open，避免限流依赖导致主链路不可用。
- 保护范围：
  - `/chat/stream`
  - `/v1/chat/completions`
  - `/api/*`
- 不保护范围：
  - `/healthz`、`/readyz`
  - `/wechat/callback`
  - 静态资源

## 方案反思

- API key 是最小可用认证方案，不等于完整用户登录态；后续仍需要 session/JWT/OAuth 或企业身份接入。
- 当前限流是固定分钟窗口，不是平滑令牌桶；它足够简单且多 pod 一致，后续可演进为 Redis Lua token bucket。
- Redis fail-open 保护可用性，但在 Redis 故障时限流会失效；生产环境需要监控告警。
