package model

// StepType 表示计划步骤的执行类型。
// research 交给 Researcher 收集信息，processing 交给 Coder 做计算/处理。
type StepType string

const (
	Research   StepType = "research"
	Processing StepType = "processing"
)

// Step 是 Planner 生成的单个执行步骤。
// ResearchTeam 会按顺序找到第一个 ExecutionRes 为空的 Step，并分派给 Researcher 或 Coder。
type Step struct {
	// NeedWebSearch 表示该步骤是否需要外部检索；后续接搜索 MCP 时会用到。
	NeedWebSearch bool `json:"need_web_search"`
	// Title 是步骤标题，主要用于 prompt 展示和调试。
	Title string `json:"title"`
	// Description 是步骤的详细任务说明，会传给对应 Agent 执行。
	Description string `json:"description"`
	// StepType 决定这个步骤由 Researcher 还是 Coder 执行。
	StepType StepType `json:"step_type"`
	// ExecutionRes 是步骤执行结果；nil 表示尚未执行。
	ExecutionRes *string `json:"execution_res,omitempty"`
}

// Plan 是 Planner 和后续执行节点之间的结构化契约。
// Planner 负责生成它，ResearchTeam/Researcher/Coder/Reporter 都围绕它工作。
type Plan struct {
	// Locale 表示本次任务的输出语言/地区。
	Locale string `json:"locale"`
	// HasEnoughContext 为 true 时，Planner 认为无需继续研究，可直接进入 Reporter。
	HasEnoughContext bool `json:"has_enough_context"`
	// Thought 是 Planner 对用户需求和上下文是否足够的判断。
	Thought string `json:"thought"`
	// Title 是整份研究任务的标题。
	Title string `json:"title"`
	// Steps 是后续要逐个执行的信息收集/处理步骤。
	Steps []Step `json:"steps"`
}
