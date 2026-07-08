package infra

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEWriterWriteComment(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	if err := writer.WriteComment("heartbeat"); err != nil {
		t.Fatalf("write comment: %v", err)
	}

	if got, want := rec.Body.String(), ": heartbeat\n\n"; got != want {
		t.Fatalf("comment frame=%q want %q", got, want)
	}
	if !rec.Flushed {
		t.Fatal("comment frame was not flushed")
	}
}

func TestSSEWriterStartHeartbeat(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	stop := writer.StartHeartbeat(ctx, time.Millisecond)
	defer stop()

	deadline := time.After(200 * time.Millisecond)
	for {
		if strings.Contains(sseBody(writer, rec), ": heartbeat\n\n") {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("heartbeat not written, body=%q", sseBody(writer, rec))
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func sseBody(writer *SSEWriter, rec *httptest.ResponseRecorder) string {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return rec.Body.String()
}
