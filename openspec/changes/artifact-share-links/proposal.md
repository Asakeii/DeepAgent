# Artifact 分享链接

## 背景

成熟 Agent 产品需要把生成的报告、结论和资料沉淀成可传播的产物。当前系统已经能持久化 report artifact 和 citation，但只能由拥有者通过受保护 API 查询，无法把单个报告安全地分享给外部读者。

## 变更

- 新增 `artifact_shares` 表，保存 artifact 分享令牌的 SHA-256 hash、归属用户、过期时间和撤销时间。
- 新增受保护接口 `/api/artifact-shares`：
  - `POST`：为当前用户拥有的 artifact 创建分享链接。
  - `DELETE`：撤销当前用户创建的分享链接。
- 新增公开只读接口 `/share/artifacts?token=...`，通过 token 读取未过期、未撤销的 artifact。
- 公开响应隐藏 `user_id/thread_id/run_id`，避免泄露内部归属标识。

## 反思与边界

- 本阶段只分享 artifact 内容，不分享完整 run timeline、thread 消息或 tool audit，避免公开链接扩大隐私边界。
- 数据库存储 token hash，不存原始 token；原始 token 只在创建时返回。
- 分享链接依赖 MySQL 解析，保持无状态和多 pod 一致。
- 公开链接当前返回 JSON API，不做独立前端页面；后续可以在同一路由之上增加渲染页和访问统计。
- 这不是权限协作系统；团队空间、评论和细粒度 ACL 仍留到后续阶段。
