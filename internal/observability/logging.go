package observability

import (
	"io"
	"log/slog"
	"os"
)

const (
	LogKeyRunID    = "run_id"
	LogKeyThreadID = "thread_id"
	LogKeyUserID   = "user_id"
)

func ConfigureJSONLogger(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{})))
}

func RunLogger(runID, threadID, userID string) *slog.Logger {
	return slog.Default().With(
		slog.String(LogKeyRunID, runID),
		slog.String(LogKeyThreadID, threadID),
		slog.String(LogKeyUserID, userID),
	)
}
