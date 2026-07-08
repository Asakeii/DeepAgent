# 报告产物持久化

## 背景

研究型 Agent 的最终输出目前只作为 assistant message 保存在会话历史中。对成熟 Agent 产品来说，报告是一种长期产物，需要能被独立查询、版本化、后续导出或分享，而不是只能从消息流里反查。

## 目标

- 新增 `artifacts` 持久化表，作为 report、后续 citation/export/share 的共同地基。
- 研究 run 完成后，将最终报告保存为 markdown report artifact。
- 提供用户范围内的 artifact 查询接口，支持按 thread 和 kind 过滤。
- 保持服务无状态：artifact 只落 MySQL，不依赖进程内缓存。

## 非目标

- 本阶段不实现 PDF/HTML 导出。
- 本阶段不实现分享链接和权限扩展。
- 本阶段不拆 citation 表；引用结构化会作为后续增量。
- 本阶段不改前端报告管理界面。

## 方案反思

- 版本号当前按同一 `thread_id/kind/title` 的最大版本递增，足够支撑单次 report 沉淀，但如果未来支持多人协作编辑，应引入更明确的 `artifact_key` 或独立 version 表。
- 现在直接保存 markdown 内容到 MySQL `MEDIUMTEXT`，实现简单且符合当前报告体量；如果报告包含大附件、图片或多格式导出，应迁移到对象存储加元数据索引。
- Eino 负责 Agent 编排和回调，但不提供业务产物管理；这里不重复造 Agent 框架，只在应用层把最终输出沉淀为产品数据。
