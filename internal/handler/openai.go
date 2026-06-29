package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// OpenAICompatible 提供 /v1/chat/completions 端点。
// WeClaw 的 HTTP mode 通过此端点将微信消息转发给 deepAgent。
func OpenAICompatible(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	// 转换消息格式
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

	runnable := agent.GetAgent()
	opts := []compose.Option{
		compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
			st := s.(*model.State)
			st.Messages = msgs
			st.ThreadID = threadID
			st.Locale = "zh-CN"
			st.AutoAcceptedPlan = true
			return nil
		}),
	}
	if threadID != "" {
		opts = append(opts, compose.WithCheckPointID(threadID))
	}

	var sb strings.Builder
	outChan := make(chan string)
	done := make(chan struct{})
	go func() {
		for s := range outChan {
			sb.WriteString(s)
		}
		close(done)
	}()

	_, err := runnable.Stream(ctx, consts.Coordinator,
		append(opts, compose.WithCallbacks(&infra.LoggerCallback{
			ID:  threadID,
			Out: outChan,
		}))...,
	)
	close(outChan)
	<-done

	if err != nil {
		writeOpenAICompletions(w, sb.String())
		return
	}

	// Coordinator 的 checkin 路由信号
	if _, ok := agent.CheckinThreads.LoadAndDelete(threadID); ok {
		resp, cerr := agent.RunCheckin(context.Background(), msgs, threadID)
		if cerr != nil {
			writeOpenAICompletions(w, cerr.Error())
			return
		}
		writeOpenAICompletions(w, resp.Content)
		return
	}

	writeOpenAICompletions(w, strings.TrimSpace(sb.String()))
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
