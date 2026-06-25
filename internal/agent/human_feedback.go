package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/model"
)

// RunHumanFeedback 是教学阶段的独立 Human Feedback。
// 参考 deer-go 的 human_feedback.go：自动接受时进入 ResearchTeam；否则等待用户确认。
func RunHumanFeedback(ctx context.Context, state *model.State) error {
	state.Goto = consts.ResearchTeam

	if state.AutoAcceptedPlan {
		state.InterruptFeedback = ""
		return nil
	}

	switch state.InterruptFeedback {
	case consts.AcceptPlan:
		state.Goto = consts.ResearchTeam
	case consts.EditPlan:
		state.Goto = consts.Planner
	default:
		// v0.9.7 标准中断 API：会自动带上 execution address、生成 InterruptSignal，
		// 框架据此存 checkpoint 并可被 handler 的 ExtractInterruptInfo 识别。
		// 旧版 compose.InterruptAndRerun 是 deprecated 裸 sentinel，不会被框架包装，
		// 导致既不存 checkpoint、handler 也识别不到（端到端验证暴露）。
		return compose.Interrupt(ctx, nil)
	}

	state.InterruptFeedback = ""
	return nil
}

func routerHuman(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := RunHumanFeedback(ctx, state); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewHumanNode 构造与 deer-go 对齐的 Human Feedback 子图：
//
//	START -> router -> END
func NewHumanNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerHuman))
	_ = g.AddEdge(compose.START, "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
