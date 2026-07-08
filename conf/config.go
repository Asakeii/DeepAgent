package conf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultSSEHeartbeatSeconds = 15

// Config 对应 conf/deep-agent.yaml 的整体结构。
// 配置分为 MCP 服务、模型服务和运行参数三部分。
type Config struct {
	MCP      MCPConfig      `yaml:"mcp"`
	Model    ModelConfig    `yaml:"model"`
	Setting  SettingConfig  `yaml:"setting"`
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
}

// DatabaseConfig 描述 MySQL 连接配置。
// DSN 形如 "user:pass@tcp(host:3306)/db?parseTime=true"。
// 本阶段 DSN 允许为空，连接校验留待 InitDB（Task 2）执行。
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `yaml:"servers"`
}

// MCPServerConfig 描述一个 MCP server 的启动方式。
// command/args/env 对应 stdio MCP；url/headers 对应 SSE MCP。
type MCPServerConfig struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env,omitempty"`
	URL     string            `yaml:"url,omitempty"`
	Headers []string          `yaml:"headers,omitempty"`
}

// ModelConfig 描述 OpenAI-compatible chat completion 服务配置。
// VisionModel 为识图 agent 专用模型，未配置时为 nil，后续识图阶段回退主讲模型。
type ModelConfig struct {
	DefaultModel string       `yaml:"default_model"`
	APIKey       string       `yaml:"api_key"`
	BaseURL      string       `yaml:"base_url"`
	VisionModel  *ModelConfig `yaml:"vision_model,omitempty"`
}

// SettingConfig 是 Agent 工作流运行时参数。
type SettingConfig struct {
	MaxPlanIterations             int  `yaml:"max_plan_iterations"`
	MaxStepNum                    int  `yaml:"max_step_num"`
	EnableBackgroundInvestigation bool `yaml:"enable_background_investigation"`
	RunTimeoutSeconds             int  `yaml:"run_timeout_seconds"`
}

type ServerConfig struct {
	AllowedOrigins      []string `yaml:"allowed_origins"`
	MaxBodyBytes        int64    `yaml:"max_body_bytes"`
	APIKeys             []string `yaml:"api_keys"`
	RateLimitPerMinute  int      `yaml:"rate_limit_per_minute"`
	SSEHeartbeatSeconds int      `yaml:"sse_heartbeat_seconds"`
}

var App *Config // 全局配置变量，保存加载后的配置

// Load 负责加载并解析 conf/deep-agent.yaml 配置文件
func Load(ctx context.Context) (*Config, error) {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err) // 获取工作目录失败
	}

	// 拼接配置文件路径
	path := filepath.Join(wd, "conf", "deep-agent.yaml")
	// 读取配置文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err) // 读取配置文件失败
	}

	var cfg Config
	// 解析 YAML 配置内容到 cfg
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err) // 解析配置失败
	}

	// 如果没有配置最大步骤数，则设置默认值为3
	if cfg.Setting.MaxStepNum <= 0 {
		cfg.Setting.MaxStepNum = 3
	}

	// 如果没有配置最大计划迭代次数，则设置默认值为1
	if cfg.Setting.MaxPlanIterations <= 0 {
		cfg.Setting.MaxPlanIterations = 1
	}
	if cfg.Server.MaxBodyBytes <= 0 {
		cfg.Server.MaxBodyBytes = 1 << 20
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		cfg.Server.AllowedOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	if cfg.Server.SSEHeartbeatSeconds == 0 {
		cfg.Server.SSEHeartbeatSeconds = DefaultSSEHeartbeatSeconds
	}

	App = &cfg // 保存到全局变量
	return App, nil
}

func (c ServerConfig) SSEHeartbeatInterval() time.Duration {
	if c.SSEHeartbeatSeconds < 0 {
		return 0
	}
	return time.Duration(c.SSEHeartbeatSeconds) * time.Second
}

func (c SettingConfig) RunTimeout() time.Duration {
	if c.RunTimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(c.RunTimeoutSeconds) * time.Second
}
