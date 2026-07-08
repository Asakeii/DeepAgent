package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func Memories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listMemories(w, r)
	case http.MethodPost:
		createMemory(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listMemories(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	records, err := store.ListMemories(r.Context(), infra.DB, userID, r.URL.Query().Get("kind"), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.MemoryResp, 0, len(records))
	for _, record := range records {
		out = append(out, memoryResp(record))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListMemoriesResponse{Memories: out})
}

func createMemory(w http.ResponseWriter, r *http.Request) {
	var req model.CreateMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	userID := requestUserID(r)
	if req.ThreadID != "" {
		if ok, err := store.ThreadBelongsToUser(r.Context(), infra.DB, req.ThreadID, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(w, "thread forbidden", http.StatusForbidden)
			return
		}
	}
	record := store.MemoryRecord{
		UserID:     userID,
		ThreadID:   req.ThreadID,
		Kind:       req.Kind,
		Content:    req.Content,
		Importance: req.Importance,
		Source:     req.Source,
	}
	if record.Source == "" {
		record.Source = "api"
	}
	id, err := store.CreateMemory(r.Context(), infra.DB, record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	record.ID = id
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.CreateMemoryResponse{Memory: memoryResp(record)})
}

func memoryResp(record store.MemoryRecord) *model.MemoryResp {
	resp := &model.MemoryResp{
		ID:         record.ID,
		UserID:     record.UserID,
		ThreadID:   record.ThreadID,
		Kind:       record.Kind,
		Content:    record.Content,
		Importance: record.Importance,
		Source:     record.Source,
	}
	if !record.CreatedAt.IsZero() {
		resp.CreatedAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
