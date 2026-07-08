package app

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"deepAgent/internal/agent"
)

type defaultCheckinAgentRunner struct{}

func (defaultCheckinAgentRunner) RunCheckin(ctx context.Context, msgs []*schema.Message, threadID string) (*schema.Message, error) {
	return agent.RunCheckin(ctx, msgs, threadID)
}

func (defaultCheckinAgentRunner) AnalyzeFoodImage(ctx context.Context, imageB64, text, threadID string) (string, error) {
	return agent.AnalyzeFoodImage(ctx, imageB64, text, threadID)
}
