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

func TestRunEventWriterEnrichesAndTracksFailures(t *testing.T) {
	capture := NewCaptureWriter()
	writer := NewRunEventWriter(nil, "run-1", "thread-1", "user-1", capture)

	_ = writer.WriteEvent("error", &model.ChatResp{Content: "boom"})

	failed, errText := writer.Failed()
	if !failed || errText != "boom" {
		t.Fatalf("Failed() = %v %q, want true boom", failed, errText)
	}
	events := capture.Events()
	if len(events) != 1 {
		t.Fatalf("events length = %d, want 1", len(events))
	}
	resp, ok := events[0].Payload.(*model.ChatResp)
	if !ok {
		t.Fatalf("payload type = %T, want *model.ChatResp", events[0].Payload)
	}
	if resp.RunID != "run-1" || resp.ThreadID != "thread-1" {
		t.Fatalf("enriched ids = %q %q, want run-1 thread-1", resp.RunID, resp.ThreadID)
	}
}

func TestRunEventWriterPassthroughEvent(t *testing.T) {
	capture := NewCaptureWriter()
	writer := NewRunEventWriter(nil, "run-1", "thread-1", "user-1", capture)

	if err := writer.WritePassthroughEvent("run_cancelled", &model.ChatResp{Content: "cancelled"}); err != nil {
		t.Fatal(err)
	}
	events := capture.Events()
	if len(events) != 1 || events[0].Name != "run_cancelled" {
		t.Fatalf("unexpected events: %+v", events)
	}
	resp, ok := events[0].Payload.(*model.ChatResp)
	if !ok {
		t.Fatalf("payload type = %T, want *model.ChatResp", events[0].Payload)
	}
	if resp.RunID != "run-1" || resp.ThreadID != "thread-1" {
		t.Fatalf("payload was not enriched: %+v", resp)
	}
}
