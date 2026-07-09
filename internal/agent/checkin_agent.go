package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/scheduler"
	"deepAgent/internal/tool"
	"deepAgent/internal/toolruntime"
)

// NewCheckinAgent 构造一个独立的 ReAct checkin agent（不入总图，直接 agent.Generate 调用）。
// agent 使用 infra.ChatModel + 打卡工具集；MessageModifier 在调用前注入 system prompt。
// threadID 通过 context.WithValue(tool.CtxKeyThreadID, tid) 在调用 agent.Generate 时传入。
func NewCheckinAgent(ctx context.Context, threadID string) (*react.Agent, error) {
	useScheduler := infra.RDB != nil
	tools, err := tool.CheckinToolsWithOptions(ctx, infra.DB, infra.VisionModel, threadID, !useScheduler)
	if err != nil {
		return nil, fmt.Errorf("build checkin tools: %w", err)
	}

	// Redis-based reminder tools (schedule_reminder, cancel_reminder, list_reminders)
	if useScheduler {
		rt, err := scheduler.Tools(ctx, infra.RDB, threadID)
		if err != nil {
			log.Printf("[checkin] scheduler tools: %v", err)
		} else {
			tools = append(tools, rt...)
		}
	}
	tools = toolruntime.WrapTools(infra.DB, tools, toolruntime.DefaultPolicy())

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

// RunCheckin 被 handler 层调用（Coordinator 标记 RouteToCheckin=true 后）。
// 加载跨会话历史、持久化消息、调 checkin agent。
func RunCheckin(ctx context.Context, msgs []*schema.Message, threadID string) (*schema.Message, error) {
	if threadID == "" {
		threadID = "console-default"
	}

	agent, err := NewCheckinAgent(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("new checkin agent: %w", err)
	}

	// 加载历史 + 记录用户消息
	history, _ := infra.RecentMessagesForCheckin(ctx, threadID, 20)
	prompt := append(history, msgs...)
	for _, m := range msgs {
		if m == nil || m.Role != schema.User || strings.TrimSpace(m.Content) == "" {
			continue
		}
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(m.Role), m.Content)
	}

	audit := toolruntime.AuditContextFrom(ctx)
	resp, err := agent.Generate(ctx, prompt, einoagent.WithComposeOptions(compose.WithCallbacks(&infra.LoggerCallback{
		ID:     threadID,
		RunID:  audit.RunID,
		UserID: audit.UserID,
	})))
	if err != nil {
		return nil, err
	}

	// 记录助手回复
	_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.Assistant), resp.Content)
	return resp, nil
}

// AnalyzeFoodImage 直接调用 VisionModel 分析食物图片，不走 ReAct agent。
func AnalyzeFoodImage(ctx context.Context, imageB64, text, threadID string) (string, error) {
	return tool.AnalyzeFoodDirect(ctx, imageB64, text, threadID, infra.DB, infra.VisionModel)
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
