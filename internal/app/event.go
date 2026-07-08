package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sync"

	"deepAgent/internal/model"
	"deepAgent/internal/observability"
	"deepAgent/internal/security"
	"deepAgent/internal/store"
)

// EventWriter is the small transport boundary application services need.
// SSE, tests, WeChat, and future adapters can implement it without exposing HTTP details.
type EventWriter interface {
	WriteEvent(event string, payload any) error
}

// CaptureWriter records app events for non-streaming adapters such as WeChat
// and OpenAI-compatible responses.
type CaptureWriter struct {
	mu      sync.Mutex
	events  []CapturedEvent
	content string
	errText string
	chunks  string
}

type CapturedEvent struct {
	Name    string
	Payload any
}

func NewCaptureWriter() *CaptureWriter {
	return &CaptureWriter{}
}

func (w *CaptureWriter) WriteEvent(event string, payload any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.events = append(w.events, CapturedEvent{Name: event, Payload: payload})
	if resp, ok := payload.(*model.ChatResp); ok {
		switch event {
		case "final_message", "message":
			if resp.Content != "" {
				w.content = resp.Content
			}
		case "message_chunk":
			w.chunks += resp.Content
		case "error":
			w.errText = resp.Content
		}
	}
	return nil
}

func (w *CaptureWriter) FinalContent() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.content != "" {
		return w.content
	}
	if w.errText != "" {
		return w.errText
	}
	return w.chunks
}

func (w *CaptureWriter) Events() []CapturedEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]CapturedEvent, len(w.events))
	copy(out, w.events)
	return out
}

type RunEventWriter struct {
	inner    EventWriter
	db       *sql.DB
	runID    string
	threadID string
	userID   string

	mu      sync.Mutex
	failed  bool
	errText string
}

func NewRunEventWriter(db *sql.DB, runID, threadID, userID string, inner EventWriter) *RunEventWriter {
	return &RunEventWriter{
		inner:    inner,
		db:       db,
		runID:    runID,
		threadID: threadID,
		userID:   userID,
	}
}

func (w *RunEventWriter) WriteEvent(event string, payload any) error {
	payload = w.enrichPayload(event, payload)
	w.record(event, payload)
	if w.inner == nil {
		return nil
	}
	return w.inner.WriteEvent(event, payload)
}

func (w *RunEventWriter) WritePassthroughEvent(event string, payload any) error {
	payload = w.enrichPayload(event, payload)
	if w.inner == nil {
		return nil
	}
	return w.inner.WriteEvent(event, payload)
}

func (w *RunEventWriter) RunID() string {
	return w.runID
}

func (w *RunEventWriter) Failed() (bool, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.failed, w.errText
}

func (w *RunEventWriter) enrichPayload(event string, payload any) any {
	resp, ok := payload.(*model.ChatResp)
	if !ok || resp == nil {
		return payload
	}
	if resp.RunID == "" {
		resp.RunID = w.runID
	}
	if resp.ThreadID == "" {
		resp.ThreadID = w.threadID
	}
	if event == "error" {
		w.mu.Lock()
		w.failed = true
		w.errText = resp.Content
		w.mu.Unlock()
	}
	return resp
}

func (w *RunEventWriter) record(event string, payload any) {
	if w.db == nil || w.runID == "" || w.threadID == "" {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		w.logger().ErrorContext(context.Background(), "marshal run event failed", slog.String("event", event), slog.Any("error", err))
		return
	}
	b = security.RedactJSON(b)
	agentName := ""
	if resp, ok := payload.(*model.ChatResp); ok && resp != nil {
		agentName = resp.Agent
	}
	if err := store.AppendRunEvent(context.Background(), w.db, store.RunEventRecord{
		RunID:     w.runID,
		ThreadID:  w.threadID,
		UserID:    w.userID,
		EventName: event,
		Agent:     agentName,
		Payload:   b,
	}); err != nil {
		w.logger().ErrorContext(context.Background(), "append run event failed", slog.String("event", event), slog.Any("error", err))
	}
}

func (w *RunEventWriter) logger() *slog.Logger {
	return observability.RunLogger(w.runID, w.threadID, w.userID)
}

func newRunID() string {
	return NewRunID()
}

func NewRunID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "run"
	}
	return "run_" + hex.EncodeToString(b[:])
}
