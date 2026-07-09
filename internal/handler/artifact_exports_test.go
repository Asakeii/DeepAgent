package handler

import (
	"strings"
	"testing"
	"time"

	"deepAgent/internal/store"
)

func TestRenderArtifactHTML(t *testing.T) {
	doc, err := RenderArtifactHTML(store.ArtifactRecord{
		ID:        42,
		Kind:      store.ArtifactKindReport,
		Title:     "成熟 Agent 报告",
		Version:   2,
		Content:   "# 结论\n\n- 支持 **多 pod**\n\n<script>alert('x')</script>",
		CreatedAt: time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<title>成熟 Agent 报告</title>",
		"<h1>成熟 Agent 报告</h1>",
		"<h1>结论</h1>",
		"<strong>多 pod</strong>",
		"v2",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("rendered HTML missing %q:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "<script>alert") {
		t.Fatalf("raw script should not be rendered as executable HTML:\n%s", doc)
	}
}

func TestArtifactExportFilename(t *testing.T) {
	got := artifactExportFilename(store.ArtifactRecord{ID: 7, Title: "Report: A/B Test?"})
	if got != "Report-A-B-Test.html" {
		t.Fatalf("filename=%q", got)
	}
	got = artifactExportFilename(store.ArtifactRecord{ID: 8, Title: "中文报告"})
	if got != "artifact-8.html" {
		t.Fatalf("filename=%q", got)
	}
}
