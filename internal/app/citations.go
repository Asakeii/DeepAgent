package app

import (
	"net/url"
	"regexp"
	"strings"

	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

var markdownLinkPattern = regexp.MustCompile(`\[([^\]]{1,512})\]\((https?://[^)\s]+)\)`)

func citationRecordsFromMarkdown(artifactID int64, req model.ChatRequest, content string) []store.CitationRecord {
	if artifactID <= 0 || strings.TrimSpace(content) == "" {
		return nil
	}
	matches := markdownLinkPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	records := make([]store.CitationRecord, 0, len(matches))
	for _, match := range matches {
		title := content[match[2]:match[3]]
		rawURL := strings.TrimRight(content[match[4]:match[5]], ".,;")
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		if seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		records = append(records, store.CitationRecord{
			ArtifactID: artifactID,
			UserID:     req.UserID,
			ThreadID:   req.ThreadID,
			RunID:      req.RunID,
			Title:      title,
			URL:        rawURL,
			Position:   len(records) + 1,
		})
	}
	return records
}
