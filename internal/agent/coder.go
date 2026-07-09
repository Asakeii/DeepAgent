package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/toolruntime"
)

// RunCoder 是教学阶段的独立 Coder（不经过 Eino 子图）。
// main.go 手动 for/switch 时可直接调用；接总图后应走 NewCoder 子图。
func RunCoder(ctx context.Context, state *model.State) error {
	messages, err := loadCoderMessages(ctx, state)
	if err != nil {
		return err
	}

	// 简化版：单次 ChatModel 调用，不接 MCP tool
	resp, err := infra.ChatModelFor(ctx).Generate(ctx, messages)
	if err != nil {
		return fmt.Errorf("call coder model: %w", err)
	}

	return routeCoderResult(ctx, state, resp)
}

// loadCoderMessages 组装 Coder 的 prompt：系统提示 + 当前 processing step 任务描述。
func loadCoderMessages(ctx context.Context, state *model.State) ([]*schema.Message, error) {
	curStep, _, err := findCurrentStep(state)
	if err != nil {
		return nil, err
	}

	sysPrompt, err := infra.GetPromptTemplate(ctx, "coder")
	if err != nil {
		return nil, err
	}

	promptTemp := prompt.FromMessages(
		schema.Jinja2,
		schema.SystemMessage(sysPrompt),
		schema.MessagesPlaceholder("user_input", true),
	)

	// 把当前 step 的 title/description 拼成用户消息，作为本次数据处理/计算任务
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

// routeCoderResult 把模型最终输出写入当前 step 的 ExecutionRes，并回到 ResearchTeam 继续调度。
func routeCoderResult(ctx context.Context, state *model.State, input *schema.Message) error {
	_, idx, err := findCurrentStep(state)
	if err != nil {
		return err
	}

	// 复制一份字符串再存入 State，避免直接持有 Message 字段地址。
	result := strings.Clone(input.Content)
	state.CurrentPlan.Steps[idx].ExecutionRes = &result
	state.Goto = consts.ResearchTeam
	return nil
}

// loadCoderMsg 是 Coder 子图「load」节点：在 Graph 上下文中读取 State，生成 []*schema.Message。
func loadCoderMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		output, err = loadCoderMessages(ctx, state)
		return err
	})
	return output, err
}

// routerCoder 是 Coder 子图「router」节点：解析 agent 输出，写 State，返回下一跳节点名。
func routerCoder(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := routeCoderResult(ctx, state, input); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// modifyCoderInput 是 ReAct Agent 的 MessageModifier：调模型前裁剪过长消息，防止 context 爆掉。
// ReAct 多轮 tool call 后单条消息可能很大（如 execute_python 输出），超过 50000 字符则只保留尾部。
func modifyCoderInput(ctx context.Context, input []*schema.Message) []*schema.Message {
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

// toolCallChecker 判断流式输出里是否出现了 tool call，供 ReAct 决定是否进入 Tools 节点。
func toolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

// NewCoder 构造 Coder 子图，与 deer-go 对齐：load -> ReAct agent -> router。
//
// 子图流程：
//
//	load    读取 State，拼 prompt（当前 processing step）
//	agent   ReAct 循环：模型可多次调用 Python MCP tools（execute_python 等）
//	router  把 agent 最终回复写入 ExecutionRes，Goto = ResearchTeam
func NewCoder[I, O any](ctx context.Context) *compose.Graph[I, O] {
	// 只挂载名字以 python 开头的 MCP server 的 tools，与 Researcher 的搜索 tools 隔离
	coderTools := []tool.BaseTool{}

	for name, cli := range infra.MCPServer {
		if !strings.HasPrefix(name, "python") {
			continue
		}

		// 把 MCP client 上的 tool 转成 Eino 的 tool.BaseTool，供 ReAct Agent 调用
		ts, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			continue
		}

		coderTools = append(coderTools, ts...)
	}
	coderTools = toolruntime.WrapTools(infra.DB, coderTools, toolruntime.DefaultPolicy())

	// ReAct Agent：模型 ↔ tool call 可多轮循环，最多 MaxStep 轮
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:               40,
		ToolCallingModel:      infra.ChatModelFor(ctx),
		ToolsConfig:           compose.ToolsNodeConfig{Tools: coderTools},
		MessageModifier:       modifyCoderInput,
		StreamToolCallChecker: toolCallChecker,
	})
	if err != nil {
		panic(err)
	}

	// 把 ReAct Agent 包成 Graph 节点（支持 Generate / Stream）
	agentLambda, err := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		panic(err)
	}

	return buildLoadAgentRouter(
		compose.InvokableLambdaWithOption(loadCoderMsg),
		withLambda[I, O](agentLambda),
		compose.InvokableLambdaWithOption(routerCoder),
	)
}
