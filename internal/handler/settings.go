package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func UserSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getUserSettings(w, r)
	case http.MethodPut:
		updateUserSettings(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func getUserSettings(w http.ResponseWriter, r *http.Request) {
	record, err := store.GetUserSettings(r.Context(), infra.DB, requestUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.UserSettingsResponse{Settings: userSettingsResp(record)})
}

func updateUserSettings(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	record, err := store.GetUserSettings(r.Context(), infra.DB, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var req model.UpdateUserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	applyUserSettingsUpdate(&record, req)
	if record.ModelProfile != "" && !infra.HasModelProfile(record.ModelProfile) {
		http.Error(w, "unknown model_profile", http.StatusBadRequest)
		return
	}
	if err := store.UpsertUserSettings(r.Context(), infra.DB, record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	record, err = store.GetUserSettings(r.Context(), infra.DB, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.UserSettingsResponse{Settings: userSettingsResp(record)})
}

func applyUserSettingsUpdate(record *store.UserSettingsRecord, req model.UpdateUserSettingsRequest) {
	if req.Locale != nil {
		record.Locale = *req.Locale
	}
	if req.Timezone != nil {
		record.Timezone = *req.Timezone
	}
	if req.ModelProfile != nil {
		record.ModelProfile = *req.ModelProfile
	}
	if req.MaxPlanIterations != nil {
		record.MaxPlanIterations = sql.NullInt64{Int64: int64(*req.MaxPlanIterations), Valid: true}
	}
	if req.MaxStepNum != nil {
		record.MaxStepNum = sql.NullInt64{Int64: int64(*req.MaxStepNum), Valid: true}
	}
	if req.DailyTokenBudget != nil {
		if *req.DailyTokenBudget <= 0 {
			record.DailyTokenBudget = sql.NullInt64{}
		} else {
			record.DailyTokenBudget = sql.NullInt64{Int64: int64(*req.DailyTokenBudget), Valid: true}
		}
	}
	if req.DailyCostBudgetMicros != nil {
		if *req.DailyCostBudgetMicros <= 0 {
			record.DailyCostBudgetMicros = sql.NullInt64{}
		} else {
			record.DailyCostBudgetMicros = sql.NullInt64{Int64: *req.DailyCostBudgetMicros, Valid: true}
		}
	}
	if req.EnableBackgroundInvestigation != nil {
		record.EnableBackgroundInvestigation = sql.NullBool{Bool: *req.EnableBackgroundInvestigation, Valid: true}
	}
	if req.AutoAcceptPlan != nil {
		record.AutoAcceptPlan = sql.NullBool{Bool: *req.AutoAcceptPlan, Valid: true}
	}
}

func userSettingsResp(record store.UserSettingsRecord) *model.UserSettingsResp {
	resp := &model.UserSettingsResp{
		UserID:       record.UserID,
		Locale:       record.Locale,
		Timezone:     record.Timezone,
		ModelProfile: record.ModelProfile,
	}
	if record.MaxPlanIterations.Valid {
		v := int(record.MaxPlanIterations.Int64)
		resp.MaxPlanIterations = &v
	}
	if record.MaxStepNum.Valid {
		v := int(record.MaxStepNum.Int64)
		resp.MaxStepNum = &v
	}
	if record.DailyTokenBudget.Valid {
		v := int(record.DailyTokenBudget.Int64)
		resp.DailyTokenBudget = &v
	}
	if record.DailyCostBudgetMicros.Valid {
		v := record.DailyCostBudgetMicros.Int64
		resp.DailyCostBudgetMicros = &v
	}
	if record.EnableBackgroundInvestigation.Valid {
		v := record.EnableBackgroundInvestigation.Bool
		resp.EnableBackgroundInvestigation = &v
	}
	if record.AutoAcceptPlan.Valid {
		v := record.AutoAcceptPlan.Bool
		resp.AutoAcceptPlan = &v
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
