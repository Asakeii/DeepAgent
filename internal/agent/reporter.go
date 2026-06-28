package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// RunReporter 是教学阶段的独立 Reporter。
// 它基于已完成的 Plan 和各 step 的 ExecutionRes 生成最终报告。
func RunReporter(ctx context.Context, state *model.State) (string, error) {
	messages, err := loadReporterMessages(ctx, state)
	if err != nil {
		return "", err
	}

	resp, err := infra.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("call reporter model: %w", err)
	}

	return routeReporterResult(ctx, state, resp)
}

func loadReporterMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	if state.CurrentPlan == nil {
		return nil, fmt.Errorf("current plan is nil")
	}

	sysPrompt, err := infra.GetPromptTemplate(ctx, "reporter")
	if err != nil {
		return nil, err
	}

	promptTemp := prompt.FromMessages(
		schema.Jinja2,
		schema.SystemMessage(sysPrompt),
		schema.MessagesPlaceholder("user_input", true),
	)

	userMsgs := []*schema.Message{
		schema.UserMessage(fmt.Sprintf(
			"# Research Requirements\n\n## Task\n\n %s \n\n## Description\n\n %s",
			state.CurrentPlan.Title,
			state.CurrentPlan.Thought,
		)),
		schema.SystemMessage("IMPORTANT: Structure your report according to the format in the prompt. Remember to include key points, overview, detailed analysis, and key citations. Use markdown tables when useful."),
	}

	for _, step := range state.CurrentPlan.Steps {
		if step.ExecutionRes == nil {
			continue
		}

		userMsgs = append(userMsgs, schema.UserMessage(fmt.Sprintf(
			"Below are some observations for the research task:\n\n%s",
			*step.ExecutionRes,
		)))
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

func routeReporterResult(ctx context.Context, state *model.State, input *schema.Message) (string, error) {
	// Reporter 是当前任务的终点：写 END 后，总图不再继续路由。
	state.Goto = compose.END
	return strings.TrimSpace(input.Content), nil
}

func loadReporterMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadReporterMessages(ctx, state)
		return err
	})
	return output, err
}

func routerReporter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = routeReporterResult(ctx, state, input)
		return err
	})
	return output, err
}

// NewReporter 构造与 deer-go 对齐的 Reporter 子图：
//
//	START -> load -> agent -> router -> END
func NewReporter[I, O any](ctx context.Context) *compose.Graph[I, O] {
	return buildLoadAgentRouter(
		compose.InvokableLambdaWithOption(loadReporterMsg),
		withChatModel[I, O](infra.ChatModel),
		compose.InvokableLambdaWithOption(routerReporter),
	)
}
