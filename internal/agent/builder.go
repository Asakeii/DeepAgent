package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// agentHandOff is the main graph branch function: reads state.Goto to decide next hop.
// Matches deer-go's agentHandOff pattern exactly.
func agentHandOff(ctx context.Context, input string) (string, error) {
	next := compose.END
	_ = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		next = state.Goto
		return nil
	})
	return next, nil
}

// Builder compiles the agent graph per-request, matching deer-go's pattern.
// genFunc captures request-level data (Messages, ThreadID) at compile time,
// ensuring sub-graph nodes see the correct state via eino's GenLocalState propagation.
func Builder(ctx context.Context, genFunc compose.GenLocalState[*model.State]) (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string](
		compose.WithGenLocalState(genFunc),
	)

	outMap := map[string]bool{
		consts.Coordinator:            true,
		consts.Planner:                true,
		consts.Human:                  true,
		consts.ResearchTeam:           true,
		consts.Researcher:             true,
		consts.Coder:                  true,
		consts.Reporter:               true,
		consts.BackgroundInvestigator: true,
		compose.END:                   true,
	}

	_ = g.AddGraphNode(consts.Coordinator, NewCoordinator[string, string](ctx), compose.WithNodeName(consts.Coordinator))
	_ = g.AddGraphNode(consts.Planner, NewPlanner[string, string](ctx), compose.WithNodeName(consts.Planner))
	_ = g.AddGraphNode(consts.Human, NewHumanNode[string, string](ctx), compose.WithNodeName(consts.Human))
	_ = g.AddGraphNode(consts.ResearchTeam, NewResearchTeamNode[string, string](ctx), compose.WithNodeName(consts.ResearchTeam))
	_ = g.AddGraphNode(consts.Researcher, NewResearcher[string, string](ctx), compose.WithNodeName(consts.Researcher))
	_ = g.AddGraphNode(consts.Coder, NewCoder[string, string](ctx), compose.WithNodeName(consts.Coder))
	_ = g.AddGraphNode(consts.Reporter, NewReporter[string, string](ctx), compose.WithNodeName(consts.Reporter))
	_ = g.AddGraphNode(consts.BackgroundInvestigator, NewBackgroundInvestigator[string, string](ctx), compose.WithNodeName(consts.BackgroundInvestigator))

	_ = g.AddBranch(consts.Coordinator, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Planner, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Human, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.ResearchTeam, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Researcher, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Coder, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Reporter, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.BackgroundInvestigator, compose.NewGraphBranch(agentHandOff, outMap))

	_ = g.AddEdge(compose.START, consts.Coordinator)

	return g.Compile(
		ctx,
		compose.WithGraphName("DeepAgent"),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
		compose.WithCheckPointStore(model.NewDeepAgentCheckPoint(ctx, infra.DB)),
	)
}
