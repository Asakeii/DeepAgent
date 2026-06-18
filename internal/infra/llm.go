package infra

import (
	"context"
	"fmt"

	modelopenai "github.com/cloudwego/eino-ext/components/model/openai"
	openaiacl "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/eino-contrib/jsonschema"

	"deepAgent/conf"
	"deepAgent/internal/model"
)

var (
	// ChatModel 通用对话模型，供 Coordinator、Researcher、Reporter 等节点使用
	ChatModel *modelopenai.ChatModel
	// PlanModel 结构化输出模型，强制 Planner 按 model.Plan JSON Schema 返回
	PlanModel *modelopenai.ChatModel
)

// InitModel 根据配置初始化 ChatModel 与 PlanModel。
func InitModel(ctx context.Context) error {
	if conf.App == nil {
		return fmt.Errorf("config is not loaded")
	}

	cfg := conf.App.Model
	if cfg.DefaultModel == "" {
		return fmt.Errorf("model.default_model is empty")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("model.api_key is empty")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("model.base_url is empty")
	}

	// 普通 Chat Completion，无 response_format 约束
	chatModel, err := modelopenai.NewChatModel(ctx, &modelopenai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.DefaultModel,
	})
	if err != nil {
		return fmt.Errorf("init chat model: %w", err)
	}
	ChatModel = chatModel

	// 从 model.Plan 反射 JSON Schema，供 Planner 输出结构化计划
	planSchema := jsonschema.Reflect(&model.Plan{})
	planModel, err := modelopenai.NewChatModel(ctx, &modelopenai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.DefaultModel,
		ResponseFormat: &openaiacl.ChatCompletionResponseFormat{
			Type: openaiacl.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openaiacl.ChatCompletionResponseFormatJSONSchema{
				Name:       "plan",
				Strict:     false,
				JSONSchema: planSchema,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("init plan model: %w", err)
	}
	PlanModel = planModel

	return nil
}
