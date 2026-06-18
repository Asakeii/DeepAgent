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

func loadCoordinatorMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadCoordinatorMessages(ctx, state)
		return err
	})
	return output, err
}

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
