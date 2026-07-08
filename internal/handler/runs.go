package handler

import (
	"encoding/json"
	"net/http"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func CancelRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.CancelRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RunID == "" {
		http.Error(w, "run_id required", http.StatusBadRequest)
		return
	}
	userID := requestUserID(r)
	if ok, err := store.RunBelongsToUser(r.Context(), infra.DB, req.RunID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "run forbidden", http.StatusForbidden)
		return
	}
	cancelled, err := store.CancelRun(r.Context(), infra.DB, req.RunID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	run, err := store.GetRun(r.Context(), infra.DB, req.RunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cancelled {
		if err := store.AppendRunCancelledEvent(r.Context(), infra.DB, run); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.CancelRunResponse{
		RunID:     req.RunID,
		Status:    run.Status,
		Cancelled: cancelled,
	})
}
