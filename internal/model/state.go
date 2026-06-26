package model

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/store"
)

func init() {
	if err := compose.RegisterSerializableType[State]("DeepAgentState"); err != nil {
		panic(err)
	}
}

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
	// ThreadID 会话标识，用于 checkin agent 的跨会话记忆和 checkpoint 恢复
	ThreadID string `json:"thread_id,omitempty"`
}

func (s *State) MarshalJSON() ([]byte, error) {
	type Alias State
	return json.Marshal((*Alias)(s))
}

func (s *State) UnmarshalJSON(b []byte) error {
	type Alias State
	var tmp Alias
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*s = State(tmp)
	return nil
}

// NewDeepAgentCheckPoint 构造一个基于 MySQL 的无状态 CheckPointStore。
// Eino 会用 checkPointID 保存/读取中断时的 Graph 现场；服务化时通常使用 thread_id 作为 checkPointID。
//
// 这里通过依赖注入接收 *sql.DB，而不是在 model 包内直接读取 infra.DB：
// infra 包（llm.go/logger.go）已经 import internal/model，若 model 再 import infra，
// 会形成 model → infra → model 循环依赖。改为由调用方（agent/builder.go，已同时 import
// model 和 infra）把 infra.DB 显式传入，import 方向保持单向 infra → model / model → store，
// 无环。同时这也更符合无状态/可测试原则。
func NewDeepAgentCheckPoint(ctx context.Context, db *sql.DB) compose.CheckPointStore {
	return store.NewMySQLCheckPoint(db)
}
