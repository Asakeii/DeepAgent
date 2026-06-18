package model

import "github.com/cloudwego/eino/schema"

// State is the shared graph state.
// TODO: Add Eino message fields and runtime metadata when implementing the graph.

// State 是整张 Agent Graph 共享的运行状态。
// 后续每个节点都会读取/修改它，并通过 Goto 决定下一跳。
type State struct {
	// Messages 各 Agent 节点共享的对话历史
	Messages []*schema.Message `json:"messages,omitempty"`

	// Goto 下一跳节点名；当前节点完成后写入，总图读取并路由（见 internal/consts）
	Goto string `json:"goto,omitempty"`
	// Locale 用户语言/地域，由 Coordinator 识别
	Locale string `json:"locale,omitempty"`
	// CurrentPlan 当前结构化研究计划，Planner 生成，ResearchTeam/Reporter 等节点读写
	CurrentPlan *Plan `json:"current_plan,omitempty"`
	// PlanIterations 计划已迭代轮次，Planner 每重新规划一次递增
	PlanIterations int `json:"plan_iterations,omitempty"`
	// BackgroundInvestigationResults Background Investigator 的搜索结果，供 Planner 参考
	BackgroundInvestigationResults string `json:"background_investigation_results,omitempty"`
	// InterruptFeedback 人工确认反馈：accepted（接受）或 edit_plan（回 Planner 修改）
	InterruptFeedback string `json:"interrupt_feedback,omitempty"`

	// MaxPlanIterations 计划最大迭代轮次上限
	MaxPlanIterations int `json:"max_plan_iterations,omitempty"`
	// MaxStepNum 单次计划允许的最大步骤数
	MaxStepNum int `json:"max_step_num,omitempty"`
	// AutoAcceptedPlan 为 true 时跳过 Human Feedback，自动接受计划
	AutoAcceptedPlan bool `json:"auto_accepted_plan,omitempty"`
	// EnableBackgroundInvestigation 为 true 时，Coordinator 后先走背景调查再进 Planner
	EnableBackgroundInvestigation bool `json:"enable_background_investigation,omitempty"`
}
