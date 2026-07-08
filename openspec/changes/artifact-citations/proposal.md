# 报告引用存储

## 背景

报告 artifact 已经能独立持久化，但引用仍埋在 markdown 文本里。成熟 Agent 项目需要把引用作为结构化数据保存，才能支持 citation panel、引用质量评测、来源审计和后续导出。

## 目标

- 新增 `artifact_citations` 表，绑定 artifact、run、thread、user。
- 研究报告保存为 artifact 后，从 markdown 链接中提取引用并落库。
- 提供按 artifact 查询 citation 的只读接口。
- 复用现有用户边界：只能读取当前用户自己的 artifact citations。

## 非目标

- 不做网页抓取或引用内容二次验证。
- 不做可信度评分、引用质量模型评测。
- 不支持非 markdown 链接格式的复杂解析。

## 方案反思

- 当前提取器只识别标准 markdown link，足够覆盖 reporter prompt 的主路径；如果未来报告格式更自由，需要引入更完整的 markdown parser。
- 引用表没有外键约束，延续当前项目迁移风格，降低本地开发环境兼容风险；生产环境可以后续增加外键或定期一致性检查。
- citation 的质量判断没有塞进写入路径，避免把在线 run 延迟和外部网络验证耦合在一起。
