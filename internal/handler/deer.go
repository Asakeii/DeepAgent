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

	// 图片粘贴：直接调用 VisionModel 分析，返回结果
	if req.ImageBase64 != "" && len(req.Messages) > 0 {
		txt := req.Messages[len(req.Messages)-1].Content
		resp, cerr := agent.AnalyzeFoodImage(context.Background(), req.ImageBase64, txt, req.ThreadID)
		if cerr != nil {
			_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "分析失败: " + cerr.Error()})
		} else {
			_ = sse.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp})
		}
		return
	}

	// per-request Builder + genFunc（对齐 deer-go）：genFunc 在编译时创建 State，
	// sub-graph 节点通过 GenLocalState 继承正确的 Messages。
	genFunc := func(ctx context.Context) *model.State {
		return &model.State{
			Messages:                      req.Messages,
			Goto:                          consts.Coordinator,
			Locale:                        "zh-CN",
			MaxPlanIterations:             req.MaxPlanIterations,
			MaxStepNum:                    req.MaxStepNum,
			AutoAcceptedPlan:              req.AutoAcceptedPlan,
			EnableBackgroundInvestigation: req.EnableBackgroundInvestigation,
			ThreadID:                      req.ThreadID,
		}
	}
	runnable, err := agent.Builder(ctx, genFunc)
	if err != nil {
		_ = sse.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "build graph failed: " + err.Error()})
		return
	}

	opts := []compose.Option{}
	if req.ThreadID != "" {
		opts = append(opts, compose.WithCheckPointID(req.ThreadID))
	}
	// 中断恢复：回填 InterruptFeedback
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
			// EOF: checkin routing signal from Coordinator
			if recvErr == io.EOF {
				if _, ok := agent.CheckinThreads.LoadAndDelete(req.ThreadID); ok {
					resp, cerr := agent.RunCheckin(context.Background(), req.Messages, req.ThreadID)
					if cerr == nil {
						_ = sse.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp.Content})
					}
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
