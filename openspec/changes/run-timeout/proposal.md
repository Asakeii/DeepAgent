# Agent Run 超时控制

## 背景

路线图在可靠性与并发中明确提出 `graph timeout`。当前工具层已经有 timeout wrapper，但一次 Agent run 可能卡在规划、模型流式输出、外部工具组合或 check-in agent 中。没有整次运行上限时，多 pod 部署下很难控制资源占用和用户等待时间。

## 目标

- 新增 `setting.run_timeout_seconds`，用于配置整次 Agent run 的最大执行时间。
- 使用 Go `context.WithTimeout` 约束 ChatService 传给 Research / Checkin / Tool runtime 的上下文。
- 超时时向用户返回明确的 timeout 错误事件。
- run 最终状态通过现有 `RunEventWriter` 标记为 failed，并保留 run event 记录。
- 默认值为 0，表示不改变现有行为；生产环境可在配置中启用。

## 非目标

- 不改变单个工具的 timeout 策略。
- 不做 step-level timeout 或自动重试。
- 不强制中断已经无法响应 context 的外部依赖。

## 方案反思

- 选用标准 `context.WithTimeout`，因为 Eino graph、ReAct agent 和工具运行都已经沿用 context，不需要新增调度器或本地状态。
- 默认不启用 timeout 是为了避免突然影响已有长研究任务；示例配置给出生产建议值，后续可按用户套餐或任务类型动态配置。
- 超时当前统一计为 failed，后续如果需要更细的错误分类，可以给 runs 增加 `failure_reason=timeout` 或独立状态。

