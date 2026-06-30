package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// searxngBaseURL is the SearXNG instance endpoint (Docker service name).
var searxngBaseURL = ""

func init() {
	searxngBaseURL = "http://searxng:8080"
	if u := os.Getenv("SEARXNG_URL"); u != "" {
		searxngBaseURL = strings.TrimRight(u, "/")
	}
}

// SetSearXNGURL allows the main package to override the SearXNG URL.
func SetSearXNGURL(u string) {
	searxngBaseURL = strings.TrimRight(u, "/")
}

// ---------------------------------------------------------------------------
// SearXNG JSON API response types
// ---------------------------------------------------------------------------

type searxngResponse struct {
	Query   string          `json:"query"`
	Results []searxngResult `json:"results"`
}

type searxngResult struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Content     string   `json:"content"`
	Engine      string   `json:"engine"`
	Score       float64  `json:"score"`
	Published   string   `json:"publishedDate"`
	Engines     []string `json:"engines"`
}

// ---------------------------------------------------------------------------
// Tool input / output structs
// ---------------------------------------------------------------------------

type webSearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
	Num   int    `json:"num" jsonschema_description:"返回结果数量，默认 5，最大 10"`
}

type webSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type webSearchOutput struct {
	Query   string            `json:"query"`
	Results []webSearchResult `json:"results"`
}

// ---------------------------------------------------------------------------
// Core search logic
// ---------------------------------------------------------------------------

func searchWeb(ctx context.Context, in webSearchInput) (webSearchOutput, error) {
	if in.Query == "" {
		return webSearchOutput{}, fmt.Errorf("query is required")
	}
	if in.Num <= 0 || in.Num > 10 {
		in.Num = 5
	}

	// Build SearXNG JSON API URL: /search?q=...&format=json&pageno=1
	apiURL := fmt.Sprintf("%s/search?format=json&pageno=1&q=%s",
		searxngBaseURL, url.QueryEscape(in.Query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return webSearchOutput{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "deepAgent/1.0")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return webSearchOutput{}, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return webSearchOutput{}, fmt.Errorf("searxng returned %d: %s", resp.StatusCode, string(body))
	}

	var sr searxngResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return webSearchOutput{}, fmt.Errorf("parse search results: %w", err)
	}

	out := webSearchOutput{Query: sr.Query}
	for i, r := range sr.Results {
		if i >= in.Num {
			break
		}
		// Clean snippet: strip HTML tags, truncate
		snippet := cleanHTML(r.Content)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		out.Results = append(out.Results, webSearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: snippet,
		})
	}

	if len(out.Results) == 0 {
		out.Results = []webSearchResult{}
	}

	return out, nil
}

// ---------------------------------------------------------------------------
// Web fetch: read a single page's text content
// ---------------------------------------------------------------------------

type webFetchInput struct {
	URL string `json:"url" jsonschema:"required" jsonschema_description:"要抓取的网页 URL"`
}

type webFetchOutput struct {
	URL     string `json:"url"`
	Content string `json:"content"`
}

func fetchPage(ctx context.Context, in webFetchInput) (webFetchOutput, error) {
	if in.URL == "" {
		return webFetchOutput{}, fmt.Errorf("url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, in.URL, nil)
	if err != nil {
		return webFetchOutput{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; deepAgent/1.0)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return webFetchOutput{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return webFetchOutput{}, fmt.Errorf("page returned %d", resp.StatusCode)
	}

	// Read up to 256KB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return webFetchOutput{}, fmt.Errorf("read page: %w", err)
	}

	// Extract text from HTML
	text := extractText(string(body))
	// Truncate to ~6000 chars for model context
	if len(text) > 6000 {
		text = text[:6000] + "...[truncated]"
	}

	return webFetchOutput{URL: in.URL, Content: text}, nil
}

// ---------------------------------------------------------------------------
// HTML utilities
// ---------------------------------------------------------------------------

func cleanHTML(s string) string {
	// Remove HTML tags
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	// Normalize whitespace
	result := strings.Join(strings.Fields(b.String()), " ")
	return result
}

func extractText(html string) string {
	// Remove script and style blocks
	html = removeTag(html, "script")
	html = removeTag(html, "style")
	html = removeTag(html, "noscript")
	html = removeTag(html, "nav")
	html = removeTag(html, "footer")
	html = removeTag(html, "header")

	// Extract text from remaining HTML
	text := cleanHTML(html)
	if len(text) > 200 {
		// Only show the "meat" — skip very short or empty pages
		return text
	}
	return ""
}

func removeTag(html, tag string) string {
	// Simple regex-free tag remover
	var result strings.Builder
	result.Grow(len(html))
	i := 0
	openTag := "<" + tag
	closeTag := "</" + tag + ">"
	for i < len(html) {
		// Find opening tag
		idx := strings.Index(strings.ToLower(html[i:]), openTag)
		if idx < 0 {
			result.WriteString(html[i:])
			break
		}
		result.WriteString(html[i : i+idx])

		// Find matching closing tag
		i += idx
		closeIdx := strings.Index(strings.ToLower(html[i:]), closeTag)
		if closeIdx < 0 {
			// No closing tag found, skip to end of this element
			gtIdx := strings.Index(html[i:], ">")
			if gtIdx >= 0 {
				i += gtIdx + 1
			} else {
				i = len(html)
			}
			break
		}
		i += closeIdx + len(closeTag)
	}
	return result.String()
}

// ---------------------------------------------------------------------------
// Eino tool constructors
// ---------------------------------------------------------------------------

// NewWebSearchTool creates a native web search tool backed by SearXNG.
// This replaces the need for MCP-based search tools like Tavily.
func NewWebSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("web_search",
		"搜索互联网获取最新信息。输入搜索关键词，返回相关网页的标题、URL 和摘要内容。",
		searchWeb)
}

// NewWebFetchTool creates a tool that fetches and extracts text from a web page.
func NewWebFetchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("web_fetch",
		"抓取指定 URL 的网页内容并提取纯文本。用于读取搜索结果中的页面详情。输入完整的 HTTP/HTTPS URL。",
		fetchPage)
}
