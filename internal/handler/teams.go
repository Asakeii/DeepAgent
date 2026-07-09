package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func Teams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listTeams(w, r)
	case http.MethodPost:
		createTeam(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func TeamMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listTeamMembers(w, r)
	case http.MethodPost:
		addTeamMember(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func listTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := store.ListTeamsForUser(r.Context(), infra.DB, requestUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]*model.TeamResp, 0, len(teams))
	for _, team := range teams {
		out = append(out, teamResp(team))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListTeamsResponse{Teams: out})
}

func createTeam(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	team, err := store.CreateTeam(r.Context(), infra.DB, requestUserID(r), req.Name)
	if err != nil {
		if errors.Is(err, store.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(model.CreateTeamResponse{Team: teamResp(team)})
}

func listTeamMembers(w http.ResponseWriter, r *http.Request) {
	members, err := store.ListTeamMembers(r.Context(), infra.DB, r.URL.Query().Get("team_id"), requestUserID(r))
	if err != nil {
		writeTeamError(w, err)
		return
	}
	out := make([]*model.TeamMemberResp, 0, len(members))
	for _, member := range members {
		out = append(out, teamMemberResp(member))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ListTeamMembersResponse{Members: out})
}

func addTeamMember(w http.ResponseWriter, r *http.Request) {
	var req model.AddTeamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	member, err := store.AddTeamMember(r.Context(), infra.DB, req.TeamID, requestUserID(r), req.UserID, req.Role)
	if err != nil {
		writeTeamError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.AddTeamMemberResponse{Member: teamMemberResp(member)})
}

func writeTeamError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrValidation):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, store.ErrTeamForbidden):
		http.Error(w, "team forbidden", http.StatusForbidden)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func teamResp(record store.TeamRecord) *model.TeamResp {
	resp := &model.TeamResp{
		ID:   record.ID,
		Name: record.Name,
		Role: record.Role,
	}
	if !record.CreatedAt.IsZero() {
		resp.CreatedAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}

func teamMemberResp(record store.TeamMemberRecord) *model.TeamMemberResp {
	resp := &model.TeamMemberResp{
		TeamID: record.TeamID,
		UserID: record.UserID,
		Role:   record.Role,
	}
	if !record.CreatedAt.IsZero() {
		resp.CreatedAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
