package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/store"
)

// ListSessions 返回会话列表。
func ListSessions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	threads, err := store.ListThreads(r.Context(), infra.DB, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if threads == nil {
		threads = []store.ThreadInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threads)
}

// LoadMessages 返回某 thread 的历史消息。
func LoadMessages(w http.ResponseWriter, r *http.Request) {
	threadID := r.URL.Query().Get("thread_id")
	if threadID == "" {
		http.Error(w, "thread_id required", http.StatusBadRequest)
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	msgs, err := store.RecentMessages(r.Context(), infra.DB, threadID, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	type msgOut struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	out := make([]msgOut, len(msgs))
	for i, m := range msgs {
		out[i] = msgOut{Role: string(m.Role), Content: m.Content}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
