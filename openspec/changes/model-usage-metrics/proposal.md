# 模型用量指标

## 背景

成熟 Agent 项目需要知道每次 run 消耗了多少模型 token，才能进一步做成本预算、质量/成本分析和异常告警。Eino model callback 已经提供 `TokenUsage`，应优先复用框架输出，而不是自行估算。

## 目标

- 新增 `model_usage_logs` 表，记录 run、thread、user、agent、model 与 token usage。
- 在 `LoggerCallback` 中复用 Eino `TokenUsage` 写入模型用量。
- `/api/metrics/runs` 汇总 prompt/completion/total/cached/reasoning tokens。
- 保持无状态多 Pod：usage 落 MySQL，指标按共享数据库聚合。

## 非目标

- 不在本阶段配置模型价格或计算金额成本。
- 不做用户预算拦截。
- 不接入第三方 tracing 平台。

## 方案反思

- 这次只记录 Eino callback 能拿到的 usage，不做 token 估算，避免不同模型 tokenizer 不一致。
- 当前只在 ResearchService 的 graph callback 注入 run/user 信息；后续应继续覆盖 Checkin 直接模型调用和 Vision 模型调用。
- 成本预算应该基于这张 usage 表再叠加价格配置、用户预算和拦截策略，而不是混进 callback 写入路径。
