package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"deepAgent/conf"
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
		var env []string
		for k, v := range server.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		mcpClient, err := client.NewStdioMCPClient(
			server.Command,
			env,
			server.Args...,
		)
		if err != nil {
			// 如果创建MCP客户端失败，则关闭所有已创建的MCP客户端
			closeMCPClients(clients)
			return fmt.Errorf("create mcp client %s: %w", name, err)
		}

		// 为每个MCP客户端创建一个带60s超时的初始化上下文
		initCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

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

		_, err = mcpClient.Initialize(initCtx, initReq)

		if err != nil {
			_ = mcpClient.Close()
			closeMCPClients(clients)
			return fmt.Errorf("initialize mcp client %s: %w", name, err)
		}

		clients[name] = mcpClient
	}

	MCPServer = clients
	return nil
}

func closeMCPClients(clients map[string]client.MCPClient) {
	for _, c := range clients {
		_ = c.Close()
	}
}
