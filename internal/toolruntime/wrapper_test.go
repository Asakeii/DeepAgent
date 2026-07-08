package toolruntime

import (
	"context"
	"fmt"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type fakeInvokableTool struct {
	name string
	err  error
}

func (t fakeInvokableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name}, nil
}

func (t fakeInvokableTool) InvokableRun(ctx context.Context, args string, opts ...einotool.Option) (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return "ok:" + args, nil
}

func TestWrapToolConvertsErrorsToToolText(t *testing.T) {
	wrapped := WrapTool(nil, fakeInvokableTool{name: "record_checkin", err: fmt.Errorf("db down")}, DefaultPolicy())
	invokable, ok := wrapped.(einotool.InvokableTool)
	if !ok {
		t.Fatalf("wrapped type = %T, want InvokableTool", wrapped)
	}
	got, err := invokable.InvokableRun(context.Background(), `{"x":1}`)
	if err != nil {
		t.Fatalf("InvokableRun err = %v, want nil", err)
	}
	if got == "" || got == "ok" {
		t.Fatalf("unexpected result %q", got)
	}
}

func TestWrapToolBlocksDangerousToolByPolicy(t *testing.T) {
	policy := DefaultPolicy()
	policy.Risks["danger"] = RiskDangerous
	wrapped := WrapTool(nil, fakeInvokableTool{name: "danger"}, policy)
	got, err := wrapped.(einotool.InvokableTool).InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun err = %v, want nil", err)
	}
	if got == "" {
		t.Fatal("expected policy error text")
	}
}
