package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/model"
)

// RunResearchTeam 是教学阶段的独立 ResearchTeam。
// 它不调用模型，只负责根据 CurrentPlan 中未完成的 step 决定下一跳。
func RunResearchTeam(ctx context.Context, state *model.State) error {
	state.Goto = consts.Planner

	if state.CurrentPlan == nil {
		return nil
	}

	for i := range state.CurrentPlan.Steps {
		step := state.CurrentPlan.Steps[i]
		if step.ExecutionRes != nil {
			continue
		}

		switch step.StepType {
		case model.Research:
			state.Goto = consts.Researcher
		case model.Processing:
			state.Goto = consts.Coder
		default:
			state.Goto = consts.Reporter
		}

		return nil
	}

	if state.PlanIterations >= state.MaxPlanIterations {
		state.Goto = consts.Reporter
		return nil
	}

	state.Goto = consts.Planner
	return nil
}

func routerResearchTeam(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		if err := RunResearchTeam(ctx, state); err != nil {
			return err
		}
		output = state.Goto
		return nil
	})
	return output, err
}

// NewResearchTeamNode 构造与 deer-go 对齐的 ResearchTeam 子图：
//
//	START -> router -> END
func NewResearchTeamNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerResearchTeam))
	_ = g.AddEdge(compose.START, "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
