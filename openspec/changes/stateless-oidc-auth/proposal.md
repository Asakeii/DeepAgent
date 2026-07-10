# 无状态 OIDC/JWKS 认证边界

## 背景

已有 API key middleware 能挡住匿名流量，但业务身份仍来自调用方可写的 `X-DeepAgent-User` 或 `user_id`。持有同一个全局 key 的调用方可以伪装成其他用户，thread ownership 因此缺少可信身份根。多 Pod 部署也不能依赖单 Pod session 内存来补这个缺口。

## 方案

- 使用 `github.com/coreos/go-oidc/v3/oidc`，通过 OIDC Discovery 获取 `jwks_uri`，复用库提供的远端公钥缓存和刷新能力。
- 校验 Bearer JWT 的签名、issuer、audience 和有效期，将稳定 `sub`（或显式配置的 claim）映射成内部用户 ID。
- 将认证结果写入 request context；handler 优先且强制使用该主体，忽略请求 Header、Query 和聊天 body 中的伪造用户 ID。
- 支持 `api_key_principals`，把机器密钥绑定到稳定服务账号和管理员属性；旧 `api_keys` 继续兼容，但映射成密钥派生的服务账号。
- OIDC 管理员权限来自可配置 role/group claim，普通已认证用户访问管理员 API 返回 403。
- 认证主体参与 Redis 限流 key，并写入已有 MySQL `users` 表；所有 Pod 只持有可重建的 JWKS 缓存，不持有用户 session 状态。
- 未配置 OIDC/API key 时保留本地开发模式，继续允许轻量身份 Header；生产配置任意认证方式后自动关闭这条信任路径。

## 大厂与框架参考

- Microsoft Identity Platform 要求 Web API 校验 access token 的签名、issuer 和目标 API audience，并建议从 OIDC metadata 获取签名公钥，而不是手写校验逻辑：<https://learn.microsoft.com/en-us/entra/identity-platform/access-tokens>
- Google Identity Platform 使用标准 `/.well-known/openid-configuration` 发现 Provider 端点和公钥：<https://cloud.google.com/identity-platform/docs/web/oidc>
- `go-oidc` 的 `IDTokenVerifier` 实现 OIDC 校验；`RemoteKeySet` 是长生命周期、可缓存并在未知 `kid` 时刷新公钥的 verifier，应在进程内复用：<https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc>

## 方案反思

- 本阶段实现的是资源服务器认证，不包含浏览器 Authorization Code + PKCE 登录页、刷新 token 和退出登录；前端接企业 IdP 时仍需补登录体验。
- OIDC Provider discovery 在启动时 fail-fast。它能避免 Pod 带错误 issuer/audience 接流量，但 IdP 短暂不可用会阻塞新 Pod 启动；生产应通过本地网络、启动重试和发布探针降低影响。
- JWKS 缓存是每 Pod 的可重建派生状态，不影响业务无状态；轮转时库会按新 `kid` 拉取，不能将 JWKS 静态烘焙进镜像。
- 部分 OAuth Provider 发放 opaque access token，不能走本地 JWT/JWKS 校验；此类接入需要 token introspection adapter，不应放松当前 JWT 校验。
- `api_key_principals.key` 仍是敏感配置，示例只放占位符；生产应由 Secret Manager/Kubernetes Secret 注入挂载文件，并定期轮换。
- 为保持历史兼容，完全未配置认证时仍允许开发身份 Header。部署门禁后续应增加“生产环境禁止空认证配置”的启动检查。

## 影响范围

- `conf/config.go` 与示例配置
- `internal/auth` OIDC/API key 认证实现
- HTTP guard、管理员授权和限流身份
- handler 身份解析与 `/api/me`
- users 身份资料幂等写入

## 非目标

- 自建用户名密码系统。
- 在 Agent 服务内保存登录 session。
- 实现 OAuth 授权服务器。
- 为 opaque token 自建 introspection 协议。
