- [x] 为 `runs` 新增 `cancel_requested_at` schema 和 migration。
- [x] 新增 run 取消 Store 方法和取消状态查询。
- [x] 新增 `POST /api/runs/cancel` 接口并做 run ownership 校验。
- [x] ChatService 执行期间监听共享取消状态并取消运行 context。
- [x] 避免 `CompleteRun` 覆盖已取消 run 状态。
- [x] 补充 run cancellation 存储测试。

