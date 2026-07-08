package security

import (
	"encoding/json"
	"regexp"
	"strings"
)

const Redacted = "[REDACTED]"

var secretKeyFragments = []string{
	"authorization",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"token",
	"password",
	"passwd",
	"secret",
	"credential",
}

var secretTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)((api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret)\s*[:=]\s*)[^\s,;&]+`),
}

func RedactString(value string) string {
	out := value
	for _, pattern := range secretTextPatterns {
		out = pattern.ReplaceAllString(out, "${1}"+Redacted)
	}
	return out
}

func RedactJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		b, _ := json.Marshal(map[string]string{"raw": RedactString(string(raw))})
		return b
	}
	redacted := redactValue(value, "")
	b, err := json.Marshal(redacted)
	if err != nil {
		return raw
	}
	return b
}

func redactValue(value any, key string) any {
	if isSensitiveKey(key) {
		return Redacted
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = redactValue(v, k)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			out[i] = redactValue(v, key)
		}
		return out
	case string:
		return RedactString(typed)
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	for _, fragment := range secretKeyFragments {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}
