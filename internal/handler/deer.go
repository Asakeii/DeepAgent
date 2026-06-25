package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

// ChatStreamEino 是 deepAgent 的 SSE 接口。
// 第一次请求会跑完整个图，遇到 Human Feedback 时返回 interrupt 事件；
// 第二次请求带同一个 thread_id 和 interrupt_feedback，就能继续恢复执行。
func ChatStreamEino(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	sse := infra.NewSSEWriter(w)

	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = sse.WriteEvent("error", &model.ChatResp{
			Role:    "assistant",
			Content: "invalid request body: " + err.Error(),
		})
		return
	}
	if req.InterruptFeedback != "" {
		req.InterruptFeedback = NormalizeInterruptFeedback(req.InterruptFeedback)
	}

	state := &model.State{
		Messages:                      req.Messages,
		Goto:                          consts.Coordinator,
		Locale:                        "zh-CN",
		MaxPlanIterations:             conf.App.Setting.MaxPlanIterations,
		MaxStepNum:                    conf.App.Setting.MaxStepNum,
		AutoAcceptedPlan:              req.AutoAcceptedPlan,
		EnableBackgroundInvestigation: req.EnableBackgroundInvestigation,
	}
	if req.MaxPlanIterations > 0 {
		state.MaxPlanIterations = req.MaxPlanIterations
	}
	if req.MaxStepNum > 0 {
		state.MaxStepNum = req.MaxStepNum
	}
	if state.Messages == nil {
		state.Messages = []*schema.Message{}
	}

	runnable, err := agent.Builder(ctx, func(ctx context.Context) *model.State {
		return state
	})
	if err != nil {
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "build graph failed: " + err.Error()})
		return
	}

	opts := []compose.Option{}
	if req.ThreadID != "" {
		opts = append(opts, compose.WithCheckPointID(req.ThreadID))
	}
	if req.InterruptFeedback != "" {
		opts = append(opts, compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
			st := s.(*model.State)
			st.InterruptFeedback = req.InterruptFeedback
			if req.InterruptFeedback == consts.EditPlan && len(req.Messages) > 0 {
				st.Messages = append(st.Messages, req.Messages...)
			}
			return nil
		}))
	}
	opts = append(opts, compose.WithCallbacks(&infra.LoggerCallback{
		ID:  req.ThreadID,
		SSE: sse,
	}))

	out, err := runnable.Stream(ctx, consts.Coordinator, opts...)
	if err != nil {
		if _, ok := compose.ExtractInterruptInfo(err); ok {
			_ = sse.WriteEvent("interrupt", &model.ChatResp{
				ThreadID:     req.ThreadID,
				ID:           "human_feedback",
				Role:         "assistant",
				Content:      "检查计划",
				FinishReason: "interrupt",
				Options: []map[string]any{
					{"text": "编辑计划", "value": "edit_plan"},
					{"text": "开始执行", "value": "accepted"},
				},
			})
			return
		}
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "run graph failed: " + err.Error()})
		return
	}
	defer out.Close()

	for {
		msg, recvErr := out.Recv()
		if recvErr != nil {
			if recvErr != io.EOF {
				_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "stream failed: " + recvErr.Error()})
			}
			return
		}
		if strings.TrimSpace(msg) == "" {
			continue
		}
		_ = sse.WriteEvent("message", &model.ChatResp{
			Role:    "assistant",
			Content: msg,
		})
	}
}

// NormalizeInterruptFeedback 统一中断反馈值，方便前端和命令行复用。
func NormalizeInterruptFeedback(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
