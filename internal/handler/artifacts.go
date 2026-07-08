package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func ListArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := requestUserID(r)
	threadID := r.URL.Query().Get("thread_id")
	if threadID != "" {
		if ok, err := store.ThreadBelongsToUser(r.Context(), infra.DB, threadID, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(w, "thread forbidden", http.StatusForbidden)
			return
		}
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	records, err := store.ListArtifacts(r.Context(), infra.DB, userID, threadID, r.URL.Query().Get("kind"), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.ArtifactResp, 0, len(records))
	for _, record := range records {
		out = append(out, artifactResp(record))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListArtifactsResponse{Artifacts: out})
}

func artifactResp(record store.ArtifactRecord) *model.ArtifactResp {
	resp := &model.ArtifactResp{
		ID:       record.ID,
		UserID:   record.UserID,
		ThreadID: record.ThreadID,
		RunID:    record.RunID,
		Kind:     record.Kind,
		Title:    record.Title,
		Format:   record.Format,
		Content:  record.Content,
		Version:  record.Version,
		Source:   record.Source,
	}
	if !record.CreatedAt.IsZero() {
		resp.CreatedAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
