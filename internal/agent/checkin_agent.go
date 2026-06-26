package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/tool"
)

// NewCheckinAgent 构造一个独立的 ReAct checkin agent（不入总图，直接 agent.Generate 调用）。
// agent 使用 infra.ChatModel + 打卡工具集；MessageModifier 在调用前注入 system prompt。
// 历史消息由调用方（runCheckin）传入 []*schema.Message，这里 MessageModifier 只补 system prompt。
func NewCheckinAgent(ctx context.Context) (*react.Agent, error) {
	tools, err := tool.CheckinTools(ctx, infra.DB, infra.VisionModel)
	if err != nil {
		return nil, fmt.Errorf("build checkin tools: %w", err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:               40,
		ToolCallingModel:      infra.ChatModel,
		ToolsConfig:           compose.ToolsNodeConfig{Tools: tools},
		MessageModifier:       checkinMessageModifier,
		StreamToolCallChecker: toolCallChecker, // 复用 researcher/coder 里已有的 checker
	})
	if err != nil {
		return nil, fmt.Errorf("new checkin agent: %w", err)
	}
	return agent, nil
}

// checkinMessageModifier 在每次模型调用前注入 system prompt。
func checkinMessageModifier(ctx context.Context, msgs []*schema.Message) []*schema.Message {
	sysPrompt, err := infra.GetPromptTemplate(ctx, "checkin_coach")
	if err != nil {
		// prompt 加载失败不阻断，用兜底 system
		sysPrompt = "你是自律打卡教练，通过工具帮用户记录和查询打卡。"
	}
	out := make([]*schema.Message, 0, len(msgs)+1)
	out = append(out, schema.SystemMessage(sysPrompt))
	out = append(out, msgs...)
	return out
}
