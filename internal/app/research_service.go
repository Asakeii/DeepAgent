package app

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/compose"

	"deepAgent/conf"
	"deepAgent/internal/agent"
	"deepAgent/internal/consts"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
)

type ResearchService struct{}

type ResearchRunResult struct {
	RouteToCheckin bool
	Final          string
}

func NewResearchService() *ResearchService {
	return &ResearchService{}
}

func (s *ResearchService) Run(ctx context.Context, req model.ChatRequest, writer EventWriter) (ResearchRunResult, error) {
	req.InterruptFeedback = NormalizeInterruptFeedback(req.InterruptFeedback)
	routeWriter := newRouteDetectingWriter(writer)
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
		return ResearchRunResult{}, err
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
	opts = append(opts, compose.WithCallbacks(&infra.LoggerCallback{ID: req.ThreadID, Events: routeWriter, Final: finalCh}))

	out, err := runnable.Stream(ctx, consts.Coordinator, opts...)
	if err != nil {
		if _, ok := compose.ExtractInterruptInfo(err); ok {
			writeInterrupt(writer, req.ThreadID)
			return ResearchRunResult{}, nil
		}
		_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "run graph failed: " + err.Error()})
		return ResearchRunResult{}, err
	}
	defer out.Close()

	lastGraphMsg := ""
	for {
		msg, recvErr := out.Recv()
		if recvErr != nil {
			if _, ok := compose.ExtractInterruptInfo(recvErr); ok {
				writeInterrupt(writer, req.ThreadID)
				return ResearchRunResult{}, nil
			}
			if recvErr != io.EOF {
				_ = writer.WriteEvent("error", &model.ChatResp{Role: "assistant", Content: "stream failed: " + recvErr.Error()})
				return ResearchRunResult{}, recvErr
			}
			result := ResearchRunResult{}
			if routeWriter.RouteToCheckin() {
				result.RouteToCheckin = true
				return result, nil
			}
			final := waitFinalMessage(finalCh)
			if final == "" {
				final = lastGraphMsg
			}
			result.Final = final
			return result, nil
		}
		if strings.TrimSpace(msg) != "" {
			lastGraphMsg = msg
		}
	}
}

type routeDetectingWriter struct {
	inner          EventWriter
	mu             sync.Mutex
	routeToCheckin bool
}

func newRouteDetectingWriter(inner EventWriter) *routeDetectingWriter {
	return &routeDetectingWriter{inner: inner}
}

func (w *routeDetectingWriter) WriteEvent(event string, payload any) error {
	if resp, ok := payload.(*model.ChatResp); ok {
		for _, call := range resp.ToolCalls {
			if call.Name == "hand_to_checkin" {
				w.mu.Lock()
				w.routeToCheckin = true
				w.mu.Unlock()
				break
			}
		}
		for _, call := range resp.ToolCallChunks {
			if call.Name == "hand_to_checkin" {
				w.mu.Lock()
				w.routeToCheckin = true
				w.mu.Unlock()
				break
			}
		}
	}
	return w.inner.WriteEvent(event, payload)
}

func (w *routeDetectingWriter) RouteToCheckin() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.routeToCheckin
}
