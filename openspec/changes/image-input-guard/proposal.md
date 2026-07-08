# 图片输入安全校验

## 背景

路线图在安全与 Prompt Injection 防护中明确提出 `image size/type limit`。当前 `/chat/stream` 支持 `image_base64`，最终会进入 VisionModel；但服务入口没有独立校验图片大小、MIME 类型，也没有阻止本地文件路径被当成图片输入传入。

## 目标

- 新增 `server.image_max_bytes` 和 `server.image_allowed_types` 配置。
- 在 ChatService 进入图片分析前校验图片输入。
- 支持 data URL、原始 base64 和 HTTP(S) URL；拒绝本地文件路径。
- 对 data URL / 原始 base64 校验解码后大小和图片类型。
- 校验失败时输出 `finish_reason=invalid_image` 的错误事件，并通过现有 run completion 标记失败。

## 非目标

- 不在本阶段实现通用 URL fetch allow/deny list。
- 不改变底层 `analyze_food` 工具的本地调试能力。
- 不引入对象存储上传链路。

## 方案反思

- 入口层无法在不下载的情况下验证 HTTP(S) 图片 URL 的真实大小和类型，本阶段只校验 URL scheme，具体下载仍由底层工具的 size limit 保护；后续应补 URL allow/deny list 与 SSRF 防护。
- 当前 body limit 仍可能比 image limit 更小，生产环境需要按上传方式一起配置 `max_body_bytes` 和 `image_max_bytes`。
- 本阶段优先拦截本地路径输入，避免服务端文件路径被外部请求直接传给图片读取逻辑。

