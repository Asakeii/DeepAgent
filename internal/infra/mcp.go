package infra

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"deepAgent/conf"
)

const (
	transportStdio = "stdio"
	transportSSE   = "sse"
)

// 维护多个MCP客户端，用于与不同的MCP服务器通信
// 多个MCP服务器可以做tool隔离、运行时隔离、安全权限隔离、独立配置
var MCPServer map[string]client.MCPClient

// InitMCP 初始化MCP客户端
func InitMCP(ctx context.Context) error {
	if conf.App == nil {
		return fmt.Errorf("config is not loaded")
	}

	clients := make(map[string]client.MCPClient)

	// 读取配置文件中的MCP服务器配置
	for name, server := range conf.App.MCP.Servers {
		transport := getMCPTransport(server)
		log.Printf("load mcp client: name=%s transport=%s", name, transport)

		mcpClient, err := createMCPClient(ctx, server)
		if err != nil {
			// 如果创建MCP客户端失败，则关闭所有已创建的MCP客户端
			closeMCPClients(clients)
			return fmt.Errorf("create mcp client %s: %w", name, err)
		}

		// 为每个MCP客户端创建一个带60s超时的初始化上下文
		initCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		// 构造MCP初始化请求体
		initReq := mcp.InitializeRequest{}
		// 设置协议版本为MCP官方推荐的最新版本
		initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		// 填写客户端实现信息，可选项
		initReq.Params.ClientInfo = mcp.Implementation{
			Name:    "deepAgent",
			Version: "0.1.0",
		}
		// 可按需声明你的mcp能力，这里用默认空能力
		initReq.Params.Capabilities = mcp.ClientCapabilities{}

		log.Printf("initializing mcp server: name=%s", name)
		_, err = mcpClient.Initialize(initCtx, initReq)
		cancel()

		if err != nil {
			_ = mcpClient.Close()
			closeMCPClients(clients)
			return fmt.Errorf("initialize mcp client %s: %w", name, err)
		}

		clients[name] = mcpClient
		log.Printf("initialized mcp server: name=%s transport=%s", name, transport)
	}

	MCPServer = clients
	return nil
}

func getMCPTransport(server conf.MCPServerConfig) string {
	if server.URL != "" {
		return transportSSE
	}
	return transportStdio
}

func createMCPClient(ctx context.Context, server conf.MCPServerConfig) (client.MCPClient, error) {
	if server.URL != "" {
		return createSSEMCPClient(ctx, server)
	}
	return createStdioMCPClient(server)
}

func createStdioMCPClient(server conf.MCPServerConfig) (client.MCPClient, error) {
	if server.Command == "" {
		return nil, fmt.Errorf("stdio mcp command is empty")
	}

	var env []string
	for k, v := range server.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return client.NewStdioMCPClient(
		server.Command,
		env,
		server.Args...,
	)
}

func createSSEMCPClient(ctx context.Context, server conf.MCPServerConfig) (client.MCPClient, error) {
	options := []mcptransport.ClientOption{}

	if len(server.Headers) > 0 {
		headers := parseMCPHeaders(server.Headers)
		if len(headers) > 0 {
			options = append(options, client.WithHeaders(headers))
		}
	}

	sseClient, err := client.NewSSEMCPClient(server.URL, options...)
	if err != nil {
		return nil, err
	}
	if err := sseClient.Start(ctx); err != nil {
		_ = sseClient.Close()
		return nil, err
	}

	return sseClient, nil
}

func parseMCPHeaders(values []string) map[string]string {
	headers := make(map[string]string)
	for _, header := range values {
		key, value, ok := strings.Cut(header, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		headers[key] = value
	}
	return headers
}

type MCPToolInfo struct {
	Server      string
	Name        string
	Description string
}

func ListMCPTools(ctx context.Context) ([]MCPToolInfo, error) {
	if len(MCPServer) == 0 {
		return nil, nil
	}

	toolInfos := []MCPToolInfo{}
	for name, cli := range MCPServer {
		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			return nil, fmt.Errorf("list mcp tools %s: %w", name, err)
		}

		for _, t := range tools {
			info, err := t.Info(ctx)
			if err != nil {
				return nil, fmt.Errorf("get mcp tool info %s: %w", name, err)
			}
			toolInfos = append(toolInfos, MCPToolInfo{
				Server:      name,
				Name:        info.Name,
				Description: info.Desc,
			})
		}
	}

	return toolInfos, nil
}

// LogMCPTools 打印当前已连接 MCP server 暴露出的工具列表。
// 设置 DEEPAGENT_DEBUG_MCP=true 时 main.go 会调用它，方便确认 list_tools 链路是否正常。
func LogMCPTools(ctx context.Context) error {
	tools, err := ListMCPTools(ctx)
	if err != nil {
		return err
	}
	if len(tools) == 0 {
		log.Printf("mcp tools: no tool loaded")
		return nil
	}

	log.Printf("mcp tools: total=%d", len(tools))
	for _, t := range tools {
		log.Printf("mcp tool: server=%s name=%s desc=%s", t.Server, t.Name, t.Description)
	}
	return nil
}

func closeMCPClients(clients map[string]client.MCPClient) {
	for _, c := range clients {
		_ = c.Close()
	}
}
