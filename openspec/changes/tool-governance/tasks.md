# Tasks

- [x] 新增工具审计表与 Store。
- [x] 基于 Eino 工具接口实现工具 runtime wrapper。
- [x] 增加默认风险分级、默认超时、危险工具阻断和错误归一化。
- [x] 将 Researcher、Coder、Checkin 工具列表接入治理 wrapper。
- [x] 在 ChatService / CheckinService 注入 run/thread/user 审计上下文。
- [x] OpenAI-compatible 入口创建 run 并传递 run_id。
- [x] 增加 `/api/tool-audits` 查询接口并复用 run ownership 校验。
- [x] 补充工具审计 Store 与 wrapper 单元测试。
- [ ] 后续：工具策略配置化与版本化。
- [ ] 后续：危险工具人工确认闭环。
- [ ] 后续：重试/熔断策略，仅对幂等工具启用。
