package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func AdminOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	windowHours := 24
	if raw := r.URL.Query().Get("window_hours"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			windowHours = n
		}
	}
	overview, err := store.GetAdminOverview(r.Context(), infra.DB, windowHours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.AdminOverviewResp{
		WindowHours:      overview.WindowHours,
		UsersTotal:       overview.UsersTotal,
		ThreadsTotal:     overview.ThreadsTotal,
		ArtifactsTotal:   overview.ArtifactsTotal,
		ArtifactShares:   overview.ArtifactShares,
		RunsTotal:        overview.RunsTotal,
		RunsSucceeded:    overview.RunsSucceeded,
		RunsFailed:       overview.RunsFailed,
		RunsRunning:      overview.RunsRunning,
		RunSuccessRate:   overview.RunSuccessRate,
		ToolsTotal:       overview.ToolsTotal,
		ToolsFailed:      overview.ToolsFailed,
		ToolsBlocked:     overview.ToolsBlocked,
		ToolErrorRate:    overview.ToolErrorRate,
		PromptTokens:     overview.PromptTokens,
		CompletionTokens: overview.CompletionTokens,
		TotalTokens:      overview.TotalTokens,
		CachedTokens:     overview.CachedTokens,
		ReasoningTokens:  overview.ReasoningTokens,
	})
}
