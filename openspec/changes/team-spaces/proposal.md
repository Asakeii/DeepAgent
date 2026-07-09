# Team Spaces

## 背景

成熟 Agent 产品需要支持多人协作。参考主流云厂商和 AI 平台的 Project / Workspace / IAM 设计，团队空间应成为资源、成员、权限、用量和运营分析的共享边界，而不是把所有会话都压在个人 `user_id` 下。

当前项目已经有用户身份、thread ownership、run/event、artifact 和管理概览，但缺少团队维度：

- 无法把 thread 归属到共享空间。
- 成员无法访问同一团队内的会话和产物。
- 权限判断仍主要是单用户 owner 模式。

## 变更

- 新增 `teams` 和 `team_members` 共享存储表。
- `threads` 新增 `team_id` 字段，团队会话归属到团队空间。
- 新增团队 API：
  - `GET /api/teams`
  - `POST /api/teams`
  - `GET /api/team-members?team_id=...`
  - `POST /api/team-members`
- `ChatRequest` 支持 `team_id`，创建团队 thread 前校验用户是团队成员。
- `ThreadBelongsToUser` 语义升级为“个人拥有或团队成员可访问”，复用到消息、run event、提醒和 artifact 等已有边界。
- 会话列表返回个人 thread 和用户所在团队的 thread。
- Artifact 读取和导出支持团队 thread 成员访问。

## 反思与边界

- 本阶段先做团队空间最小后端闭环，不做复杂 RBAC 矩阵；角色仅有 `owner/admin/member`。
- 成员增删先只支持新增/改角色，不支持移除、转让 owner 和邀请链接，避免把权限生命周期一次性做复杂。
- 分享链接创建仍保守要求 artifact 所有人操作；团队成员可读/导出团队产物，但不能默认公开分享他人产物。
- 团队空间的预算、用量汇总、审计日志和前端团队切换是后续增强项。
