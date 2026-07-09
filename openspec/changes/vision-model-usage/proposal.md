# Vision 模型用量记录

## 背景

Research graph 和 Checkin ReAct agent 已经通过 Eino callback 记录模型 token usage。图片饮食分析路径直接调用 `VisionModel.Generate`，不会经过 ReAct callback，因此需要从返回的 `schema.Message.ResponseMeta.Usage` 读取 usage。

## 目标

- VisionModel 直接调用返回后读取 `ResponseMeta.Usage`。
- 将图片分析 token usage 记录到 `model_usage_logs`。
- 复用 `toolruntime.AuditContext` 提供的 run/thread/user 维度。

## 非目标

- 不估算缺失的 usage。
- 不计算金额成本或执行预算拦截。

## 方案反思

- Eino schema 已经在 `ResponseMeta.Usage` 中暴露 token usage，本阶段直接复用，不为 Vision 单独设计统计模型。
- 如果具体模型供应商不返回 usage，系统会跳过记录；后续成本预算应区分“未上报”和“零消耗”。
