package infra

import (
	"context"
	"fmt"
	"strings"

	modelopenai "github.com/cloudwego/eino-ext/components/model/openai"
	openaiacl "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/eino-contrib/jsonschema"

	"deepAgent/conf"
	"deepAgent/internal/model"
)

var (
	// ChatModel 通用对话模型，供 Coordinator、Researcher、Reporter 等节点使用。
	ChatModel *modelopenai.ChatModel
	// PlanModel 结构化输出模型，强制 Planner 按 model.Plan JSON Schema 返回。
	PlanModel *modelopenai.ChatModel
	// VisionModel 多模态视觉模型，供识图 agent 使用。未单独配置时回退主 ChatModel。
	VisionModel *modelopenai.ChatModel

	defaultModels *ModelBundle
	modelProfiles map[string]*ModelBundle
)

type modelProfileContextKey struct{}

type ModelBundle struct {
	Profile string
	Chat    *modelopenai.ChatModel
	Plan    *modelopenai.ChatModel
	Vision  *modelopenai.ChatModel
}

// InitModel 根据配置初始化默认模型和命名模型 profile。
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

	base := conf.ModelEndpointConfig{
		DefaultModel: cfg.DefaultModel,
		APIKey:       cfg.APIKey,
		BaseURL:      cfg.BaseURL,
	}
	bundle, err := newModelBundle(ctx, "", base, cfg.VisionModel)
	if err != nil {
		return err
	}
	defaultModels = bundle
	modelProfiles = map[string]*ModelBundle{}
	for name, profile := range cfg.Profiles {
		name = NormalizeModelProfile(name)
		if name == "" {
			continue
		}
		endpoint := conf.ModelEndpointConfig{
			DefaultModel: strings.TrimSpace(profile.DefaultModel),
			APIKey:       firstNonEmpty(profile.APIKey, cfg.APIKey),
			BaseURL:      firstNonEmpty(profile.BaseURL, cfg.BaseURL),
		}
		if endpoint.DefaultModel == "" {
			return fmt.Errorf("model.profiles.%s.default_model is empty", name)
		}
		profileBundle, err := newModelBundle(ctx, name, endpoint, nil)
		if err != nil {
			return err
		}
		modelProfiles[name] = profileBundle
	}

	ChatModel = bundle.Chat
	PlanModel = bundle.Plan
	VisionModel = bundle.Vision
	return nil
}

func newModelBundle(ctx context.Context, profile string, endpoint conf.ModelEndpointConfig, vision *conf.ModelEndpointConfig) (*ModelBundle, error) {
	chatModel, err := modelopenai.NewChatModel(ctx, &modelopenai.ChatModelConfig{
		BaseURL:         endpoint.BaseURL,
		APIKey:          endpoint.APIKey,
		Model:           endpoint.DefaultModel,
		ReasoningEffort: modelopenai.ReasoningEffortLevelMedium,
	})
	if err != nil {
		return nil, fmt.Errorf("init chat model %q: %w", profile, err)
	}

	planSchema := jsonschema.Reflect(&model.Plan{})
	planModel, err := modelopenai.NewChatModel(ctx, &modelopenai.ChatModelConfig{
		BaseURL: endpoint.BaseURL,
		APIKey:  endpoint.APIKey,
		Model:   endpoint.DefaultModel,
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
		return nil, fmt.Errorf("init plan model %q: %w", profile, err)
	}

	visionModel := chatModel
	if vision != nil && strings.TrimSpace(vision.DefaultModel) != "" {
		vm, err := modelopenai.NewChatModel(ctx, &modelopenai.ChatModelConfig{
			BaseURL: firstNonEmpty(vision.BaseURL, endpoint.BaseURL),
			APIKey:  firstNonEmpty(vision.APIKey, endpoint.APIKey),
			Model:   strings.TrimSpace(vision.DefaultModel),
		})
		if err != nil {
			return nil, fmt.Errorf("init vision model %q: %w", profile, err)
		}
		visionModel = vm
	}

	return &ModelBundle{Profile: profile, Chat: chatModel, Plan: planModel, Vision: visionModel}, nil
}

func WithModelProfile(ctx context.Context, profile string) context.Context {
	profile = NormalizeModelProfile(profile)
	if profile == "" {
		return ctx
	}
	return context.WithValue(ctx, modelProfileContextKey{}, profile)
}

func NormalizeModelProfile(profile string) string {
	return strings.ToLower(strings.TrimSpace(profile))
}

func HasModelProfile(profile string) bool {
	profile = NormalizeModelProfile(profile)
	if profile == "" {
		return true
	}
	_, ok := modelProfiles[profile]
	return ok
}

func ActiveModelProfile(ctx context.Context) string {
	profile, _ := ctx.Value(modelProfileContextKey{}).(string)
	if HasModelProfile(profile) {
		return NormalizeModelProfile(profile)
	}
	return ""
}

func ChatModelFor(ctx context.Context) *modelopenai.ChatModel {
	return modelBundleFor(ctx).Chat
}

func PlanModelFor(ctx context.Context) *modelopenai.ChatModel {
	return modelBundleFor(ctx).Plan
}

func VisionModelFor(ctx context.Context) *modelopenai.ChatModel {
	return modelBundleFor(ctx).Vision
}

func modelBundleFor(ctx context.Context) *ModelBundle {
	profile, _ := ctx.Value(modelProfileContextKey{}).(string)
	if bundle, ok := modelProfiles[NormalizeModelProfile(profile)]; ok {
		return bundle
	}
	if defaultModels != nil {
		return defaultModels
	}
	return &ModelBundle{Chat: ChatModel, Plan: PlanModel, Vision: VisionModel}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
