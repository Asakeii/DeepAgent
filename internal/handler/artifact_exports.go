package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"deepAgent/internal/infra"
	"deepAgent/internal/store"
)

var artifactMarkdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

func ExportArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "html"
	}
	if format != "html" {
		http.Error(w, "unsupported artifact export format", http.StatusBadRequest)
		return
	}
	artifactID, err := strconv.ParseInt(r.URL.Query().Get("artifact_id"), 10, 64)
	if err != nil || artifactID <= 0 {
		http.Error(w, "artifact_id required", http.StatusBadRequest)
		return
	}
	record, err := store.GetArtifact(r.Context(), infra.DB, artifactID, requestUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if record.ID == 0 {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	doc, err := RenderArtifactHTML(record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filename := artifactExportFilename(record)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = w.Write([]byte(doc))
}

func RenderArtifactHTML(record store.ArtifactRecord) (string, error) {
	var body bytes.Buffer
	if err := artifactMarkdown.Convert([]byte(record.Content), &body); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	data := struct {
		Title     string
		Kind      string
		Version   int64
		CreatedAt string
		Body      template.HTML
	}{
		Title:   record.Title,
		Kind:    record.Kind,
		Version: record.Version,
		Body:    template.HTML(body.String()),
	}
	if !record.CreatedAt.IsZero() {
		data.CreatedAt = record.CreatedAt.Format("2006-01-02 15:04:05")
	}
	var out bytes.Buffer
	if err := artifactHTMLTemplate.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render artifact html: %w", err)
	}
	return out.String(), nil
}

var artifactHTMLTemplate = template.Must(template.New("artifact-html").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <style>
    :root { color-scheme: light; --text: #17202a; --muted: #667085; --line: #d0d5dd; --accent: #175cd3; }
    body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", "PingFang SC", sans-serif; color: var(--text); background: #ffffff; }
    main { max-width: 860px; margin: 0 auto; padding: 48px 28px 72px; }
    header { border-bottom: 1px solid var(--line); margin-bottom: 32px; padding-bottom: 20px; }
    h1 { font-size: 32px; line-height: 1.25; margin: 0 0 12px; }
    .meta { color: var(--muted); font-size: 14px; display: flex; gap: 12px; flex-wrap: wrap; }
    article { font-size: 16px; line-height: 1.72; }
    article h1, article h2, article h3 { line-height: 1.35; margin-top: 1.6em; }
    article pre { overflow-x: auto; padding: 16px; background: #f2f4f7; border-radius: 6px; }
    article code { background: #f2f4f7; padding: 0.1em 0.3em; border-radius: 4px; }
    article pre code { padding: 0; background: transparent; }
    article blockquote { margin-left: 0; padding-left: 16px; border-left: 3px solid var(--line); color: #475467; }
    article table { border-collapse: collapse; width: 100%; overflow: auto; }
    article th, article td { border: 1px solid var(--line); padding: 8px 10px; text-align: left; }
    article a { color: var(--accent); }
    @media print { main { max-width: none; padding: 24px; } a { color: inherit; } }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>{{ .Title }}</h1>
      <div class="meta">
        {{ if .Kind }}<span>{{ .Kind }}</span>{{ end }}
        {{ if .Version }}<span>v{{ .Version }}</span>{{ end }}
        {{ if .CreatedAt }}<span>{{ .CreatedAt }}</span>{{ end }}
      </div>
    </header>
    <article>
      {{ .Body }}
    </article>
  </main>
</body>
</html>`))

var filenameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func artifactExportFilename(record store.ArtifactRecord) string {
	name := strings.TrimSpace(record.Title)
	if name == "" {
		name = fmt.Sprintf("artifact-%d", record.ID)
	}
	name = filenameUnsafe.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._")
	if name == "" {
		name = fmt.Sprintf("artifact-%d", record.ID)
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name + ".html"
}
