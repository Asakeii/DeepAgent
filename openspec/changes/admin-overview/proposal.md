# Admin Overview 后端

## 背景

成熟 Agent 项目需要运营视角：管理员要能看到用户规模、会话规模、运行成功率、工具错误、模型 token 和 artifact 使用情况。当前项目已经有用户级 run metrics，但缺少跨用户的管理概览，也没有独立 admin key 保护管理接口。

## 变更

- 新增 `server.admin_api_keys` 配置。
- `/api/admin/*` 使用独立 admin key 鉴权；配置 admin key 后普通 API key 不能访问管理接口。
- 新增 `/api/admin/overview`：
  - 用户数、线程数、artifact 数、活跃分享数。
  - 窗口内 run 成功率、失败数、运行中数量。
  - 窗口内工具调用总数、失败/阻断和错误率。
  - 窗口内模型 token 用量。
- 所有聚合来自 MySQL 共享状态，支持多 pod。

## 反思与边界

- 本阶段只做后端只读概览 API，不做前端管理台。
- 不在 admin overview 返回明细用户数据，先降低隐私泄露面。
- 不把 admin 权限复用普通 API key，避免生产环境中用户调用方拿到管理权限。
- 后续管理后台可以在该 API 之上增加趋势图、用户检索、团队空间和告警。
