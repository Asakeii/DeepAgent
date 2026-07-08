package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func ListToolAudits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		http.Error(w, "run_id required", http.StatusBadRequest)
		return
	}
	userID := requestUserID(r)
	if ok, err := store.RunBelongsToUser(r.Context(), infra.DB, runID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "run forbidden", http.StatusForbidden)
		return
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	audits, err := store.ListToolAuditsByRun(r.Context(), infra.DB, runID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]*model.ToolAuditResp, 0, len(audits))
	for _, audit := range audits {
		var args any
		if err := json.Unmarshal(audit.Arguments, &args); err != nil {
			args = string(audit.Arguments)
		}
		out = append(out, &model.ToolAuditResp{
			ID:         audit.ID,
			RunID:      audit.RunID,
			ThreadID:   audit.ThreadID,
			UserID:     audit.UserID,
			ToolName:   audit.ToolName,
			Risk:       audit.Risk,
			Status:     audit.Status,
			Arguments:  args,
			Result:     audit.Result,
			Error:      audit.Error,
			DurationMS: audit.DurationMS,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListToolAuditsResponse{Audits: out})
}
