package agent

import (
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// buildLoadAgentRouter eliminates the repeated AddEdge ×4 boilerplate in agent sub-graphs.
// Callers create their own Lambdas and ChatModel, pass them to the appropriate helper.
func buildLoadAgentRouter[I, O any](
	loadLambda *compose.Lambda,
	agentNode func(g *compose.Graph[I, O]),
	routerLambda *compose.Lambda,
) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", loadLambda)
	agentNode(g)
	_ = g.AddLambdaNode("router", routerLambda)
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func withChatModel[I, O any](m model.BaseChatModel) func(g *compose.Graph[I, O]) {
	return func(g *compose.Graph[I, O]) {
		_ = g.AddChatModelNode("agent", m)
	}
}

func withLambda[I, O any](l *compose.Lambda) func(g *compose.Graph[I, O]) {
	return func(g *compose.Graph[I, O]) {
		_ = g.AddLambdaNode("agent", l)
	}
}
