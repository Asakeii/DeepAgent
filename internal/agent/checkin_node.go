package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/tool"
)

// loadCheckinMsg 是 Checkin 子图的 load 节点。
// 从 State.Messages 中取出用户最新消息，加载历史，构造 checkin agent 的输入。
func loadCheckinMsg(ctx context.Context, input string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		threadID := state.ThreadID
		if threadID == "" {
			threadID = "console-default"
		}

		// 加载历史消息（跨会话记忆）
		history, histErr := infra.RecentMessagesForCheckin(ctx, threadID, 20)
		if histErr != nil {
			history = nil
		}

		// 取用户最新的那条 user message
		var userMsg string
		for i := len(state.Messages) - 1; i >= 0; i-- {
			if state.Messages[i].Role == schema.User {
				userMsg = state.Messages[i].Content
				break
			}
		}

		output = append(history, schema.UserMessage(userMsg))
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.User), userMsg)
		return nil
	})
	return output, err
}

// routerCheckin 是 Checkin 子图的 router 节点。
func routerCheckin(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		threadID := state.ThreadID
		if threadID == "" {
			threadID = "console-default"
		}
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.Assistant), input.Content)
		state.Goto = compose.END
		output = compose.END
		return nil
	})
	return output, err
}

// NewCheckinNode 构造 Checkin 子图：START -> load -> agent(ReAct) -> router -> END
func NewCheckinNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	checkinAgent, err := NewCheckinAgent(ctx)
	if err != nil {
		fmt.Printf("WARNING: NewCheckinAgent failed: %v\n", err)
		return g
	}

	// 用 AnyLambda wrapper：从 state.ThreadID 注入 context，让 tools 在运行时
	// 通过 tool.ThreadIDFromCtx(ctx) 读取正确的 threadID，而非编译时闭包捕获。
	agentLambda, err := compose.AnyLambda(
		func(ctx context.Context, msgs []*schema.Message, opts ...any) (*schema.Message, error) {
			var tid string
			_ = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
				tid = state.ThreadID
				return nil
			})
			agentCtx := ctx
			if tid != "" {
				agentCtx = tool.WithThreadID(ctx, tid)
			}
			return checkinAgent.Generate(agentCtx, msgs)
		},
		func(ctx context.Context, msgs []*schema.Message, opts ...any) (*schema.StreamReader[*schema.Message], error) {
			var tid string
			_ = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
				tid = state.ThreadID
				return nil
			})
			agentCtx := ctx
			if tid != "" {
				agentCtx = tool.WithThreadID(ctx, tid)
			}
			return checkinAgent.Stream(agentCtx, msgs)
		},
		nil, nil,
	)
	if err != nil {
		fmt.Printf("WARNING: AnyLambda failed: %v\n", err)
		return g
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadCheckinMsg))
	_ = g.AddLambdaNode("agent", agentLambda)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerCheckin))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
