package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/model"
)

// agentHandOff 是总图的分支路由函数。
// 每个 Agent 节点执行完后会写入 state.Goto；本函数读取 Goto，告诉 Eino 下一跳去哪个节点。
// 这是 deer-go 的「轻量路由协议」：节点不直接调用下一个节点，只改 State，由总图统一调度。
func agentHandOff(ctx context.Context, input string) (string, error) {
	next := compose.END

	err := compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		next = state.Goto
		return nil
	})

	return next, err
}

// Builder 组装 DeepAgent 总图并编译为 Runnable。
//
// 图结构示意：
//
//	START -> Coordinator -> Planner -+-> Human -> ...
//	                               +-> ResearchTeam -> Researcher/Coder -> ...
//	                               +-> Reporter -> END
//
// genFunc 由调用方传入，负责创建/初始化每次运行共享的 *model.State。
// 每个 Agent 都通过 AddGraphNode 接入子图，子图内部再拆成 load/agent/router 或 router。
func Builder(ctx context.Context, genFunc compose.GenLocalState[*model.State]) (compose.Runnable[string, string], error) {
	g := compose.NewGraph[string, string](
		compose.WithGenLocalState(genFunc),
	)

	// outMap 声明每个分支节点允许跳转的目标；agentHandOff 的返回值必须在此 map 中。
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

	// 每个节点后接 Branch：执行 agentHandOff 读 state.Goto，决定实际下一跳
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
		compose.WithCheckPointStore(model.NewDeepAgentCheckPoint(ctx)),
	)
}
