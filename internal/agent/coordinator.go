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
// Coordinator 做意图分类：闲聊直接回答，研究任务 hand_to_planner，打卡/饮食/运动 hand_to_checkin。
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
// hand_to_planner → 研究流程，hand_to_checkin → 打卡流程，无 tool call → 结束。
func routeCoordinatorResult(ctx context.Context, state *model.State, input *schema.Message) error {
	state.Goto = compose.END

	if len(input.ToolCalls) == 0 {
		return nil
	}

	toolCall := input.ToolCalls[0]

	switch toolCall.Function.Name {
	case "hand_to_planner":
		argMap := map[string]string{}
		_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &argMap)
		if locale := argMap["locale"]; locale != "" {
			state.Locale = locale
		}
		if state.EnableBackgroundInvestigation {
			state.Goto = consts.BackgroundInvestigator
		} else {
			state.Goto = consts.Planner
		}

	case "hand_to_checkin":
		argMap := map[string]string{}
		_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &argMap)
		if locale := argMap["locale"]; locale != "" {
			state.Locale = locale
		}
		// 打卡任务不进研究图：sigal handler 层切到 checkin agent
		state.RouteToCheckin = true
		state.Goto = compose.END
	}

	return nil
}

// loadCoordinatorMsg 是 Coordinator 子图的 load 节点。
func loadCoordinatorMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadCoordinatorMessages(ctx, state)
		return err
	})
	return output, err
}

// routerCoordinator 是 Coordinator 子图的 router 节点。
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

// NewCoordinator 构造 Coordinator 子图：
//
//	START -> load -> agent -> router -> END
func NewCoordinator[I, O any](ctx context.Context) *compose.Graph[I, O] {
	// hand_to_planner：模型调用此工具表示"需要进入研究流程"
	handToPlanner := &schema.ToolInfo{
		Name: "hand_to_planner",
		Desc: "Handoff to planner agent to do research/investigation task.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_title": {
				Type:     schema.String,
				Desc:     "The title of the research task to be handed off.",
				Required: true,
			},
			"locale": {
				Type:     schema.String,
				Desc:     "The user's detected language locale (e.g., en-US, zh-CN).",
				Required: true,
			},
		}),
	}

	// hand_to_checkin：模型调用此工具表示"打卡/饮食/运动/自律/图片识别相关"
	handToCheckin := &schema.ToolInfo{
		Name: "hand_to_checkin",
		Desc: "Handoff to checkin coach agent for daily check-in tasks: exercise logging, diet tracking, study recording, food image analysis, and check-in history/summary queries.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"user_message": {
				Type:     schema.String,
				Desc:     "The user's original message to forward to checkin agent.",
				Required: true,
			},
			"locale": {
				Type:     schema.String,
				Desc:     "The user's detected language locale (e.g., en-US, zh-CN).",
				Required: true,
			},
		}),
	}

	// 给 Coordinator 绑定两个路由工具
	coordinatorModel, err := infra.ChatModel.WithTools([]*schema.ToolInfo{handToPlanner, handToCheckin})
	if err != nil {
		coordinatorModel = infra.ChatModel
	}

	return buildLoadAgentRouter(
		compose.InvokableLambdaWithOption(loadCoordinatorMsg),
		withChatModel[I, O](coordinatorModel),
		compose.InvokableLambdaWithOption(routerCoordinator),
	)
}
