# 工具调用治理

## 背景

当前项目已经有 Researcher、Coder、Checkin 等 Agent 工具调用能力，但工具调用缺少统一治理入口。随着项目走向多用户、多 pod、可运营形态，工具调用需要可审计、可追踪、可限时、可分级，否则线上排障、权限控制和质量分析都会缺少基础证据。

## 方案

- 复用 Eino `BaseTool` / `InvokableTool` / `StreamableTool` 接口，不重写 ReAct Agent 运行时。
- 在工具列表装配后统一包裹 `toolruntime` wrapper。
- 通过 context 传递 `run_id`、`thread_id`、`user_id`，工具执行时写入审计表。
- OpenAI-compatible 入口也创建 run，并把 run_id 注入 Checkin 工具调用上下文。
- 增加工具风险分级：
  - `safe`：查询类工具，可自动执行。
  - `write`：写入业务数据，允许自动执行但必须审计。
  - `external`：外部网络/API 工具，允许自动执行但必须审计。
  - `dangerous`：危险动作，默认阻断并记录。
- 增加 `/api/tool-audits?run_id=...` 只读接口，按 run 查询工具审计记录，并复用 run 用户归属校验。

## 不做

- 不在本阶段替换 Eino ReAct Agent。
- 不在本阶段引入复杂 Tool Registry 服务或远端策略中心。
- 不在本阶段实现人工审批交互 UI。
- 不在本阶段实现熔断和重试策略，避免在业务写工具上造成重复写。

## 方案反思

- 当前工具治理仍是进程内包装器，策略配置还在代码里；后续成熟形态应把策略做成可配置版本化能力。
- `dangerous` 工具只有默认阻断，没有完整人工确认闭环；后续需要 human-in-the-loop approval。
- 流式工具的审计目前只能记录启动和同步返回错误，不能精确覆盖完整消费生命周期；后续如大量使用 streamable tool，需要封装 reader close/EOF。
- 直接调用型能力，如图片分析直连函数，不完全经过 Eino Tool wrapper；后续应统一进入 Tool Runtime 或补单独审计适配。
