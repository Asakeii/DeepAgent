# Artifact PDF Export

## 背景

成熟 Agent 项目生成的研究报告需要可交付、可归档。当前 artifact 已支持 HTML 导出，但 PDF 仍缺失。中文 PDF 的难点不在接口，而在排版和字体：手写 PDF 字体嵌入容易不稳定，也会重复实现浏览器已经成熟处理的 HTML/CSS 排版能力。

## 变更

- `/api/artifact-exports` 支持 `format=pdf`。
- PDF 导出复用现有 Markdown -> HTML 模板，再通过可配置 Headless Chrome/Chromium 渲染为 PDF。
- 新增服务端配置：
  - `server.pdf_renderer_command`
  - `server.pdf_renderer_args`
  - `server.pdf_renderer_timeout_seconds`
- 渲染参数支持 `{{input}}`、`{{input_path}}`、`{{output}}` 占位符。
- 没有显式配置 renderer 时，服务尝试发现常见 Chrome/Chromium 命令。
- 渲染失败或未配置时返回明确错误，不影响 HTML 导出。

## 反思与边界

- 本阶段不在 Go 中实现 PDF 字体排版引擎，避免中文字体、分页、表格和代码块渲染质量不可控。
- PDF 渲染依赖部署环境存在 Chrome/Chromium；容器化部署应显式安装并配置 renderer。
- PDF 只支持已授权用户/团队可读的 artifact，不改变现有 artifact ownership/team access 边界。
- 后续可加入异步导出任务、导出缓存和水印。
