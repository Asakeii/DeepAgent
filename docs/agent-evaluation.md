# Agent 评测体系

## 目标

评测体系用于防止 prompt、模型、工具 schema、路由规则改动后出现隐性退化。当前阶段先提供离线 eval 基座：用固定用例描述期望，再用运行观测结果进行评分。

## 当前能力

- 用例文件：`evals/cases/routing.jsonl`
- 观测文件：JSONL，每行一个 case 的实际运行结果。
- Runner：`go run ./cmd/evalrun -cases evals/cases/routing.jsonl -observations <observations.jsonl>`
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

## 设计取舍

- 先做离线评分，不在 CI 中直接调用真实模型，避免外部模型、网络和数据库波动导致测试不稳定。
- 观测结果可以来自本地手工记录、线上 `runs/run_events/tool_audit_logs` 导出，或后续 live runner 自动生成。
- 工具评分只检查必需工具和禁用工具，避免把模型的合理多步调用误判为失败。

## 后续计划

- 增加 live eval runner：按 case 调用 `ChatService` 并自动采集 run events/tool audits。
- 增加引用质量评测：检查报告是否有引用、引用是否可访问、引用是否支持结论。
- 增加成本和延迟指标：结合模型 token、run_events 和 tool_audit_logs 汇总。
- 增加 golden answer / rubric：对复杂研究报告做 LLM-as-judge 或人工评分。
