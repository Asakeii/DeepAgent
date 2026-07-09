package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"deepAgent/conf"
	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

func Plugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := requestUserID(r)
	teamID := strings.TrimSpace(r.URL.Query().Get("team_id"))
	if teamID != "" {
		if ok, err := store.UserIsTeamMember(r.Context(), infra.DB, teamID, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(w, "team forbidden", http.StatusForbidden)
			return
		}
	}
	scopeType, scopeID := store.PluginScopeFor(userID, teamID)
	resp, err := pluginCatalogResponse(r, userID, teamID, scopeType, scopeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func PluginInstalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.UpdatePluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	userID := requestUserID(r)
	server := strings.TrimSpace(req.Server)
	if !configuredPluginExists(server) {
		http.Error(w, "plugin server not found", http.StatusNotFound)
		return
	}
	scopeType, scopeID := store.PluginScopeFor(userID, req.TeamID)
	if scopeType == store.PluginScopeTeam {
		if ok, err := store.UserCanManageTeam(r.Context(), infra.DB, scopeID, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			http.Error(w, "team forbidden", http.StatusForbidden)
			return
		}
	}
	record, err := store.UpsertPluginInstall(r.Context(), infra.DB, store.PluginInstallRecord{
		ScopeType: scopeType,
		ScopeID:   scopeID,
		Server:    server,
		Enabled:   req.Enabled,
		UpdatedBy: userID,
	})
	if err != nil {
		if errors.Is(err, store.ErrValidation) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.PluginInstallResponse{Install: pluginInstallResp(record)})
}

func pluginCatalogResponse(r *http.Request, userID, teamID, scopeType, scopeID string) (model.ListPluginsResponse, error) {
	plugins := []*model.PluginResp{}
	if conf.App == nil {
		return model.ListPluginsResponse{ScopeType: scopeType, ScopeID: scopeID, Plugins: plugins}, nil
	}
	configured := configuredPluginServers()
	enabled, err := store.EnabledPluginServers(r.Context(), infra.DB, scopeType, scopeID, configured)
	if err != nil {
		return model.ListPluginsResponse{}, err
	}
	toolsByServer := map[string][]*model.PluginToolResp{}
	ctx := infra.WithPluginScope(r.Context(), infra.PluginScope{UserID: userID, TeamID: teamID})
	tools, err := infra.ListMCPTools(ctx)
	if err != nil {
		return model.ListPluginsResponse{}, err
	}
	for _, tool := range tools {
		toolsByServer[tool.Server] = append(toolsByServer[tool.Server], &model.PluginToolResp{
			Name:        tool.Name,
			Description: tool.Description,
		})
	}
	for _, server := range configured {
		cfg := conf.App.MCP.Servers[server]
		plugins = append(plugins, &model.PluginResp{
			Server:    server,
			Transport: cfg.Transport(),
			Enabled:   enabled[server],
			Tools:     toolsByServer[server],
		})
	}
	return model.ListPluginsResponse{ScopeType: scopeType, ScopeID: scopeID, Plugins: plugins}, nil
}

func configuredPluginServers() []string {
	if conf.App == nil {
		return nil
	}
	servers := make([]string, 0, len(conf.App.MCP.Servers))
	for server := range conf.App.MCP.Servers {
		servers = append(servers, server)
	}
	sort.Strings(servers)
	return servers
}

func configuredPluginExists(server string) bool {
	if conf.App == nil || strings.TrimSpace(server) == "" {
		return false
	}
	_, ok := conf.App.MCP.Servers[server]
	return ok
}

func pluginInstallResp(record store.PluginInstallRecord) *model.PluginInstallResp {
	resp := &model.PluginInstallResp{
		ScopeType: record.ScopeType,
		ScopeID:   record.ScopeID,
		Server:    record.Server,
		Enabled:   record.Enabled,
		UpdatedBy: record.UpdatedBy,
	}
	if !record.UpdatedAt.IsZero() {
		resp.UpdatedAt = record.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
