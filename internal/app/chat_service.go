package app

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/scheduler"
)

// EventWriter is the small transport boundary ChatService needs.
// SSE, tests, and future adapters can implement it without exposing HTTP details.
type EventWriter interface {
	WriteEvent(event string, payload any) error
}

// ChatService orchestrates one chat turn while keeping transport handlers thin.
type ChatService struct{}

func NewChatService() *ChatService {
	return &ChatService{}
}

// RunStream preserves the existing /chat/stream event contract while moving
// agent orchestration out of the HTTP handler.
func (s *ChatService) RunStream(ctx context.Context, req model.ChatRequest, writer EventWriter) {
	if req.InterruptFeedback != "" {
		req.InterruptFeedback = NormalizeInterruptFeedback(req.InterruptFeedback)
	}

	if req.ThreadID != "" && infra.RDB != nil {
		notifCh := scheduler.DefaultRegistry.Register(req.ThreadID)
		defer scheduler.DefaultRegistry.Unregister(req.ThreadID)
		scheduler.DrainPending(ctx, infra.RDB, req.ThreadID, notifCh)
		go func() {
			for event := range notifCh {
				_ = writer.WriteEvent("reminder", &model.ChatResp{
					ThreadID: req.ThreadID,
					Role:     "assistant",
					Content:  "提醒：" + event.Message,
					Reminder: reminderResp(event),
				})
			}
		}()
	}

	if req.ImageBase64 != "" && len(req.Messages) > 0 {
		s.runImageAnalysis(ctx, req, writer)
		return
	}

	s.runGraph(ctx, req, writer)
}

func (s *ChatService) runImageAnalysis(ctx context.Context, req model.ChatRequest, writer EventWriter) {
	txt := req.Messages[len(req.Messages)-1].Content
	resp, err := agent.AnalyzeFoodImage(ctx, req.ImageBase64, txt, req.ThreadID)
	if err != nil {
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "分析失败: " + err.Error()})
		return
	}
	_ = writer.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp})
}

func (s *ChatService) runGraph(ctx context.Context, req model.ChatRequest, writer EventWriter) {
	maxPlanIterations := req.MaxPlanIterations
	if maxPlanIterations <= 0 {
		maxPlanIterations = conf.App.Setting.MaxPlanIterations
	}
	if maxPlanIterations <= 0 {
		maxPlanIterations = 1
	}
	maxStepNum := req.MaxStepNum
	if maxStepNum <= 0 {
		maxStepNum = conf.App.Setting.MaxStepNum
	}
	if maxStepNum <= 0 {
		maxStepNum = 3
	}
	enableBackgroundInvestigation := conf.App.Setting.EnableBackgroundInvestigation
	if req.EnableBackgroundInvestigation != nil {
		enableBackgroundInvestigation = *req.EnableBackgroundInvestigation
	}

	genFunc := func(ctx context.Context) *model.State {
		return &model.State{
			Messages:                      req.Messages,
			Goto:                          consts.Coordinator,
			Locale:                        "zh-CN",
			MaxPlanIterations:             maxPlanIterations,
			MaxStepNum:                    maxStepNum,
			AutoAcceptedPlan:              req.AutoAcceptedPlan,
			EnableBackgroundInvestigation: enableBackgroundInvestigation,
			ThreadID:                      req.ThreadID,
		}
	}
	runnable, err := agent.Builder(ctx, genFunc)
	if err != nil {
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "build graph failed: " + err.Error()})
		return
	}

	opts := []compose.Option{}
	if req.ThreadID != "" {
		opts = append(opts, compose.WithCheckPointID(req.ThreadID))
	}
	if req.InterruptFeedback != "" {
		opts = append(opts, compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, state any) error {
			st := state.(*model.State)
			st.InterruptFeedback = req.InterruptFeedback
			if req.InterruptFeedback == consts.EditPlan && len(req.Messages) > 0 {
				st.Messages = append(st.Messages, req.Messages...)
			}
			return nil
		}))
	}

	finalCh := make(chan string, 1)
	opts = append(opts, compose.WithCallbacks(&infra.LoggerCallback{ID: req.ThreadID, Events: writer, Final: finalCh}))

	out, err := runnable.Stream(ctx, consts.Coordinator, opts...)
	if err != nil {
		if _, ok := compose.ExtractInterruptInfo(err); ok {
			writeInterrupt(writer, req.ThreadID)
			return
		}
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "run graph failed: " + err.Error()})
		return
	}
	defer out.Close()

	lastGraphMsg := ""
	for {
		msg, recvErr := out.Recv()
		if recvErr != nil {
			if _, ok := compose.ExtractInterruptInfo(recvErr); ok {
				writeInterrupt(writer, req.ThreadID)
				return
			}
			if recvErr != io.EOF {
				_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "stream failed: " + recvErr.Error()})
			}
			if recvErr == io.EOF {
				s.finishGraph(ctx, req, writer, finalCh, lastGraphMsg)
			}
			return
		}
		if strings.TrimSpace(msg) != "" {
			lastGraphMsg = msg
		}
	}
}

func (s *ChatService) finishGraph(
	ctx context.Context,
	req model.ChatRequest,
	writer EventWriter,
	finalCh <-chan string,
	lastGraphMsg string,
) {
	if _, ok := agent.CheckinThreads.LoadAndDelete(req.ThreadID); ok {
		s.runCheckin(ctx, req, writer)
		return
	}

	final := waitFinalMessage(finalCh)
	if final == "" {
		final = lastGraphMsg
	}
	persistResearchMessages(ctx, req.ThreadID, req.Messages, final)
}

func (s *ChatService) runCheckin(ctx context.Context, req model.ChatRequest, writer EventWriter) {
	var reminderEvents []scheduler.ReminderEvent
	var reminderMu sync.Mutex
	checkinCtx := scheduler.WithEventSink(ctx, func(event scheduler.ReminderEvent) {
		reminderMu.Lock()
		defer reminderMu.Unlock()
		reminderEvents = append(reminderEvents, event)
	})

	resp, err := agent.RunCheckin(checkinCtx, req.Messages, req.ThreadID)
	if err != nil {
		return
	}

	reminderMu.Lock()
	events := append([]scheduler.ReminderEvent(nil), reminderEvents...)
	reminderMu.Unlock()
	for _, event := range events {
		_ = writer.WriteEvent("reminder_scheduled", &model.ChatResp{
			ThreadID: req.ThreadID,
			Role:     "assistant",
			Content:  event.Message,
			Reminder: reminderResp(event),
		})
	}
	_ = writer.WriteEvent("message", &model.ChatResp{Role: "assistant", Content: resp.Content})
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
