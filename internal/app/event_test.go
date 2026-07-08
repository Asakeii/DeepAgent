package app

import (
	"testing"

	"deepAgent/internal/model"
)

func TestCaptureWriterFinalContentPrefersTerminalEvents(t *testing.T) {
	writer := NewCaptureWriter()

	_ = writer.WriteEvent("message_chunk", &model.ChatResp{Content: "partial"})
	_ = writer.WriteEvent("final_message", &model.ChatResp{Content: "final"})

	if got := writer.FinalContent(); got != "final" {
		t.Fatalf("FinalContent() = %q, want %q", got, "final")
	}
	if got := len(writer.Events()); got != 2 {
		t.Fatalf("Events length = %d, want 2", got)
	}
}

func TestCaptureWriterFallsBackToError(t *testing.T) {
	writer := NewCaptureWriter()

	_ = writer.WriteEvent("error", &model.ChatResp{Content: "boom"})

	if got := writer.FinalContent(); got != "boom" {
		t.Fatalf("FinalContent() = %q, want %q", got, "boom")
	}
}
