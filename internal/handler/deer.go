package handler

import (
	"encoding/json"
	"net/http"

	"deepAgent/conf"
	"deepAgent/internal/app"
	"deepAgent/internal/auth"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// ChatStreamEino 是 deepAgent 的 SSE 接口。
func ChatStreamEino(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	sse := infra.NewSSEWriter(w)
	if conf.App != nil {
		stopHeartbeat := sse.StartHeartbeat(ctx, conf.App.Server.SSEHeartbeatInterval())
		defer stopHeartbeat()
	}

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "invalid request body: " + err.Error()})
		return
	}
	if _, authenticated := auth.PrincipalFromContext(r.Context()); authenticated || req.UserID == "" {
		req.UserID = requestUserID(r)
	}
	app.NewChatService().RunStream(ctx, req, sse)
}
