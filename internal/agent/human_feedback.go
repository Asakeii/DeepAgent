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
		return compose.InterruptAndRerun
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
