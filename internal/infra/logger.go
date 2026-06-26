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

	if strings.TrimSpace(msg.Content) == "" {
		return nil
	}

	data.Role = "assistant"
	return cb.push(ctx, "message_chunk", data)
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
	if cb.Out == nil {
		return ctx
	}
	if inputStr, ok := input.(string); ok {
		cb.Out <- "\n==================\n"
		cb.Out <- inputStr
		cb.Out <- "\n==================\n"
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

	go func() {
		defer output.Close()
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
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
				_ = cb.pushMsg(ctx, msgID, v)
			case *ecmodel.CallbackOutput:
				_ = cb.pushMsg(ctx, msgID, v.Message)
			case []*schema.Message:
				for _, msg := range v {
					_ = cb.pushMsg(ctx, msgID, msg)
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
