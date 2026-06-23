package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// loadCoordinatorMessages 负责构造 Coordinator 的输入消息。
// Coordinator 只做任务分类：闲聊直接回答，复杂研究任务通过工具调用 hand_to_planner 移交 Planner。
func loadCoordinatorMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	sysPrompt, err := infra.GetPromptTemplate(ctx, "coordinator")
	if err != nil {
		return nil, err
	}

	promptTemp := prompt.FromMessages(
		schema.Jinja2,
		schema.SystemMessage(sysPrompt),
		schema.MessagesPlaceholder("user_input", true),
	)

	return promptTemp.Format(ctx, map[string]any{
		"locale":              state.Locale,
		"max_step_num":        state.MaxStepNum,
		"max_plan_iterations": state.MaxPlanIterations,
		"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
		"user_input":          state.Messages,
	})
}

// routeCoordinatorResult 读取 Coordinator 的模型输出，并决定总图下一跳。
// 如果模型调用 hand_to_planner 工具，说明这是研究任务；否则默认结束。
func routeCoordinatorResult(ctx context.Context, state *model.State, input *schema.Message) error {
	state.Goto = compose.END

	if len(input.ToolCalls) == 0 {
		return nil
	}

	toolCall := input.ToolCalls[0]
	if toolCall.Function.Name != "hand_to_planner" {
		return nil
	}

	argMap := map[string]string{}
	_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &argMap)
	if locale := argMap["locale"]; locale != "" {
		state.Locale = locale
	}

	if state.EnableBackgroundInvestigation {
		state.Goto = consts.BackgroundInvestigator
		return nil
	}

	state.Goto = consts.Planner
	return nil
}

// loadCoordinatorMsg 是 Coordinator 子图的 load 节点。
// 子图节点通过 ProcessState 访问 Eino Graph 的共享 State。
func loadCoordinatorMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadCoordinatorMessages(ctx, state)
		return err
	})
	return output, err
}

// routerCoordinator 是 Coordinator 子图的 router 节点。
// 它把模型输出转换成 state.Goto，交给总图的 agentHandOff 继续路由。
func routerCoordinator(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := routeCoordinatorResult(ctx, state, input); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewCoordinator 构造与 deer-go 对齐的 Coordinator 子图：
//
//	START -> load -> agent -> router -> END
func NewCoordinator[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	// 这个工具不是给外部系统调用的真实工具，而是一个“路由信号”：
	// 模型一旦调用 hand_to_planner，router 就知道要把控制权交给 Planner。
	handToPlanner := &schema.ToolInfo{
		Name: "hand_to_planner",
		Desc: "Handoff to planner agent to do plan.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_title": {
				Type:     schema.String,
				Desc:     "The title of the task to be handed off.",
				Required: true,
			},
			"locale": {
				Type:     schema.String,
				Desc:     "The user's detected language locale (e.g., en-US, zh-CN).",
				Required: true,
			},
		}),
	}

	// 给 Coordinator 绑定 hand_to_planner，让模型可以用 tool call 表达“需要进入研究流程”。
	coordinatorModel, err := infra.ChatModel.WithTools([]*schema.ToolInfo{handToPlanner})
	if err != nil {
		coordinatorModel = infra.ChatModel
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadCoordinatorMsg))
	_ = g.AddChatModelNode("agent", coordinatorModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerCoordinator))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
