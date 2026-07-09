# Plugin Marketplace

## 背景

成熟 Agent 项目需要可治理的工具生态。主流 Agent 平台正在把工具接入收敛到标准协议和应用/连接器市场：MCP 官方规范把工具、资源、提示词作为 server 暴露能力；OpenAI Apps SDK 也构建在 MCP 之上。当前项目已经通过 Eino MCP adapter 加载 MCP 工具，但缺少面向用户/团队的插件发现、启用状态和运行时过滤。

## 变更

- 复用现有 MCP 配置作为插件目录来源，不新增自研插件协议。
- 新增 `plugin_installs` 表，按 `user` / `team` scope 持久化插件启用状态。
- 新增插件市场 API：
  - `GET /api/plugins`
  - `GET /api/plugins?team_id=...`
  - `POST /api/plugin-installs`
- 插件目录返回 MCP server、transport、启用状态和已发现 tools。
- Chat run 注入用户/团队 plugin scope。
- Coder 和 Background Investigator 加载 MCP tools 时按 scope 过滤。
- 团队插件启用/禁用要求 `owner/admin`，普通成员只能读取目录。

## 反思与边界

- 本阶段只治理“已配置 MCP server”的可见性和启用状态，不支持用户上传任意代码，避免供应链风险。
- 已配置插件默认继续可用，保证兼容现有部署；用户/团队可以逐项禁用或重新启用。
- 插件市场暂不做版本、评分、沙箱安装、OAuth 授权和插件级审计视图；这些适合作为下一阶段能力。
- MCP server 的认证仍来自服务端配置，前端不会返回密钥和启动参数。

## 参考

- MCP 官方规范：server 可以暴露 tools/resources/prompts。
- OpenAI Apps SDK：基于 MCP 构建应用/工具连接能力。
