# HTTP 安全边界

## 背景

当前 server 的 CORS 使用 `*`，请求体也没有统一大小限制。成熟 Agent 服务会接收 Web、OpenAI-compatible、WeChat 等入口请求，如果缺少边界控制，容易出现跨域误用和大请求拖垮进程的问题。

## 方案

- 新增 `server.allowed_origins` 配置，CORS 只允许显式白名单 Origin。
- 无 `Origin` 的同源、服务端和 CLI 请求保持可用。
- 新增 `server.max_body_bytes` 配置，统一限制 HTTP 请求体大小，默认 1MB。
- 保留 `Content-Type`、`X-DeepAgent-User`、`X-DeepAgent-Run` 作为允许的跨域请求头；API key 认证和限流由后续 `api-auth-rate-limit` 变更补齐。

## 不做

- 不在本阶段实现完整登录态。
- 不在本阶段改造 WeChat 签名校验逻辑。
- 不在本阶段按路由区分不同 body limit。

## 方案反思

- CORS 不是认证，只是浏览器边界；API key 和 rate limit 已在 `api-auth-rate-limit` 中继续推进。
- 图片 base64 请求可能超过 1MB，生产环境应根据业务入口单独配置更大的限制或改为对象存储上传。
