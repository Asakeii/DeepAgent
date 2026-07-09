# 用户 Token 预算控制

## 背景

成熟 Agent 项目需要能控制模型用量，避免单个用户在多 pod 部署下无限消耗模型资源。当前项目已经具备 `user_settings` 和 `model_usage_logs` 两个基础能力：用户偏好保存在 MySQL，Research、Checkin、Vision 路径的 token usage 也已经按 `run_id/thread_id/user_id` 归因落库。因此本阶段应复用现有共享存储，不新增进程内计数器。

## 变更

- 在 `user_settings` 增加 `daily_token_budget`，表示用户每日 token 预算。
- `/api/settings` 支持读取和更新该字段；传入 `<=0` 表示清空预算。
- ChatService 在创建 run 并建立 run event writer 后、调用模型或工具前，查询当天用户已用 token。
- 当 `model_usage_logs` 中当天 token 总量达到预算时，写入 `token_budget_exceeded` 错误事件并终止本次 run。
- token 用量查询基于 MySQL，保持多 pod 一致。

## 反思与边界

- 这不是金额成本预算。当前还没有模型价格表、供应商币种和上下文缓存折扣规则，直接做金额计算会产生不可靠的账单语义。
- 预算采用用户时区自然日，而不是简单滚动 24 小时窗口，更贴近产品设置中的“每日预算”。
- 预算检查失败时选择 fail-closed，避免共享用量不可读时绕过预算。
- 本阶段只做用户级预算，不做团队空间、项目级预算或管理员审批。
- 已经开始执行的 run 不会被中途预算熔断；后续可以结合流式 token 统计或模型回调做更细粒度控制。
