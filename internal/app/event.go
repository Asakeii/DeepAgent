package app

import (
	"sync"

	"deepAgent/internal/model"
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
