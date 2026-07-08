package app

import (
	"context"
	"log"
	"sync"
	"time"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

const runCancellationPollInterval = time.Second

func withRunCancellation(ctx context.Context, runID string) (context.Context, func()) {
	if runID == "" || infra.DB == nil {
		return ctx, func() {}
	}
	runCtx, cancel := context.WithCancel(ctx)
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(runCancellationPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				cancelled, err := store.IsRunCancelled(context.Background(), infra.DB, runID)
				if err != nil {
					log.Printf("[run] cancellation check run=%s: %v", runID, err)
					continue
				}
				if cancelled {
					cancel()
					return
				}
			}
		}
	}()

	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
			cancel()
			<-doneCh
		})
	}
	return runCtx, stop
}

func isRunCancelled(runID string) bool {
	if runID == "" || infra.DB == nil {
		return false
	}
	cancelled, err := store.IsRunCancelled(context.Background(), infra.DB, runID)
	if err != nil {
		log.Printf("[run] cancellation status run=%s: %v", runID, err)
		return false
	}
	return cancelled
}

func writeRunCancelled(writer EventWriter, runID, threadID string) {
	payload := &model.ChatResp{
		RunID:        runID,
		ThreadID:     threadID,
		Role:         "assistant",
		Content:      "运行已取消",
		FinishReason: store.RunStatusCancelled,
	}
	if passthrough, ok := writer.(interface {
		WritePassthroughEvent(string, any) error
	}); ok {
		_ = passthrough.WritePassthroughEvent("run_cancelled", payload)
		return
	}
	_ = writer.WriteEvent("run_cancelled", payload)
}
