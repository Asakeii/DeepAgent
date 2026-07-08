package app

import (
	"testing"

	"deepAgent/internal/model"
)

type noopWriter struct{}

func (noopWriter) WriteEvent(event string, payload any) error {
	return nil
}

func TestRouteDetectingWriterDetectsCheckinToolCall(t *testing.T) {
	writer := newRouteDetectingWriter(noopWriter{})

	_ = writer.WriteEvent("tool_calls", &model.ChatResp{
		ToolCalls: []model.ToolResp{{Name: "hand_to_checkin"}},
	})

	if !writer.RouteToCheckin() {
		t.Fatal("RouteToCheckin() = false, want true")
	}
}

func TestRouteDetectingWriterIgnoresOtherToolCalls(t *testing.T) {
	writer := newRouteDetectingWriter(noopWriter{})

	_ = writer.WriteEvent("tool_calls", &model.ChatResp{
		ToolCalls: []model.ToolResp{{Name: "hand_to_planner"}},
	})

	if writer.RouteToCheckin() {
		t.Fatal("RouteToCheckin() = true, want false")
	}
}
