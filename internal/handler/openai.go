package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/app"
	"deepAgent/internal/infra"
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
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = strings.TrimSpace(r.Header.Get("X-DeepAgent-Run"))
	}
	if runID == "" {
		runID = app.NewRunID()
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

	if err := store.CreateRun(r.Context(), infra.DB, store.RunRecord{
		ID:       runID,
		UserID:   userID,
		ThreadID: threadID,
		Mode:     "openai",
	}); err != nil {
		log.Printf("[openai] create run=%s thread=%s: %v", runID, threadID, err)
	}
	resp, err := app.NewCheckinService().RunTurn(r.Context(), app.CheckinTurnRequest{
		RunID:    runID,
		UserID:   userID,
		ThreadID: threadID,
		Messages: msgs,
	})
	if err != nil {
		_ = store.CompleteRun(r.Context(), infra.DB, runID, store.RunStatusFailed, err.Error())
		writeOpenAICompletions(w, runID, err.Error())
		return
	}
	_ = store.CompleteRun(r.Context(), infra.DB, runID, store.RunStatusSucceeded, "")
	content := ""
	if resp.Response != nil {
		content = resp.Response.Content
	}
	writeOpenAICompletions(w, runID, content)
}

// ---- OpenAI-compatible types ----

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	RunID    string        `json:"run_id,omitempty"`
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

func writeOpenAICompletions(w http.ResponseWriter, id, content string) {
	if content == "" {
		content = "processed"
	}
	if id == "" {
		id = app.NewRunID()
	}
	resp := chatCompletionResponse{
		ID:     id,
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
