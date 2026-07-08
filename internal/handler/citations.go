package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func ListArtifactCitations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	artifactID, err := strconv.ParseInt(r.URL.Query().Get("artifact_id"), 10, 64)
	if err != nil || artifactID <= 0 {
		http.Error(w, "artifact_id required", http.StatusBadRequest)
		return
	}
	userID := requestUserID(r)
	if ok, err := store.ArtifactBelongsToUser(r.Context(), infra.DB, artifactID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "artifact forbidden", http.StatusForbidden)
		return
	}
	records, err := store.ListArtifactCitations(r.Context(), infra.DB, userID, artifactID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.CitationResp, 0, len(records))
	for _, record := range records {
		out = append(out, citationResp(record))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListCitationsResponse{Citations: out})
}

func citationResp(record store.CitationRecord) *model.CitationResp {
	resp := &model.CitationResp{
		ID:         record.ID,
		ArtifactID: record.ArtifactID,
		UserID:     record.UserID,
		ThreadID:   record.ThreadID,
		RunID:      record.RunID,
		Title:      record.Title,
		URL:        record.URL,
		Quote:      record.Quote,
		Position:   record.Position,
	}
	if !record.CreatedAt.IsZero() {
		resp.CreatedAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
