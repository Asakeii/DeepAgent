# Agent 评测基座

## 背景

成熟 Agent 项目需要在 prompt、模型、路由、工具 schema 改动后快速发现退化。当前项目已经有 run events 和 tool audit logs，但缺少固定 eval dataset 和 runner。

## 方案

- 新增 JSONL eval case 格式，覆盖路由、工具调用和最终回答关键内容。
- 新增离线 runner，读取用例和观测结果后输出结构化评分。
- 第一阶段不直接调用真实模型，避免 CI 受模型、网络、数据库和外部工具波动影响。
- 观测结果后续可以由 live runner 从 `runs`、`run_events`、`tool_audit_logs` 自动导出。

## 方案反思

- 离线 runner 只能验证观测结果是否符合预期，不能自动产生观测；后续必须补 live runner。
- 当前评分规则偏确定性，适合路由和工具回归；复杂研究报告还需要 rubric 或人工/模型裁判。
- 数据集规模还小，只能作为回归地基，不能代表完整产品质量。
