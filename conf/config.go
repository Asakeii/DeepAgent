package conf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config describes the YAML configuration shape for deepAgent.
// TODO: Implement YAML loading from conf/deep-agent.yaml.
type Config struct {
	MCP     MCPConfig     `yaml:"mcp"`
	Model   ModelConfig   `yaml:"model"`
	Setting SettingConfig `yaml:"setting"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env,omitempty"`
}

type ModelConfig struct {
	DefaultModel string `yaml:"default_model"`
	APIKey       string `yaml:"api_key"`
	BaseURL      string `yaml:"base_url"`
}

type SettingConfig struct {
	MaxPlanIterations int `yaml:"max_plan_iterations"`
	MaxStepNum        int `yaml:"max_step_num"`
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

	App = &cfg // 保存到全局变量
	return App, nil
}
