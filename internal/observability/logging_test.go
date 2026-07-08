package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestRunLoggerAddsStableFields(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	defer slog.SetDefault(prev)
	ConfigureJSONLogger(&buf)

	RunLogger("run-1", "thread-1", "user-1").Info("hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log: %v\n%s", err, buf.String())
	}
	if record[LogKeyRunID] != "run-1" || record[LogKeyThreadID] != "thread-1" || record[LogKeyUserID] != "user-1" {
		t.Fatalf("missing run fields: %#v", record)
	}
	if record["msg"] != "hello" {
		t.Fatalf("msg=%v, want hello", record["msg"])
	}
}
