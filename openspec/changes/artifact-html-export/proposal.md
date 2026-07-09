# Artifact HTML 导出

## 背景

成熟 Agent 项目需要把研究报告等 artifact 导出为可归档、可发送、可打印的格式。当前系统已经能持久化 report artifact 和生成分享链接，但还缺少面向用户的导出接口。

## 变更

- 新增 `/api/artifact-exports?artifact_id=...&format=html`。
- 导出接口校验 artifact 归属，只允许用户导出自己的 artifact。
- 使用 Go 生态成熟 Markdown 渲染库 `goldmark` 将 artifact markdown 内容转换为 HTML。
- 返回完整 HTML 文档和 `Content-Disposition: attachment` 下载头。
- HTML 内置打印样式，可作为后续 PDF 打印/服务端 PDF 渲染的稳定输入。

## 反思与边界

- 本阶段只做 HTML 导出，不强行生成 PDF。当前后端没有可靠的中文字体、分页和排版方案，直接用简单 PDF 库会导致中文报告不可用或排版很差。
- Markdown 渲染复用 `goldmark`，不手写 Markdown parser。
- 导出结果即时从 MySQL artifact 读取，不保存进程内状态，支持多 pod。
- 后续 PDF 应基于 HTML 产物接入 headless browser 或具备中文字体嵌入能力的渲染服务。
