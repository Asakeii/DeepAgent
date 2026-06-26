package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
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
			history = nil // 历史加载失败不阻断
		}

		// 取用户最新的那条 user message
		var userMsg string
		for i := len(state.Messages) - 1; i >= 0; i-- {
			if state.Messages[i].Role == schema.User {
				userMsg = state.Messages[i].Content
				break
			}
		}

		// 拼接：历史 + 当前用户消息
		output = append(history, schema.UserMessage(userMsg))

		// 记录用户消息到 messages 表
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.User), userMsg)

		return nil
	})
	return output, err
}

// routerCheckin 是 Checkin 子图的 router 节点。
// checkin 完成后记录 assistant 回复，然后结束。
func routerCheckin(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		threadID := state.ThreadID
		if threadID == "" {
			threadID = "console-default"
		}

		// 记录 assistant 回复到 messages 表
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.Assistant), input.Content)

		// checkin 完成后直接结束
		state.Goto = compose.END
		output = compose.END
		return nil
	})
	return output, err
}

// NewCheckinNode 构造 Checkin 子图：
//
//	START -> load -> agent(ReAct) -> router -> END
//
// 内部使用独立的 ReAct agent（带打卡工具集），不与研究图共享工具。
func NewCheckinNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	// threadID 从环境变量取（console 模式）或从 state 中取（图模式由 Coordinator 设置）。
	// 因为 ReAct agent 构造时需要 threadID 绑定工具闭包，这里先用环境变量兜底，
	// 图运行时 loadCheckinMsg 会从 state 取真正的 threadID。
	threadID := os.Getenv("DEEPAGENT_THREAD_ID")
	if threadID == "" {
		threadID = "console-default"
	}

	// 构造 ReAct checkin agent 作为 ChatModel 使用
	checkinAgent, err := NewCheckinAgent(ctx, threadID)
	if err != nil {
		// 构造失败不 panic，返回空图
		fmt.Printf("WARNING: NewCheckinAgent failed: %v\n", err)
		return g
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadCheckinMsg))
	_ = g.AddLambdaNode("agent", compose.InvokableLambdaWithOption(
		func(ctx context.Context, msgs []*schema.Message, opts ...any) (*schema.Message, error) {
			resp, err := checkinAgent.Generate(ctx, msgs)
			if err != nil {
				return nil, fmt.Errorf("checkin agent: %w", err)
			}
			return resp, nil
		},
	))
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerCheckin))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
