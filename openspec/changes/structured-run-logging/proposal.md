# 结构化 Run 日志

## 背景

项目已经有 `runs`、`run_events`、tool audit 和运行指标接口，但服务日志仍主要是散落的 `log.Printf`。当服务以多 pod 运行时，排障需要能按 `run_id`、`thread_id`、`user_id` 在日志平台聚合一次 Agent 运行的完整轨迹。

## 目标

- 使用 Go 标准库 `log/slog` 配置 JSON 结构化日志，不引入第三方日志框架。
- 提供统一的 run logger，稳定输出 `run_id`、`thread_id`、`user_id` 字段。
- 先覆盖 ChatService run 生命周期、run event 落库失败、run cancellation 检查失败等主链路日志。
- 在 run 完成日志中输出 `status` 和 `duration_ms`。

## 非目标

- 不一次性迁移所有历史 `log.Printf`。
- 不引入 OpenTelemetry tracing exporter。
- 不计算 token/cost 指标。

## 方案反思

- 选择标准库 `slog` 是为了减少依赖和运行时复杂度；如果后续接入企业日志平台，可以在进程入口替换 handler，而不需要改业务代码。
- 这次只迁 run 主链路，仍会有 scheduler、MCP、WeChat 等旧日志。后续应按模块逐步收敛，避免一次性机械替换带来噪音。
- JSON 日志适合平台采集，但本地 CLI 可读性会下降；后续可以按环境配置 text/json handler。

