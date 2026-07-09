# Agent 评测体系

## 目标

评测体系用于防止 prompt、模型、工具 schema、路由规则改动后出现隐性退化。当前阶段先提供离线 eval 基座：用固定用例描述期望，再用运行观测结果进行评分。

## 当前能力

- 用例文件：`evals/cases/routing.jsonl`
- 观测文件：JSONL，每行一个 case 的实际运行结果。
- Runner：`go run ./cmd/evalrun -cases evals/cases/routing.jsonl -observations <observations.jsonl>`
- A/B 对比：`go run ./cmd/evalcompare -cases evals/cases/routing.jsonl -baseline <baseline.jsonl> -candidate <candidate.jsonl>`
- 指标：
  - `routing_accuracy`
  - `tool_call_accuracy`
  - `final_answer_completeness`

## 用例格式

```json
{"id":"checkin_001","category":"checkin","input":"我今天跑步 5 公里，帮我记录一下","expected_route":"checkin","expected_tools":["hand_to_checkin","record_checkin"],"forbidden_tools":["web_search"]}
```

## 观测格式

```json
{"id":"checkin_001","route":"checkin","tools":["hand_to_checkin","record_checkin"],"final":"已记录运动打卡"}
```

## A/B 对比

`evalcompare` 用同一套 cases 分别评分 baseline 和 candidate observations，然后输出：

- baseline / candidate 的完整 suite result。
- `pass_rate_delta`。
- 各指标平均值差异 `average_deltas`。
- regressions：baseline 通过但 candidate 失败的 case。
- improvements：baseline 失败但 candidate 通过的 case。
- changed_scores：通过状态未变但指标分数发生变化的 case。

常见用法：

```bash
go run ./cmd/evalcompare \
  -cases evals/cases/routing.jsonl \
  -baseline evals/observations/default-model.jsonl \
  -candidate evals/observations/deep-profile.jsonl \
  -max-regressions 0 \
  -min-pass-rate-delta 0
```

这适合比较不同模型 profile、prompt 版本或工具 schema 变更。命令不调用在线模型，适合在 CI 或发布前检查中运行。

## 设计取舍

- 先做离线评分，不在 CI 中直接调用真实模型，避免外部模型、网络和数据库波动导致测试不稳定。
- 观测结果可以来自本地手工记录、线上 `runs/run_events/tool_audit_logs` 导出，或后续 live runner 自动生成。
- 工具评分只检查必需工具和禁用工具，避免把模型的合理多步调用误判为失败。
- A/B 对比先比较离线 observation，不做在线流量分流；这样可以先保护发布质量，再逐步接入线上实验平台。

## 后续计划

- 增加 live eval runner：按 case 调用 `ChatService` 并自动采集 run events/tool audits。
- 增加引用质量评测：检查报告是否有引用、引用是否可访问、引用是否支持结论。
- 增加成本和延迟指标：结合模型 token、run_events 和 tool_audit_logs 汇总。
- 增加 golden answer / rubric：对复杂研究报告做 LLM-as-judge 或人工评分。
