package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// RunResearcher 是教学阶段的独立 Researcher。
// 对当前计划中第一个未完成的 research step 调用 ChatModel，将结果写入 ExecutionRes，再回到 ResearchTeam。
func RunResearcher(ctx context.Context, state *model.State) error {
	messages, err := loadResearcherMessages(ctx, state)
	if err != nil {
		return err
	}

	resp, err := infra.ChatModel.Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("call researcher model: %w", err)
	}

	return routeResearcherResult(ctx, state, resp)
}

func loadResearcherMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	curStep, _, err := findCurrentStep(state)
	if err != nil {
		return nil, err
	}

	sysPrompt, err := infra.GetPromptTemplate(ctx, "researcher")
	if err != nil {
		return nil, err
	}

	promptTemp := prompt.FromMessages(
		schema.Jinja2,
		schema.SystemMessage(sysPrompt),
		schema.MessagesPlaceholder("user_input", true),
	)

	// 把当前 step 的 title/description 拼成用户消息，作为本次研究任务
	userMsgs := []*schema.Message{
		schema.UserMessage(fmt.Sprintf(
			"#Task\n\n##title\n\n %s \n\n##description\n\n %s \n\n##locale\n\n %s",
			curStep.Title,
			curStep.Description,
			state.Locale,
		)),
	}

	msgs, err := promptTemp.Format(ctx, map[string]any{
		"locale":              state.Locale,
		"max_step_num":        state.MaxStepNum,
		"max_plan_iterations": state.MaxPlanIterations,
		"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
		"user_input":          userMsgs,
	})
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func routeResearcherResult(ctx context.Context, state *model.State, input *schema.Message) error {
	_, idx, err := findCurrentStep(state)
	if err != nil {
		return err
	}

	state.CurrentPlan.Steps[idx].ExecutionRes = &input.Content
	state.Goto = consts.ResearchTeam
	return nil
}

func loadResearcherMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadResearcherMessages(ctx, state)
		return err
	})
	return output, err
}

func routerResearcher(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := routeResearcherResult(ctx, state, input); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewResearcher 构造与 deer-go 对齐的 Researcher 子图。
// 当前版本先使用 ChatModel 直接生成；后续会把 agent 节点升级为 ReAct + MCP tools。
func NewResearcher[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadResearcherMsg))
	_ = g.AddChatModelNode("agent", infra.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerResearcher))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}

// findCurrentStep 返回计划中第一个 ExecutionRes 为 nil 的 step 及其下标。
func findCurrentStep(state *model.State) (*model.Step, int, error) {
	if state.CurrentPlan == nil {
		return nil, 0, fmt.Errorf("current plan is nil")
	}

	for i := range state.CurrentPlan.Steps {
		if state.CurrentPlan.Steps[i].ExecutionRes == nil {
			return &state.CurrentPlan.Steps[i], i, nil
		}
	}

	return nil, 0, fmt.Errorf("no unfinished step found")
}
