package app

import (
	"testing"

	"deepAgent/internal/model"
)

func TestCitationRecordsFromMarkdown(t *testing.T) {
	req := model.ChatRequest{RunID: "run-1", ThreadID: "thread-1", UserID: "user-1"}
	records := citationRecordsFromMarkdown(42, req, `参考 [OpenAI](https://openai.com/research), 以及重复 [OpenAI](https://openai.com/research)。`)
	if len(records) != 1 {
		t.Fatalf("len=%d, want 1", len(records))
	}
	if records[0].ArtifactID != 42 || records[0].Title != "OpenAI" || records[0].URL != "https://openai.com/research" {
		t.Fatalf("unexpected record: %+v", records[0])
	}
	if records[0].Position != 1 {
		t.Fatalf("position=%d, want 1", records[0].Position)
	}
}

func TestCitationRecordsFromMarkdownIgnoresInvalidLinks(t *testing.T) {
	records := citationRecordsFromMarkdown(42, model.ChatRequest{}, `[bad](javascript:alert(1)) [relative](/docs)`)
	if len(records) != 0 {
		t.Fatalf("len=%d, want 0", len(records))
	}
}
