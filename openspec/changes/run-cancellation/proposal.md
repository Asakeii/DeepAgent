# Run 取消控制

## 背景

成熟 Agent 项目需要让用户能停止长时间运行的任务。当前项目已经有 `runs` 表、`run_id`、SSE run events 和心跳保活，但缺少显式取消能力；如果研究任务、工具调用或模型流式输出耗时较长，用户只能关闭页面，服务端仍可能继续执行。

## 目标

- 新增 `POST /api/runs/cancel`，允许用户取消自己拥有的 running run。
- 将取消请求写入共享 MySQL `runs` 状态，支持取消请求命中任意 pod。
- ChatService 在执行期间监听共享 run 状态，并在检测到取消后 cancel 当前运行 context。
- run 最终状态保持为 `cancelled`，不会被 defer completion 覆盖成 succeeded/failed。
- 记录 `run_cancelled` run event，便于前端、回放和审计识别取消动作。

## 非目标

- 不实现 step-level retry。
- 不实现强制终止已经发出的外部 API 请求；取消通过 Go context 传递，依赖下游组件遵守 context。
- 不引入进程内 run registry 或 sticky session。

## 方案反思

- 轮询 MySQL 的取消延迟取决于 poll interval，本阶段选择简单可靠的跨 pod 方案。后续可以用 Redis Pub/Sub 或数据库通知降低延迟，但不能牺牲共享状态的最终一致性。
- 取消状态能阻止最终状态被覆盖，但已经完成的工具副作用无法自动回滚；后续需要为 write/dangerous 工具补幂等键、补偿和人工确认。
- 取消接口当前只针对 run，不针对 plan step；后续前端成熟度提升时应增加 step retry / step cancel。

