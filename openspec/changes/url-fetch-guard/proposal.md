# URL Fetch 安全策略

## 背景

路线图在安全边界中提出 `URL fetch allow/deny list`。当前 `web_fetch` 和图片 URL 下载都会请求用户或模型提供的 URL，如果不加策略，可能访问 localhost、私网地址或云元数据地址，形成 SSRF 风险。

## 目标

- 新增统一 URL guard，复用到 `web_fetch` 和图片 URL 输入。
- 默认仅允许 `http` / `https` scheme。
- 默认拒绝 localhost、loopback、private、link-local、unspecified 等本地/私网目标。
- 新增 `server.url_allowed_hosts`、`server.url_denied_hosts`、`server.url_allow_private_networks` 配置。
- 支持 exact host 和 `*.example.com` 形式的 host 匹配。

## 非目标

- 不在本阶段实现 DNS 解析后的 IP 再校验。
- 不替换底层 HTTP client 或 transport。
- 不影响 SearXNG 内部搜索服务 URL，因为它不是用户提供的 fetch URL。

## 方案反思

- 这次把策略做成 `internal/security` 小包，避免把安全规则散落在工具和 app 中。
- 默认拒绝私网可以覆盖最常见 SSRF 高风险输入；但域名解析到私网 IP 仍需后续在 HTTP transport 层做 DNS/IP 绑定校验。
- Allowlist 非空时只允许匹配 host，适合生产强约束；denylist 优先级更高，便于快速封禁风险域名。

