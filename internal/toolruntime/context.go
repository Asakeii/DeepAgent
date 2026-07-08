package toolruntime

import "context"

type auditContextKey struct{}

type AuditContext struct {
	RunID    string
	ThreadID string
	UserID   string
}

func WithAuditContext(ctx context.Context, meta AuditContext) context.Context {
	return context.WithValue(ctx, auditContextKey{}, meta)
}

func AuditContextFrom(ctx context.Context) AuditContext {
	if meta, ok := ctx.Value(auditContextKey{}).(AuditContext); ok {
		return meta
	}
	return AuditContext{}
}
