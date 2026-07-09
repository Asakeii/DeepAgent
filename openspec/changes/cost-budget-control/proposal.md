# Cost Budget Control

## 背景

成熟 Agent 项目不能只看 token 数，还要能把 token usage 转成业务可理解的金额预算。OpenAI API 平台支持 project 级预算/用量管理，Google Cloud Billing Budget 也以预算金额追踪实际成本并触发响应。当前项目已经有 `model_usage_logs`、用户 token 预算和团队空间，但缺少模型价格表、金额预算和团队级预算。

## 变更

- `model.prices` 新增模型价格表配置，单位为 USD / 1M tokens。
- `user_settings` 新增 `daily_cost_budget_micros`，用于用户级每日金额预算。
- 新增 `team_settings` 表和 `/api/team-settings`，用于团队级每日金额预算。
- 成本计算复用 `model_usage_logs`，按模型维度聚合 prompt、completion、cached、reasoning tokens。
- ChatService 在模型/工具执行前检查用户和团队每日金额预算。
- 团队预算读取要求团队成员，更新要求 `owner/admin`。

## 反思与边界

- 不硬编码供应商价格，因为模型价格和企业合同会变化；生产环境必须维护 `model.prices`。
- 预算金额用 micros 存储，避免数据库浮点误差。
- 当前仍是运行前拦截，不做运行中 streaming usage 熔断。
- 如果设置了金额预算但缺少对应模型价格，预算检查 fail-closed，避免绕过成本控制。
- 成本是基于模型上报 usage 的估算，不替代供应商最终账单。

## 参考

- OpenAI API platform projects：project owners can manage project budgets.
- Google Cloud Billing Budgets：track actual costs against planned spend and automate cost control responses.
