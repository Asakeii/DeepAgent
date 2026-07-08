package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func ListRunEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		http.Error(w, "run_id required", http.StatusBadRequest)
		return
	}
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	events, err := store.ListRunEvents(r.Context(), infra.DB, runID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.RunEventResp, 0, len(events))
	for _, event := range events {
		var payload any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			payload = string(event.Payload)
		}
		out = append(out, &model.RunEventResp{
			ID:        event.ID,
			RunID:     event.RunID,
			ThreadID:  event.ThreadID,
			UserID:    event.UserID,
			EventName: event.EventName,
			Agent:     event.Agent,
			Payload:   payload,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListRunEventsResponse{Events: out})
}
