package app

import (
	"context"

	"deepAgent/internal/model"
)

type ResearchRunner interface {
	Run(ctx context.Context, req model.ChatRequest, writer EventWriter) (ResearchRunResult, error)
}

type CheckinRunner interface {
	RunTurn(ctx context.Context, req CheckinTurnRequest) (CheckinTurnResult, error)
	AnalyzeImage(ctx context.Context, req model.ChatRequest) (string, error)
	EmitResult(writer EventWriter, threadID string, result CheckinTurnResult)
}

type ReminderStreamer interface {
	AttachStream(ctx context.Context, threadID string, writer EventWriter) func()
}
