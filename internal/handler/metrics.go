package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func RunMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := requestUserID(r)
	windowHours := 24
	if raw := r.URL.Query().Get("window_hours"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			windowHours = n
		}
	}
	if windowHours > 24*30 {
		windowHours = 24 * 30
	}

	metrics, err := store.GetRunMetrics(r.Context(), infra.DB, userID, windowHours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.RunMetricsResp{
		UserID:            metrics.UserID,
		WindowHours:       metrics.WindowHours,
		RunsTotal:         metrics.RunsTotal,
		RunsSucceeded:     metrics.RunsSucceeded,
		RunsFailed:        metrics.RunsFailed,
		RunsRunning:       metrics.RunsRunning,
		RunSuccessRate:    metrics.RunSuccessRate,
		AvgRunLatencyMS:   metrics.AvgRunLatencyMS,
		P95RunLatencyMS:   metrics.P95RunLatencyMS,
		ToolsTotal:        metrics.ToolsTotal,
		ToolsFailed:       metrics.ToolsFailed,
		ToolsBlocked:      metrics.ToolsBlocked,
		ToolErrorRate:     metrics.ToolErrorRate,
		AvgToolDurationMS: metrics.AvgToolDurationMS,
	})
}
