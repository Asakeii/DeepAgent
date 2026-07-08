package toolruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/store"
)

type Risk string

const (
	RiskSafe      Risk = "safe"
	RiskWrite     Risk = "write"
	RiskExternal  Risk = "external"
	RiskDangerous Risk = "dangerous"
)

type Policy struct {
	DefaultTimeout time.Duration
	Risks          map[string]Risk
	AllowDangerous bool
}

func DefaultPolicy() Policy {
	return Policy{
		DefaultTimeout: 20 * time.Second,
		Risks: map[string]Risk{
			"record_checkin":    RiskWrite,
			"analyze_food":      RiskWrite,
			"create_reminder":   RiskWrite,
			"schedule_reminder": RiskWrite,
			"cancel_reminder":   RiskWrite,
			"delete_reminder":   RiskWrite,
			"list_reminders":    RiskSafe,
			"query_checkin":     RiskSafe,
			"get_summary":       RiskSafe,
			"web_search":        RiskExternal,
			"web_fetch":         RiskExternal,
			"execute_python":    RiskExternal,
		},
	}
}

func WrapTools(db *sql.DB, tools []einotool.BaseTool, policy Policy) []einotool.BaseTool {
	out := make([]einotool.BaseTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, WrapTool(db, t, policy))
	}
	return out
}

func WrapTool(db *sql.DB, t einotool.BaseTool, policy Policy) einotool.BaseTool {
	if t == nil {
		return t
	}
	i, isInvokable := t.(einotool.InvokableTool)
	s, isStreamable := t.(einotool.StreamableTool)
	switch {
	case isInvokable && isStreamable:
		return &combinedTool{baseTool: baseTool{inner: t, db: db, policy: policy}, invokable: i, streamable: s}
	case isInvokable:
		return &invokableTool{baseTool: baseTool{inner: t, db: db, policy: policy}, invokable: i}
	case isStreamable:
		return &streamableTool{baseTool: baseTool{inner: t, db: db, policy: policy}, streamable: s}
	default:
		return t
	}
}

type baseTool struct {
	inner  einotool.BaseTool
	db     *sql.DB
	policy Policy
}

func (t *baseTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

func (t *baseTool) toolName(ctx context.Context) string {
	info, err := t.inner.Info(ctx)
	if err != nil || info == nil || info.Name == "" {
		return "unknown_tool"
	}
	return info.Name
}

func (t *baseTool) risk(name string) Risk {
	if t.policy.Risks != nil {
		if risk, ok := t.policy.Risks[name]; ok {
			return risk
		}
	}
	return RiskSafe
}

func (t *baseTool) timeout() time.Duration {
	if t.policy.DefaultTimeout <= 0 {
		return 20 * time.Second
	}
	return t.policy.DefaultTimeout
}

func (t *baseTool) before(ctx context.Context, name string, args string) (context.Context, int64, time.Time, Risk, error) {
	risk := t.risk(name)
	if risk == RiskDangerous && !t.policy.AllowDangerous {
		return ctx, 0, time.Now(), risk, fmt.Errorf("tool %s is dangerous and requires approval", name)
	}
	meta := AuditContextFrom(ctx)
	payload := json.RawMessage(args)
	if !json.Valid(payload) {
		payload, _ = json.Marshal(map[string]string{"raw": args})
	}
	id, err := store.StartToolAudit(ctx, t.db, store.ToolAuditRecord{
		RunID:     meta.RunID,
		ThreadID:  meta.ThreadID,
		UserID:    meta.UserID,
		ToolName:  name,
		Risk:      string(risk),
		Arguments: payload,
	})
	if err != nil {
		id = 0
	}
	child, _ := context.WithTimeout(ctx, t.timeout())
	return child, id, time.Now(), risk, nil
}

func (t *baseTool) after(ctx context.Context, id int64, started time.Time, status string, result string, err error) {
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	_ = store.CompleteToolAudit(context.Background(), t.db, id, status, result, errText, time.Since(started).Milliseconds())
}

type invokableTool struct {
	baseTool
	invokable einotool.InvokableTool
}

func (t *invokableTool) InvokableRun(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
	name := t.toolName(ctx)
	child, auditID, started, _, policyErr := t.before(ctx, name, args)
	if policyErr != nil {
		t.after(ctx, auditID, started, store.ToolStatusBlocked, "", policyErr)
		return toolErrorText(policyErr), nil
	}
	result, err := t.invokable.InvokableRun(child, args, opts...)
	status := store.ToolStatusSucceeded
	if err != nil {
		status = store.ToolStatusFailed
	}
	t.after(ctx, auditID, started, status, result, err)
	if err != nil {
		return toolErrorText(err), nil
	}
	return result, nil
}

type streamableTool struct {
	baseTool
	streamable einotool.StreamableTool
}

func (t *streamableTool) StreamableRun(ctx context.Context, args string, opts ...einotool.Option) (*schema.StreamReader[string], error) {
	name := t.toolName(ctx)
	child, auditID, started, _, policyErr := t.before(ctx, name, args)
	if policyErr != nil {
		t.after(ctx, auditID, started, store.ToolStatusBlocked, "", policyErr)
		return schema.StreamReaderFromArray([]string{toolErrorText(policyErr)}), nil
	}
	sr, err := t.streamable.StreamableRun(child, args, opts...)
	status := store.ToolStatusSucceeded
	if err != nil {
		status = store.ToolStatusFailed
	}
	t.after(ctx, auditID, started, status, "", err)
	if err != nil {
		return schema.StreamReaderFromArray([]string{toolErrorText(err)}), nil
	}
	return sr, nil
}

type combinedTool struct {
	baseTool
	invokable  einotool.InvokableTool
	streamable einotool.StreamableTool
}

func (t *combinedTool) InvokableRun(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
	return (&invokableTool{baseTool: t.baseTool, invokable: t.invokable}).InvokableRun(ctx, args, opts...)
}

func (t *combinedTool) StreamableRun(ctx context.Context, args string, opts ...einotool.Option) (*schema.StreamReader[string], error) {
	return (&streamableTool{baseTool: t.baseTool, streamable: t.streamable}).StreamableRun(ctx, args, opts...)
}

func toolErrorText(err error) string {
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "tool failed"
	}
	return "工具调用失败：" + msg
}
