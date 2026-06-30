package infra

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	ecmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/model"
)

// SSEWriter 是基于 net/http 的简单 SSE 写入器。
// callback 可能在 goroutine 中并发写事件，所以这里用 mutex 保护每次完整写入。
type SSEWriter struct {
	w  http.ResponseWriter
	mu sync.Mutex
}

func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	return &SSEWriter{w: w}
}

func (sw *SSEWriter) WriteEvent(event string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	if _, err = sw.w.Write([]byte("event: " + event + "\n")); err != nil {
		return err
	}
	if _, err = sw.w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err = sw.w.Write(b); err != nil {
		return err
	}
	if _, err = sw.w.Write([]byte("\n\n")); err != nil {
		return err
	}
	if f, ok := sw.w.(http.Flusher); ok {
		// Flush 在客户端断开连接后可能 panic（bufio.Writer 变 nil）。
		// recover 兜底，让 callback goroutine 不崩。
		func() {
			defer func() { recover() }()
			f.Flush()
		}()
	}

	return nil
}

// LoggerCallback 把 Eino 运行过程中的模型输出、工具调用和工具结果转成 SSE 事件。
// 它对齐 deer-go 的 logger.go，但适配 deepAgent 当前的 net/http 服务层。
type LoggerCallback struct {
	callbacks.HandlerBuilder

	ID  string
	SSE *SSEWriter
	Out chan string
	// Final receives completed assistant messages for persistence.
	// It is intentionally best-effort so callback goroutines never block the graph.
	Final chan string
}

func (cb *LoggerCallback) push(ctx context.Context, event string, data *model.ChatResp) error {
	if cb.SSE != nil {
		if err := cb.SSE.WriteEvent(event, data); err != nil {
			return err
		}
	}
	if cb.Out != nil && data.Content != "" {
		cb.Out <- data.Content
	}
	return nil
}

func isHiddenAssistantContent(content string) bool {
	switch strings.ToLower(strings.TrimSpace(content)) {
	case "", "end", "processed":
		return true
	default:
		return false
	}
}

func (cb *LoggerCallback) pushMsg(ctx context.Context, msgID string, msg *schema.Message) error {
	if msg == nil {
		return nil
	}

	agentName := ""
	_ = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		agentName = state.Goto
		return nil
	})

	finishReason := ""
	if msg.ResponseMeta != nil {
		finishReason = msg.ResponseMeta.FinishReason
	}

	data := &model.ChatResp{
		ThreadID:      cb.ID,
		Agent:         agentName,
		ID:            msgID,
		Role:          string(msg.Role),
		Content:       msg.Content,
		FinishReason:  finishReason,
		MessageChunks: msg.Content,
	}

	if msg.Role == schema.Tool {
		data.Role = "tool"
		data.ToolCallID = msg.ToolCallID
		return cb.push(ctx, "tool_call_result", data)
	}

	if len(msg.ToolCalls) > 0 {
		return cb.pushToolCall(ctx, data, msg.ToolCalls)
	}

	if isHiddenAssistantContent(msg.Content) {
		return nil
	}

	data.Role = "assistant"
	return cb.push(ctx, "message_chunk", data)
}

func (cb *LoggerCallback) agentName(ctx context.Context) string {
	agentName := ""
	_ = compose.ProcessState[*model.State](ctx, func(ctx context.Context, state *model.State) error {
		agentName = state.Goto
		return nil
	})
	return agentName
}

func (cb *LoggerCallback) pushPlan(ctx context.Context, content string) {
	content = stripJSONFence(content)
	if strings.TrimSpace(content) == "" {
		return
	}
	var plan model.Plan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return
	}
	_ = cb.push(ctx, "plan", &model.ChatResp{
		ThreadID: cb.ID,
		Agent:    "planner",
		Role:     "assistant",
		Plan:     &plan,
	})
}

func (cb *LoggerCallback) pushFinal(ctx context.Context, agentName, content string) {
	content = strings.TrimSpace(content)
	if isHiddenAssistantContent(content) {
		return
	}
	_ = cb.push(ctx, "final_message", &model.ChatResp{
		ThreadID: cb.ID,
		Agent:    agentName,
		Role:     "assistant",
		Content:  content,
	})
	if cb.Final != nil {
		select {
		case cb.Final <- content:
		default:
		}
	}
}

func stripJSONFence(content string) string {
	s := strings.TrimSpace(content)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.IndexByte(s, '\n'); nl > 0 {
		s = strings.TrimSpace(s[nl+1:])
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	return s
}

func (cb *LoggerCallback) pushToolCall(ctx context.Context, data *model.ChatResp, toolCalls []schema.ToolCall) error {
	toolCall := toolCalls[0]
	name := toolCall.Function.Name
	event := "tool_call_chunks"

	if name != "" {
		event = "tool_calls"
		if strings.HasSuffix(name, "search") {
			name = "web_search"
		}
		data.ToolCalls = []model.ToolResp{
			{
				ID:   toolCall.ID,
				Type: "tool_call",
				Name: name,
				Args: map[string]any{},
			},
		}
	}

	data.ToolCallChunks = []model.ToolChunkResp{
		{
			ID:   toolCall.ID,
			Type: "tool_call_chunk",
			Name: name,
			Args: toolCall.Function.Arguments,
		},
	}

	return cb.push(ctx, event, data)
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	// 不再往 Out 写裸文本节点标记，改为推送 SSE agent 事件供前端展示状态卡片
	if inputStr, ok := input.(string); ok && inputStr != "" && cb.SSE != nil {
		_ = cb.SSE.WriteEvent("agent", &model.ChatResp{
			ThreadID: cb.ID,
			Agent:    inputStr,
		})
	}
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if cb.SSE != nil {
		_ = cb.SSE.WriteEvent("error", &model.ChatResp{
			ThreadID: cb.ID,
			Role:     "assistant",
			Content:  err.Error(),
		})
	}
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(
	ctx context.Context,
	info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput],
) context.Context {
	msgID := newMsgID()
	agentName := cb.agentName(ctx)

	go func() {
		defer output.Close()
		var content strings.Builder
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				switch agentName {
				case "planner":
					cb.pushPlan(ctx, content.String())
				case "reporter", "coordinator":
					cb.pushFinal(ctx, agentName, content.String())
				}
				return
			}
			if err != nil {
				_ = cb.push(ctx, "error", &model.ChatResp{
					ThreadID: cb.ID,
					Role:     "assistant",
					Content:  err.Error(),
				})
				return
			}

			switch v := frame.(type) {
			case *schema.Message:
				if v != nil && v.Role == schema.Assistant && len(v.ToolCalls) == 0 {
					content.WriteString(v.Content)
				}
				if agentName != "planner" {
					_ = cb.pushMsg(ctx, msgID, v)
				}
			case *ecmodel.CallbackOutput:
				if v.Message != nil && v.Message.Role == schema.Assistant && len(v.Message.ToolCalls) == 0 {
					content.WriteString(v.Message.Content)
				}
				if agentName != "planner" {
					_ = cb.pushMsg(ctx, msgID, v.Message)
				}
			case []*schema.Message:
				for _, msg := range v {
					if msg != nil && msg.Role == schema.Assistant && len(msg.ToolCalls) == 0 {
						content.WriteString(msg.Content)
					}
					if agentName != "planner" {
						_ = cb.pushMsg(ctx, msgID, msg)
					}
				}
			}
		}
	}()

	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(
	ctx context.Context,
	info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput],
) context.Context {
	defer input.Close()
	return ctx
}
