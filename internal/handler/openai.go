package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/app"
	"deepAgent/internal/store"
)

// OpenAICompatible 提供 /v1/chat/completions 端点。
// WeClaw 微信桥接直走 checkin agent（打卡/记录/查询/提醒/闲聊），不经过 Coordinator 研究图。
func OpenAICompatible(w http.ResponseWriter, r *http.Request) {
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOpenAIError(w, "invalid request body: "+err.Error())
		return
	}
	if len(req.Messages) == 0 {
		writeOpenAIError(w, "messages is required")
		return
	}

	threadID := req.ThreadID
	if threadID == "" {
		threadID = "weclaw-default"
	}
	userID := requestUserID(r)
	if userID == store.AnonymousUserID {
		userID = "openai:" + threadID
	}

	msgs := make([]*schema.Message, len(req.Messages))
	for i, m := range req.Messages {
		role := schema.User
		switch strings.ToLower(m.Role) {
		case "assistant":
			role = schema.Assistant
		case "system":
			role = schema.System
		}
		msgs[i] = &schema.Message{Role: role, Content: m.Content}
	}

	resp, err := app.NewCheckinService().RunTurn(r.Context(), app.CheckinTurnRequest{
		UserID:   userID,
		ThreadID: threadID,
		Messages: msgs,
	})
	if err != nil {
		writeOpenAICompletions(w, err.Error())
		return
	}
	content := ""
	if resp.Response != nil {
		content = resp.Response.Content
	}
	writeOpenAICompletions(w, content)
}

// ---- OpenAI-compatible types ----

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	ThreadID string        `json:"thread_id,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
}

type chatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

func writeOpenAICompletions(w http.ResponseWriter, content string) {
	if content == "" {
		content = "processed"
	}
	resp := chatCompletionResponse{
		ID:     "chatcmpl-deepagent",
		Object: "chat.completion",
		Model:  "deepagent",
		Choices: []chatCompletionChoice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func writeOpenAIError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
