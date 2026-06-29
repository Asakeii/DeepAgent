package model

import "github.com/cloudwego/eino/schema"

// ChatRequest 对齐 deer-go 的服务请求结构。
// thread_id 用于 checkpoint 恢复；interrupt_feedback 用于 Human Feedback 回填。
type ChatRequest struct {
	Messages                      []*schema.Message `json:"messages,omitempty"`
	Debug                         bool              `json:"debug,omitempty"`
	ThreadID                      string            `json:"thread_id,omitempty"`
	MaxPlanIterations             int               `json:"max_plan_iterations,omitempty"`
	MaxStepNum                    int               `json:"max_step_num,omitempty"`
	AutoAcceptedPlan              bool              `json:"auto_accepted_plan,omitempty"`
	InterruptFeedback             string            `json:"interrupt_feedback,omitempty"`
	MCPSettings                   map[string]any    `json:"mcp_settings,omitempty"`
	EnableBackgroundInvestigation bool              `json:"enable_background_investigation,omitempty"`
	ImageBase64                   string            `json:"image_base64,omitempty"`
}

type ToolResp struct {
	ID   string         `json:"id,omitempty"`
	Type string         `json:"type,omitempty"`
	Name string         `json:"name,omitempty"`
	Args map[string]any `json:"args,omitempty"`
}

type ToolChunkResp struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

// ChatResp 是 SSE 事件里返回给前端的统一消息结构。
type ChatResp struct {
	ThreadID       string           `json:"thread_id,omitempty"`
	Agent          string           `json:"agent,omitempty"`
	ID             string           `json:"id,omitempty"`
	Role           string           `json:"role,omitempty"`
	Content        string           `json:"content,omitempty"`
	FinishReason   string           `json:"finish_reason,omitempty"`
	Options        []map[string]any `json:"options,omitempty"`
	ToolCallID     string           `json:"tool_call_id,omitempty"`
	ToolCalls      []ToolResp       `json:"tool_calls,omitempty"`
	ToolCallChunks []ToolChunkResp  `json:"tool_call_chunks,omitempty"`
	MessageChunks  any              `json:"message_chunks,omitempty"`
}
