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
	userID := requestUserID(r)
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var teamID *string
	if _, ok := r.URL.Query()["team_id"]; ok {
		v := r.URL.Query().Get("team_id")
		teamID = &v
	}
	threads, err := store.SearchThreadsForUserInScope(r.Context(), infra.DB, userID, r.URL.Query().Get("q"), teamID, limit)
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
	userID := requestUserID(r)
	threadID := r.URL.Query().Get("thread_id")
	if threadID == "" {
		http.Error(w, "thread_id required", http.StatusBadRequest)
		return
	}
	if ok, err := store.ThreadBelongsToUser(r.Context(), infra.DB, threadID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "thread forbidden", http.StatusForbidden)
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
