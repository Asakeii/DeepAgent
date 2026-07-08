package model

import "github.com/cloudwego/eino/schema"

// ChatRequest 对齐 deer-go 的服务请求结构。
// thread_id 用于 checkpoint 恢复；interrupt_feedback 用于 Human Feedback 回填。
type ChatRequest struct {
	Messages                      []*schema.Message `json:"messages,omitempty"`
	Debug                         bool              `json:"debug,omitempty"`
	UserID                        string            `json:"user_id,omitempty"`
	RunID                         string            `json:"run_id,omitempty"`
	ThreadID                      string            `json:"thread_id,omitempty"`
	Locale                        string            `json:"locale,omitempty"`
	MaxPlanIterations             int               `json:"max_plan_iterations,omitempty"`
	MaxStepNum                    int               `json:"max_step_num,omitempty"`
	AutoAcceptedPlan              bool              `json:"auto_accepted_plan,omitempty"`
	InterruptFeedback             string            `json:"interrupt_feedback,omitempty"`
	MCPSettings                   map[string]any    `json:"mcp_settings,omitempty"`
	EnableBackgroundInvestigation *bool             `json:"enable_background_investigation,omitempty"`
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

type ReminderResp struct {
	ID        string `json:"id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	Message   string `json:"message,omitempty"`
	FireAt    int64  `json:"fire_at,omitempty"`
	Cron      string `json:"cron,omitempty"`
	Recurring bool   `json:"recurring,omitempty"`
	Status    string `json:"status,omitempty"`
}

type CancelReminderRequest struct {
	ThreadID   string `json:"thread_id"`
	ReminderID string `json:"reminder_id"`
}

type ToggleReminderRequest struct {
	ThreadID   string `json:"thread_id"`
	ReminderID string `json:"reminder_id"`
	Active     bool   `json:"active"`
}

type CancelReminderResponse struct {
	Reminder *ReminderResp `json:"reminder"`
}

type ToggleReminderResponse struct {
	Reminder *ReminderResp `json:"reminder"`
}

type ListRemindersResponse struct {
	Reminders []*ReminderResp `json:"reminders"`
}

type RunEventResp struct {
	ID        int64  `json:"id"`
	RunID     string `json:"run_id"`
	ThreadID  string `json:"thread_id"`
	UserID    string `json:"user_id,omitempty"`
	EventName string `json:"event_name"`
	Agent     string `json:"agent,omitempty"`
	Payload   any    `json:"payload"`
}

type ListRunEventsResponse struct {
	Events []*RunEventResp `json:"events"`
}

type CancelRunRequest struct {
	RunID string `json:"run_id"`
}

type CancelRunResponse struct {
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	Cancelled bool   `json:"cancelled"`
}

type ToolAuditResp struct {
	ID         int64  `json:"id"`
	RunID      string `json:"run_id"`
	ThreadID   string `json:"thread_id"`
	UserID     string `json:"user_id,omitempty"`
	ToolName   string `json:"tool_name"`
	Risk       string `json:"risk"`
	Status     string `json:"status"`
	Arguments  any    `json:"arguments,omitempty"`
	Result     string `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type ListToolAuditsResponse struct {
	Audits []*ToolAuditResp `json:"audits"`
}

type RunMetricsResp struct {
	UserID            string  `json:"user_id"`
	WindowHours       int     `json:"window_hours"`
	RunsTotal         int     `json:"runs_total"`
	RunsSucceeded     int     `json:"runs_succeeded"`
	RunsFailed        int     `json:"runs_failed"`
	RunsRunning       int     `json:"runs_running"`
	RunSuccessRate    float64 `json:"run_success_rate"`
	AvgRunLatencyMS   int64   `json:"avg_run_latency_ms"`
	P95RunLatencyMS   int64   `json:"p95_run_latency_ms"`
	ToolsTotal        int     `json:"tools_total"`
	ToolsFailed       int     `json:"tools_failed"`
	ToolsBlocked      int     `json:"tools_blocked"`
	ToolErrorRate     float64 `json:"tool_error_rate"`
	AvgToolDurationMS int64   `json:"avg_tool_duration_ms"`
}

type MemoryResp struct {
	ID         int64  `json:"id"`
	UserID     string `json:"user_id,omitempty"`
	ThreadID   string `json:"thread_id,omitempty"`
	Kind       string `json:"kind"`
	Content    string `json:"content"`
	Importance int    `json:"importance"`
	Source     string `json:"source,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type CreateMemoryRequest struct {
	ThreadID   string `json:"thread_id,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Content    string `json:"content"`
	Importance int    `json:"importance,omitempty"`
	Source     string `json:"source,omitempty"`
}

type CreateMemoryResponse struct {
	Memory *MemoryResp `json:"memory"`
}

type ListMemoriesResponse struct {
	Memories []*MemoryResp `json:"memories"`
}

type ArtifactResp struct {
	ID        int64  `json:"id"`
	UserID    string `json:"user_id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Format    string `json:"format"`
	Content   string `json:"content"`
	Version   int64  `json:"version"`
	Source    string `json:"source,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ListArtifactsResponse struct {
	Artifacts []*ArtifactResp `json:"artifacts"`
}

type CitationResp struct {
	ID         int64  `json:"id"`
	ArtifactID int64  `json:"artifact_id"`
	UserID     string `json:"user_id,omitempty"`
	ThreadID   string `json:"thread_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Quote      string `json:"quote,omitempty"`
	Position   int    `json:"position"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type ListCitationsResponse struct {
	Citations []*CitationResp `json:"citations"`
}

type UserSettingsResp struct {
	UserID                        string `json:"user_id,omitempty"`
	Locale                        string `json:"locale"`
	Timezone                      string `json:"timezone"`
	MaxPlanIterations             *int   `json:"max_plan_iterations,omitempty"`
	MaxStepNum                    *int   `json:"max_step_num,omitempty"`
	EnableBackgroundInvestigation *bool  `json:"enable_background_investigation,omitempty"`
	AutoAcceptPlan                *bool  `json:"auto_accept_plan,omitempty"`
	UpdatedAt                     string `json:"updated_at,omitempty"`
}

type UpdateUserSettingsRequest struct {
	Locale                        *string `json:"locale,omitempty"`
	Timezone                      *string `json:"timezone,omitempty"`
	MaxPlanIterations             *int    `json:"max_plan_iterations,omitempty"`
	MaxStepNum                    *int    `json:"max_step_num,omitempty"`
	EnableBackgroundInvestigation *bool   `json:"enable_background_investigation,omitempty"`
	AutoAcceptPlan                *bool   `json:"auto_accept_plan,omitempty"`
}

type UserSettingsResponse struct {
	Settings *UserSettingsResp `json:"settings"`
}

// ChatResp 是 SSE 事件里返回给前端的统一消息结构。
type ChatResp struct {
	RunID          string           `json:"run_id,omitempty"`
	ThreadID       string           `json:"thread_id,omitempty"`
	Agent          string           `json:"agent,omitempty"`
	ID             string           `json:"id,omitempty"`
	Role           string           `json:"role,omitempty"`
	Content        string           `json:"content,omitempty"`
	Plan           *Plan            `json:"plan,omitempty"`
	FinishReason   string           `json:"finish_reason,omitempty"`
	Options        []map[string]any `json:"options,omitempty"`
	ToolCallID     string           `json:"tool_call_id,omitempty"`
	ToolCalls      []ToolResp       `json:"tool_calls,omitempty"`
	ToolCallChunks []ToolChunkResp  `json:"tool_call_chunks,omitempty"`
	MessageChunks  any              `json:"message_chunks,omitempty"`
	Reminder       *ReminderResp    `json:"reminder,omitempty"`
}
