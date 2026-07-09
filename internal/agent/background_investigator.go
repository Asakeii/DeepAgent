package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	deeptool "deepAgent/internal/tool"
)

func runBackgroundInvestigation(ctx context.Context, state *model.State) error {
	searchTool, err := findSearchTool(ctx)
	if err != nil {
		// 与 deer-go 的目标一致：背景调查是增强项，不应该因为没有搜索工具阻断主流程。
		state.BackgroundInvestigationResults = ""
		return nil
	}

	if len(state.Messages) == 0 {
		state.BackgroundInvestigationResults = ""
		return nil
	}

	args := map[string]any{
		"query": state.Messages[len(state.Messages)-1].Content,
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("marshal background search args: %w", err)
	}

	result, err := searchTool.InvokableRun(ctx, string(argsBytes))
	if err != nil {
		return fmt.Errorf("run background search: %w", err)
	}

	state.BackgroundInvestigationResults = result
	return nil
}

func findSearchTool(ctx context.Context) (tool.InvokableTool, error) {
	if native, err := deeptool.NewWebSearchTool(ctx); err == nil {
		if searchTool, ok := native.(tool.InvokableTool); ok {
			return searchTool, nil
		}
	}

	for _, cli := range infra.MCPClientsForScope(ctx) {
		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			continue
		}

		for _, t := range tools {
			info, err := t.Info(ctx)
			if err != nil {
				continue
			}
			if !strings.HasSuffix(info.Name, "search") {
				continue
			}

			searchTool, ok := t.(tool.InvokableTool)
			if !ok {
				continue
			}
			return searchTool, nil
		}
	}

	return nil, fmt.Errorf("search tool not found")
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
