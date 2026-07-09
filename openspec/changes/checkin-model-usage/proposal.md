# Checkin 模型用量记录

## 背景

`model-usage-metrics` 已经把 Research graph 的 Eino callback `TokenUsage` 写入 `model_usage_logs`。但 Checkin ReAct agent 是独立调用 `agent.Generate`，如果不显式传入 callback，打卡/提醒路径的模型 token 不会进入用量指标。

## 目标

- Checkin ReAct agent 调用时复用 Eino `agent.WithComposeOptions(compose.WithCallbacks(...))`。
- 将 ChatService 注入的 run/thread/user 审计上下文带入 Checkin 模型用量记录。
- 补充 callback 写入模型用量的集成测试。

## 非目标

- 不在本阶段覆盖 VisionModel 直接图片分析调用。
- 不做模型金额成本计算或预算拦截。

## 方案反思

- Eino ReAct Agent 已提供 callback 透传能力，本阶段直接复用框架接口，不在业务层包一层自定义模型代理。
- Checkin 直调没有 graph state，因此 agent 维度可能来自显式 callback 调用或为空；run/user/thread 仍足够支撑成本归因。
- VisionModel 的直接 `Generate` 后续可以用更通用的模型调用封装处理，避免为单个工具硬塞不一致的记录逻辑。
