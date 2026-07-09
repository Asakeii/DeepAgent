# A/B Eval 对比

## 背景

项目已经有离线 eval runner，可以用固定 cases 对一组 observations 打分。但成熟 Agent 项目在调整模型 profile、prompt、工具 schema 或路由策略时，更需要比较 baseline 与 candidate 的差异，快速发现回归。这类能力应先离线化，避免 CI 依赖实时模型、数据库或外部网络。

## 变更

- 新增 `evalharness.CompareSuites`，比较两组 suite result。
- 新增 `cmd/evalcompare`：
  - 输入同一套 cases。
  - 输入 baseline observations 和 candidate observations。
  - 输出 pass rate delta、指标平均值 delta、regressions、improvements、changed_scores。
  - 支持 `-max-regressions` 和 `-min-pass-rate-delta` 作为发布门禁。
- 更新 eval 文档，说明如何比较模型 profile 或 prompt 版本。

## 反思与边界

- 本阶段不做在线 A/B 流量分流；在线实验需要用户分桶、实验配置、样本量统计和隐私治理。
- 对比基于同一套 cases，不能用于跨数据集直接比较。
- 现阶段指标仍来自 observation JSONL；后续可从 run_events、tool_audit_logs 和 model_usage_logs 自动导出 observation。
