# 模型 Profile 路由

## 背景

成熟 Agent 产品通常需要在速度、质量、成本之间做模型选择。当前系统只有一个默认 ChatModel、一个 PlanModel 和一个 VisionModel，无法按用户偏好或请求选择不同模型。与此同时，Eino 已经提供 ChatModel 抽象和 graph context 传递能力，本阶段应复用现有模型组件，不重写模型调用框架。

## 变更

- 在 `model.profiles` 中配置命名模型 profile，例如 `fast`、`deep`。
- 每个 profile 在启动时初始化自己的 ChatModel 和 PlanModel；缺失的 `api_key/base_url` 继承默认模型配置。
- 请求体和用户设置支持 `model_profile`。
- ChatService 将最终选择写入 run context，Research、Planner、Reporter、Coder、Checkin 和 Vision 路径从 context-aware getter 读取模型。
- 未知 profile 在模型执行前被拒绝，并记录为 failed run。

## 反思与边界

- 不在本阶段做 A/B 流量分配、自动模型评分或管理后台；这些属于运营治理能力，需要更多指标闭环。
- 不在请求期间修改全局 `infra.ChatModel`，避免并发请求串线。
- Vision profile 目前随所选 profile 回退其 ChatModel；默认 profile 仍支持独立 `vision_model`。
- profile 配置是进程启动配置，变更需要重启或后续配置中心能力。
