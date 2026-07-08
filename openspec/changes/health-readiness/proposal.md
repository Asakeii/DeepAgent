# 健康检查与就绪检查

## 背景

成熟 Agent 服务需要支持多 pod 部署、灰度和自动恢复。当前 server 只有业务接口，缺少给负载均衡、容器编排和发布系统使用的 liveness/readiness 端点。

## 方案

- 新增 `/healthz`：只验证进程可响应，返回 `200 ok`。
- 新增 `/readyz`：验证关键依赖是否可用：
  - MySQL 已初始化且可 ping。
  - ChatModel 已初始化。
  - Redis 若启用则必须可 ping；未启用视为 optional disabled。
- readiness 未通过时返回 `503 not_ready`，避免流量打到尚未完成初始化或依赖异常的 pod。

## 方案反思

- 当前 readiness 只做同步 ping，后续可加入 MCP、SearXNG、迁移版本、模型 provider 轻量探测。
- Redis 在当前项目中主要影响提醒能力，因此未初始化不阻断整体 ready；如果提醒成为核心 SLA，应改为必需依赖。
