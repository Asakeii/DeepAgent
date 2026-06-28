package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// NewMCPBridge creates an MCP server with deepAgent tools that OpenClaw can call.
// The server exposes a `chat` tool that wraps the full Coordinator graph routing.
func NewMCPBridge() *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer(
		"deepAgent",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	s.AddTool(
		mcp.NewTool("chat",
			mcp.WithDescription("Send a message to deepAgent for processing. Supports research tasks, daily checkins (exercise/diet/study), food image analysis, reminders, and casual chat. The agent will auto-route to the correct handler."),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The user's message text"),
			),
			mcp.WithString("thread_id",
				mcp.Description("Unique session identifier for conversation continuity (optional)"),
			),
		),
		handleChat,
	)

	s.AddTool(
		mcp.NewTool("list_capabilities",
			mcp.WithDescription("List all capabilities of deepAgent"),
		),
		handleListCapabilities,
	)

	return s
}

func handleChat(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		args = map[string]any{}
	}
	msg, _ := args["message"].(string)
	if strings.TrimSpace(msg) == "" {
		return mcp.NewToolResultError("message is required"), nil
	}

	threadID, _ := args["thread_id"].(string)
	if threadID == "" {
		threadID = "openclaw-default"
	}

	initMsg := msg
	initThreadID := threadID
	opts := []compose.Option{
		compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
			st := s.(*model.State)
			st.Messages = []*schema.Message{schema.UserMessage(initMsg)}
			st.ThreadID = initThreadID
			return nil
		}),
	}

	r := agent.GetAgent()

	// Collect stream output
	var err error
	var sb strings.Builder
	outChan := make(chan string)
	done := make(chan struct{})

	go func() {
		for s := range outChan {
			sb.WriteString(s)
		}
		close(done)
	}()

	_, err = r.Stream(ctx, consts.Coordinator,
		append(opts, compose.WithCallbacks(&infra.LoggerCallback{
			ID:  threadID,
			Out: outChan,
		}))...,
	)
	close(outChan)
	<-done

	if err != nil {
		// If the graph returned a partial response, include it
		if sb.Len() > 0 {
			return mcp.NewToolResultText(fmt.Sprintf("%s\n\n[graph ended: %v]", sb.String(), err)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("graph run: %v", err)), nil
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		result = "processed"
	}

	return mcp.NewToolResultText(result), nil
}

func handleListCapabilities(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	caps := map[string]any{
		"capabilities": []string{
			"research: deep investigation with web search and report generation",
			"checkin: daily exercise/diet/study tracking",
			"food_analysis: image-based food calorie estimation",
			"reminders: cron-based scheduled reminders",
			"casual_chat: greetings and small talk",
		},
	}
	b, _ := json.MarshalIndent(caps, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

// NewMCPSSEServer creates an SSE transport wrapper for the MCP server.
// Uses default endpoints: GET /sse (SSE stream), POST /message (JSON-RPC)
func NewMCPSSEServer() *mcpserver.SSEServer {
	bridge := NewMCPBridge()
	return mcpserver.NewSSEServer(bridge)
}

// StartMCPServer starts the MCP SSE server on the given address.
// Returns the server for graceful shutdown.
func StartMCPServer(addr string) *mcpserver.SSEServer {
	sse := NewMCPSSEServer()
	go func() {
		if err := sse.Start(addr); err != nil && err != io.EOF {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()
	return sse
}
