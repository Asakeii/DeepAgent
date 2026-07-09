package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"deepAgent/conf"
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
	got := artifactExportFilename(store.ArtifactRecord{ID: 7, Title: "Report: A/B Test?"}, "html")
	if got != "Report-A-B-Test.html" {
		t.Fatalf("filename=%q", got)
	}
	got = artifactExportFilename(store.ArtifactRecord{ID: 8, Title: "中文报告"}, "pdf")
	if got != "artifact-8.pdf" {
		t.Fatalf("filename=%q", got)
	}
}

func TestRenderArtifactPDFWithConfiguredRenderer(t *testing.T) {
	previous := conf.App
	conf.App = &conf.Config{Server: conf.ServerConfig{
		PDFRendererCommand: "/bin/sh",
		PDFRendererArgs: []string{
			"-c",
			"printf '%s' '%PDF-1.4 fake' > \"$1\"",
			"renderer",
			"{{output}}",
		},
		PDFRendererTimeout: 5,
	}}
	t.Cleanup(func() { conf.App = previous })

	doc, err := RenderArtifactPDF(context.Background(), store.ArtifactRecord{
		ID:      9,
		Kind:    store.ArtifactKindReport,
		Title:   "中文报告",
		Content: "# 标题\n\n支持中文 PDF",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(doc), "%PDF-1.4") {
		t.Fatalf("pdf=%q, want PDF header", doc)
	}
}

func TestRenderPDFArgs(t *testing.T) {
	args := renderPDFArgs([]string{"--print-to-pdf={{output}}", "{{input}}", "{{input_path}}"}, "/tmp/a b.html", "/tmp/out.pdf")
	if args[0] != "--print-to-pdf=/tmp/out.pdf" {
		t.Fatalf("output arg=%q", args[0])
	}
	if args[1] != "file:///tmp/a%20b.html" {
		t.Fatalf("input url=%q", args[1])
	}
	if args[2] != "/tmp/a b.html" {
		t.Fatalf("input path=%q", args[2])
	}
}
