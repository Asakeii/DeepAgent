package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	deeptool "deepAgent/internal/tool"
	"deepAgent/internal/toolruntime"
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
		schema.SystemMessage("IMPORTANT: DO NOT include inline citations in the text. Instead, track all sources and include a References section at the end using link reference format. Include an empty line between each citation for better readability. Use this format for each reference:\n- [Source Title](URL)\n\n- [Another Source](URL)"),
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

// routeResearcherResult 把 Researcher 的最终输出写回当前未完成 Step。
// 写完后回到 ResearchTeam，由 ResearchTeam 决定继续执行下一个 Step 还是进入 Reporter。
func routeResearcherResult(ctx context.Context, state *model.State, input *schema.Message) error {
	_, idx, err := findCurrentStep(state)
	if err != nil {
		return err
	}

	result := strings.Clone(input.Content)
	state.CurrentPlan.Steps[idx].ExecutionRes = &result
	state.Goto = consts.ResearchTeam
	return nil
}

// loadResearcherMsg 是 Researcher 子图的 load 节点。
func loadResearcherMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadResearcherMessages(ctx, state)
		return err
	})
	return output, err
}

// routerResearcher 是 Researcher 子图的 router 节点。
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

// modifyResearcherInput 在 ReAct 多轮调用前裁剪过长消息。
// 工具结果可能很长，裁剪尾部能保留最近上下文并降低模型上下文溢出风险。
func modifyResearcherInput(ctx context.Context, input []*schema.Message) []*schema.Message {
	maxLimit := 50000

	for i := range input {
		if input[i] == nil {
			continue
		}

		l := len(input[i].Content)
		if l > maxLimit {
			input[i].Content = input[i].Content[l-maxLimit:]
		}
	}

	return input
}

// NewResearcher 构造 Researcher 子图，与 deer-go 对齐为 load -> ReAct agent -> router。
// Researcher 会挂载当前已初始化的所有 MCP tools，用于搜索、抓取或其他外部信息收集。
func NewResearcher[I, O any](ctx context.Context) *compose.Graph[I, O] {
	researchTools := []tool.BaseTool{}

	// Native web search tools (SearXNG) — self-hosted, no API key required.
	if ws, err := deeptool.NewWebSearchTool(ctx); err == nil {
		researchTools = append(researchTools, ws)
	}
	if wf, err := deeptool.NewWebFetchTool(ctx); err == nil {
		researchTools = append(researchTools, wf)
	}
	researchTools = toolruntime.WrapTools(infra.DB, researchTools, toolruntime.DefaultPolicy())

	// ReAct Agent 会在“模型思考 -> 工具调用 -> 观察结果”之间循环，直到得到最终回答或达到 MaxStep。
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:               40,
		ToolCallingModel:      infra.ChatModel,
		ToolsConfig:           compose.ToolsNodeConfig{Tools: researchTools},
		MessageModifier:       modifyResearcherInput,
		StreamToolCallChecker: toolCallChecker,
	})
	if err != nil {
		panic(err)
	}

	// AnyLambda 把 ReAct Agent 的 Generate/Stream 能力包装成 Eino Graph 节点。
	agentLambda, err := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		panic(err)
	}

	return buildLoadAgentRouter(
		compose.InvokableLambdaWithOption(loadResearcherMsg),
		withLambda[I, O](agentLambda),
		compose.InvokableLambdaWithOption(routerResearcher),
	)
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
