package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/model"
)

func runBackgroundInvestigation(ctx context.Context, state *model.State) error {
	// TODO: After MCP is implemented, call a search tool here and write its result
	// to state.BackgroundInvestigationResults. For now this node keeps the graph
	// shape aligned with deer-go and passes control to Planner.
	state.BackgroundInvestigationResults = ""
	return nil
}

func backgroundSearch(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		return runBackgroundInvestigation(ctx, state)
	})
	return output, err
}

func routerBackgroundInvestigator(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		state.Goto = consts.Planner
		output = state.Goto
		return nil
	})
	return output, err
}

// NewBackgroundInvestigator 构造与 deer-go 对齐的 Background Investigator 子图：
//
//	START -> search -> router -> END
func NewBackgroundInvestigator[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("search", compose.InvokableLambdaWithOption(backgroundSearch))
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerBackgroundInvestigator))

	_ = g.AddEdge(compose.START, "search")
	_ = g.AddEdge("search", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
