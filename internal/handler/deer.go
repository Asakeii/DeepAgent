package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/compose"

	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// ChatStreamEino 是 deepAgent 的 SSE 接口。
func ChatStreamEino(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	sse := infra.NewSSEWriter(w)

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "invalid request body: " + err.Error()})
		return
	}
	if req.InterruptFeedback != "" {
		req.InterruptFeedback = NormalizeInterruptFeedback(req.InterruptFeedback)
	}

	var routedToCheckin bool
	runnable := agent.GetAgent()

	opts := []compose.Option{
		compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
			st := s.(*model.State)
			if req.Messages != nil {
				st.Messages = req.Messages
			}
			if req.ThreadID != "" {
				st.ThreadID = req.ThreadID
			}
			st.Locale = "zh-CN"
			if req.MaxPlanIterations > 0 {
				st.MaxPlanIterations = req.MaxPlanIterations
			}
			if req.MaxStepNum > 0 {
				st.MaxStepNum = req.MaxStepNum
			}
			st.AutoAcceptedPlan = req.AutoAcceptedPlan
			st.EnableBackgroundInvestigation = req.EnableBackgroundInvestigation
			if req.InterruptFeedback != "" {
				st.InterruptFeedback = req.InterruptFeedback
				if req.InterruptFeedback == consts.EditPlan && len(req.Messages) > 0 {
					st.Messages = append(st.Messages, req.Messages...)
				}
			}
			// 捕获 Coordinator 的路由标记
			routedToCheckin = st.RouteToCheckin
			return nil
		}),
	}
	if req.ThreadID != "" {
		opts = append(opts, compose.WithCheckPointID(req.ThreadID))
	}
	opts = append(opts, compose.WithCallbacks(&infra.LoggerCallback{ID: req.ThreadID, SSE: sse}))

	out, err := runnable.Stream(ctx, consts.Coordinator, opts...)
	if err != nil {
		if _, ok := compose.ExtractInterruptInfo(err); ok {
			_ = sse.WriteEvent("interrupt", &model.ChatResp{ThreadID: req.ThreadID, ID: "human_feedback", Role: "assistant", Content: "检查计划", FinishReason: "interrupt", Options: []map[string]any{{"text": "编辑计划", "value": "edit_plan"}, {"text": "开始执行", "value": "accepted"}}})
			return
		}
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "run graph failed: " + err.Error()})
		return
	}
	defer out.Close()

	for {
		msg, recvErr := out.Recv()
		if recvErr != nil {
			if _, ok := compose.ExtractInterruptInfo(recvErr); ok {
				_ = sse.WriteEvent("interrupt", &model.ChatResp{ThreadID: req.ThreadID, ID: "human_feedback", Role: "assistant", Content: "检查计划", FinishReason: "interrupt", Options: []map[string]any{{"text": "编辑计划", "value": "edit_plan"}, {"text": "开始执行", "value": "accepted"}}})
				return
			}
			if recvErr != io.EOF {
				_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "stream failed: " + recvErr.Error()})
			}
			// EOF: 检查 Coordinator 的打卡路由标记
			if recvErr == io.EOF && routedToCheckin && req.Messages != nil {
				resp, cerr := agent.RunCheckin(context.Background(), req.Messages, req.ThreadID)
				if cerr == nil {
					_ = sse.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp.Content})
				}
			}
			return
		}
		if strings.TrimSpace(msg) == "" {
			continue
		}
		_ = sse.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: msg})
	}
}

// NormalizeInterruptFeedback 统一中断反馈值。
func NormalizeInterruptFeedback(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
