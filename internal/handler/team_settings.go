package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func TeamSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getTeamSettings(w, r)
	case http.MethodPut:
		updateTeamSettings(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTeamSettings(w http.ResponseWriter, r *http.Request) {
	teamID := r.URL.Query().Get("team_id")
	userID := requestUserID(r)
	if ok, err := store.UserIsTeamMember(r.Context(), infra.DB, teamID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "team forbidden", http.StatusForbidden)
		return
	}
	record, err := store.GetTeamSettings(r.Context(), infra.DB, teamID)
	if err != nil {
		writeSettingsError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.TeamSettingsResponse{Settings: teamSettingsResp(record)})
}

func updateTeamSettings(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	var req model.UpdateTeamSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if ok, err := store.UserCanManageTeam(r.Context(), infra.DB, req.TeamID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !ok {
		http.Error(w, "team forbidden", http.StatusForbidden)
		return
	}
	record, err := store.GetTeamSettings(r.Context(), infra.DB, req.TeamID)
	if err != nil {
		writeSettingsError(w, err)
		return
	}
	record.UpdatedBy = userID
	if req.DailyCostBudgetMicros != nil {
		if *req.DailyCostBudgetMicros <= 0 {
			record.DailyCostBudgetMicros = sql.NullInt64{}
		} else {
			record.DailyCostBudgetMicros = sql.NullInt64{Int64: *req.DailyCostBudgetMicros, Valid: true}
		}
	}
	if err := store.UpsertTeamSettings(r.Context(), infra.DB, record); err != nil {
		writeSettingsError(w, err)
		return
	}
	record, err = store.GetTeamSettings(r.Context(), infra.DB, req.TeamID)
	if err != nil {
		writeSettingsError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.TeamSettingsResponse{Settings: teamSettingsResp(record)})
}

func writeSettingsError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrValidation) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func teamSettingsResp(record store.TeamSettingsRecord) *model.TeamSettingsResp {
	resp := &model.TeamSettingsResp{
		TeamID:    record.TeamID,
		UpdatedBy: record.UpdatedBy,
	}
	if record.DailyCostBudgetMicros.Valid {
		v := record.DailyCostBudgetMicros.Int64
		resp.DailyCostBudgetMicros = &v
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
