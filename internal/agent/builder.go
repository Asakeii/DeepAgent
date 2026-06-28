package agent

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// ---------------------------------------------------------------
// 全局单次编译的 Runnable（compile-once）。
// Handler 通过 GetAgent() 获取，用 WithStateModifier 注入请求级 State。
// ---------------------------------------------------------------

var (
	globalRunnable   compose.Runnable[string, string]
	globalRunnableMu sync.Mutex
)

// GetAgent returns the compile-once global Runnable.
// Call InitAgent first during server startup.
func GetAgent() compose.Runnable[string, string] {
	globalRunnableMu.Lock()
	defer globalRunnableMu.Unlock()
	return globalRunnable
}

// InitAgent compiles the full agent graph once. Must be called after InitDB/InitModel/InitMCP.
func InitAgent(ctx context.Context) error {
	globalRunnableMu.Lock()
	defer globalRunnableMu.Unlock()
	if globalRunnable != nil {
		return nil // already compiled
	}
	r, err := compileGraph(ctx)
	if err != nil {
		return err
	}
	globalRunnable = r
	return nil
}

// ---------------------------------------------------------------
// 图路由与编译
// ---------------------------------------------------------------

// agentHandOff is the main graph branch function: reads state.Goto to decide next hop.
func agentHandOff(ctx context.Context, input string) (string, error) {
	next := compose.END
	err := compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		next = state.Goto
		return nil
	})
	return next, err
}

// compileGraph builds and compiles the complete agent graph once.
// genFunc creates a default State; request details are injected via WithStateModifier at Stream time.
func compileGraph(ctx context.Context) (compose.Runnable[string, string], error) {
	defaultGenFunc := func(ctx context.Context) *model.State {
		return &model.State{
			Goto:                          consts.Coordinator,
			Locale:                        "zh-CN",
			MaxPlanIterations:             1,
			MaxStepNum:                    3,
			AutoAcceptedPlan:              true,
			EnableBackgroundInvestigation: false,
		}
	}

	g := compose.NewGraph[string, string](
		compose.WithGenLocalState(defaultGenFunc),
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
