# deepAgent TODO

对照 deer-go 复现 deepAgent 的待办清单。基线：端到端已跑通（Coordinator → Planner → Human → ResearchTeam → Researcher/Coder → Reporter），agnes-2.0-flash + tavily MCP。

## 待办

### 可观测性：自部署 Langfuse（优先级高）
本地暂无 Docker，待具备 Docker 环境后执行。

- [ ] 用官方 docker-compose 起 Langfuse v3（依赖：Langfuse + ClickHouse + Postgres + Redis）
  - 参考：https://langfuse.com/self-hosting
  - 起来后访问 http://localhost:3000 创建项目，生成 pk-lf-... / sk-lf-... 及 Host
- [ ] 接入 eino-ext 官方 callback 包 `github.com/cloudwego/eino-ext/callbacks/langfuse`
  - 新建 `internal/infra/tracing.go`，实现 `InitLangfuse() func()`：
    从环境变量 `LANGFUSE_HOST` / `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY` 读取，
    未配置则跳过（不影响主流程），用 `callbacks.AppendGlobalHandlers(cb)` 注册，
    返回 shutdown 函数供 `main.go` defer 调用 flush。
  - 在 `main.go` 的 `runCLI` / `runServer` 里 `defer InitLangfuse()`，
    模式与 deer-go 的 `InitCozeLoopTracing()` 一致。
  - 验收：跑一次研究任务，Langfuse 控制台能看到 trace，含每步 LLM 的
    prompt/completion/token/延迟/成本，以及 MCP 工具调用。

### 体验：console 最终报告输出（优先级中）
- [ ] console 模式下 Reporter 的最终报告与各 step 搜索结果交错混排，看不到干净报告。
      改进 `LoggerCallback` 或 `runCLI`，让最终报告单独、清晰呈现。

### 对齐 deer-go 的行为分叉（可选，按需）
- [ ] ResearchTeam 对未知 step_type 的处理：deepAgent 是 `default→Reporter`，
      deer-go 是跳过该 step 继续找下一个。当前 Planner 只产 research/processing，
      不触发，可不动。
- [ ] Reporter system hint 文本：deepAgent 是精简版，deer-go 是长版本
      （5 点格式 + markdown table 示例）。对齐后报告更结构化。

### 测试（可选）
- [ ] 补 `internal/model/state_test.go`：State JSON 往返不丢字段、Plan 反序列化样例。
      deer-go 有 `biz/model/state_test.go`，复现指南 §1 也列为此验收项。
