# 运行指标汇总

## 背景

项目已经有 `runs`、`run_events` 和 `tool_audit_logs`，但还缺面向运营和排障的聚合指标。成熟 Agent 服务需要能快速回答：最近 run 成功率是多少、延迟如何、工具失败率多少。

## 方案

- 新增 `GET /api/metrics/runs?window_hours=24`。
- 按当前请求 `user_id` 汇总，避免跨用户数据泄露。
- 复用已有 MySQL 数据，不引入新指标系统。
- 输出：
  - run 总数、成功数、失败数、运行中数量。
  - run 成功率。
  - run 平均延迟和 P95 延迟。
  - tool 总数、失败数、阻断数。
  - tool 错误率和平均耗时。

## 方案反思

- 这是轻量运营 API，不替代 Prometheus/OpenTelemetry。
- P95 在应用侧计算，适合窗口内轻量查询；高流量场景应改为预聚合或指标系统。
- 当前未统计 token/cost，因为模型回调里还没有统一 token usage 记录。
