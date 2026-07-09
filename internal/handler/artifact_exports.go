package handler

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"deepAgent/conf"
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
	if format != "html" && format != "pdf" {
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
	switch format {
	case "html":
		doc, err := RenderArtifactHTML(record)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		filename := artifactExportFilename(record, "html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		_, _ = w.Write([]byte(doc))
	case "pdf":
		doc, err := RenderArtifactPDF(r.Context(), record)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "pdf renderer") {
				status = http.StatusServiceUnavailable
			}
			http.Error(w, err.Error(), status)
			return
		}
		filename := artifactExportFilename(record, "pdf")
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		_, _ = w.Write(doc)
	}
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

func RenderArtifactPDF(ctx context.Context, record store.ArtifactRecord) ([]byte, error) {
	htmlDoc, err := RenderArtifactHTML(record)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "deepagent-artifact-export-*")
	if err != nil {
		return nil, fmt.Errorf("create pdf temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "artifact.html")
	outputPath := filepath.Join(dir, "artifact.pdf")
	if err := os.WriteFile(inputPath, []byte(htmlDoc), 0o600); err != nil {
		return nil, fmt.Errorf("write pdf html: %w", err)
	}
	command, args, err := pdfRendererCommand(inputPath, outputPath)
	if err != nil {
		return nil, err
	}
	timeout := 30 * time.Second
	if conf.App != nil {
		timeout = conf.App.Server.PDFRendererTimeoutDuration()
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, command, args...)
	output, err := cmd.CombinedOutput()
	if runCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("pdf renderer timed out after %s", timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("pdf renderer failed: %w: %s", err, trimRendererOutput(string(output)))
	}
	pdf, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read rendered pdf: %w", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		return nil, fmt.Errorf("pdf renderer produced invalid pdf")
	}
	return pdf, nil
}

func pdfRendererCommand(inputPath, outputPath string) (string, []string, error) {
	if conf.App != nil && strings.TrimSpace(conf.App.Server.PDFRendererCommand) != "" {
		command := strings.TrimSpace(conf.App.Server.PDFRendererCommand)
		args := conf.App.Server.PDFRendererArgs
		if len(args) == 0 {
			args = defaultPDFRendererArgs()
		}
		return command, renderPDFArgs(args, inputPath, outputPath), nil
	}
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, renderPDFArgs(defaultPDFRendererArgs(), inputPath, outputPath), nil
		}
	}
	return "", nil, fmt.Errorf("pdf renderer command not configured or found")
}

func defaultPDFRendererArgs() []string {
	return []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--no-pdf-header-footer",
		"--print-to-pdf={{output}}",
		"{{input}}",
	}
}

func renderPDFArgs(args []string, inputPath, outputPath string) []string {
	replacements := map[string]string{
		"{{input}}":      fileURL(inputPath),
		"{{input_path}}": inputPath,
		"{{output}}":     outputPath,
	}
	out := make([]string, len(args))
	for i, arg := range args {
		for key, value := range replacements {
			arg = strings.ReplaceAll(arg, key, value)
		}
		out[i] = arg
	}
	return out
}

func fileURL(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func trimRendererOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 1200 {
		return output[:1200] + "...[truncated]"
	}
	return output
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

func artifactExportFilename(record store.ArtifactRecord, ext string) string {
	return artifactExportBaseName(record) + "." + ext
}

func artifactExportBaseName(record store.ArtifactRecord) string {
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
	return name
}
