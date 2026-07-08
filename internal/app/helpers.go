package app

import (
	"context"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
	"deepAgent/internal/store"
)

func firstUserMessage(req model.ChatRequest) string {
	for _, msg := range req.Messages {
		if msg != nil && msg.Role == schema.User && strings.TrimSpace(msg.Content) != "" {
			return strings.TrimSpace(msg.Content)
		}
	}
	return ""
}

func writeInterrupt(writer EventWriter, threadID string) {
	_ = writer.WriteEvent("interrupt", &model.ChatResp{
		ThreadID:     threadID,
		ID:           "human_feedback",
		Role:         "assistant",
		Content:      "检查计划",
		FinishReason: "interrupt",
		Options: []map[string]any{
			{"text": "编辑计划", "value": "edit_plan"},
			{"text": "开始执行", "value": "accepted"},
		},
	})
}

func reminderResp(event scheduler.ReminderEvent) *model.ReminderResp {
	return &model.ReminderResp{
		ID:        event.ID,
		ThreadID:  event.ThreadID,
		Message:   event.Message,
		FireAt:    event.FireAt,
		Cron:      event.Cron,
		Recurring: event.Recurring,
		Status:    event.Status,
	}
}

// NormalizeInterruptFeedback normalizes human feedback values.
func NormalizeInterruptFeedback(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func waitFinalMessage(ch <-chan string) string {
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		return ""
	}
}

func persistResearchMessages(ctx context.Context, threadID string, msgs []*schema.Message, final string) {
	if threadID == "" || final == "" {
		return
	}
	for _, msg := range msgs {
		if msg == nil || msg.Role != schema.User || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.User), msg.Content)
	}
	_ = infra.AppendMessageForCheckin(ctx, threadID, string(schema.Assistant), final)
}

func persistExplicitMemories(ctx context.Context, userID, threadID string, msgs []*schema.Message) {
	for _, msg := range msgs {
		if msg == nil || msg.Role != schema.User {
			continue
		}
		content := explicitMemoryContent(msg.Content)
		if content == "" {
			continue
		}
		_, _ = store.CreateMemory(ctx, infra.DB, store.MemoryRecord{
			UserID:     userID,
			ThreadID:   threadID,
			Kind:       store.MemoryKindPreference,
			Content:    content,
			Importance: 5,
			Source:     "explicit_user_message",
		})
	}
}

func explicitMemoryContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	triggers := []string{"请记住", "记住", "以后请记得", "以后记得", "我的目标", "我的偏好", "我喜欢", "我不喜欢"}
	for _, trigger := range triggers {
		if strings.Contains(content, trigger) {
			return content
		}
	}
	return ""
}
