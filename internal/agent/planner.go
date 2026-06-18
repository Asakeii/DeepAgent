package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// RunPlanner 是教学阶段的独立 Planner。
// 后面接 Graph 时，会拆成 load -> agent -> router 三个节点。
func RunPlanner(ctx context.Context, state *model.State) error {
	messages, err := loadPlannerMessages(ctx, state)
	if err != nil {
		return err
	}

	resp, err := infra.PlanModel.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("call plan model: %w", err)
	}

	return routePlannerResult(ctx, state, resp)
}

// loadPlannerMessages 加载 Planner 的系统提示词和用户输入
func loadPlannerMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	sysPrompt, err := infra.GetPromptTemplate(ctx, "planner")
	if err != nil {
		return nil, err
	}

	promptTemp := prompt.FromMessages(
		schema.Jinja2,
		schema.SystemMessage(sysPrompt),
		schema.MessagesPlaceholder("user_input", true),
	)

	messages, err := promptTemp.Format(ctx, map[string]any{
		"locale":              state.Locale,
		"max_step_num":        state.MaxStepNum,
		"max_plan_iterations": state.MaxPlanIterations,
		"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
		"user_input":          state.Messages,
	})
	if err != nil {
		return nil, fmt.Errorf("format planner prompt: %w", err)
	}

	return messages, nil
}

func routePlannerResult(ctx context.Context, state *model.State, input *schema.Message) error {
	state.Goto = compose.END
	state.CurrentPlan = &model.Plan{}

	if err := json.Unmarshal([]byte(input.Content), state.CurrentPlan); err != nil {
		return fmt.Errorf("parse planner json: %w\nraw content: %s", err, input.Content)
	}

	state.PlanIterations++

	if state.CurrentPlan.HasEnoughContext {
		state.Goto = consts.Reporter
		return nil
	}

	state.Goto = consts.Human
	return nil
}

// loadPlannerMsg 是 Planner 子图里的 load 节点。
// 它不直接接收 *model.State，而是通过 compose.ProcessState 读取 Graph 共享状态。
func loadPlannerMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadPlannerMessages(ctx, state)
		return err
	})
	return output, err
}

// routerPlanner 是 Planner 子图里的 router 节点。
// 它解析 PlanModel 的输出，写入 state.CurrentPlan，并设置 state.Goto。
func routerPlanner(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := routePlannerResult(ctx, state, input); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewPlanner 构造与 deer-go 对齐的 Planner 子图：
//
//	START -> load -> agent -> router -> END
//
// 当前保留 RunPlanner 作为教学版直接调用入口；后续 Builder 会改为 AddGraphNode 使用此子图。
func NewPlanner[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadPlannerMsg))
	_ = g.AddChatModelNode("agent", infra.PlanModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerPlanner))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
