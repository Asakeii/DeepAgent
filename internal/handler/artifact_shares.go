package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"deepAgent/internal/infra"
	"deepAgent/internal/model"
	"deepAgent/internal/store"
)

const maxArtifactShareTTLHours = 24 * 365

func ArtifactShares(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createArtifactShare(w, r)
	case http.MethodDelete:
		revokeArtifactShare(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func createArtifactShare(w http.ResponseWriter, r *http.Request) {
	var req model.CreateArtifactShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ArtifactID <= 0 {
		http.Error(w, "artifact_id required", http.StatusBadRequest)
		return
	}
	expiresAt := artifactShareExpiresAt(req.ExpiresInHours)
	share, err := store.CreateArtifactShare(r.Context(), infra.DB, req.ArtifactID, requestUserID(r), expiresAt)
	if err != nil {
		if errors.Is(err, store.ErrArtifactShareForbidden) {
			http.Error(w, "artifact forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.ArtifactShareResponse{
		Share: artifactShareResp(share, artifactShareURL(r, share.Token)),
	})
}

func revokeArtifactShare(w http.ResponseWriter, r *http.Request) {
	var req model.RevokeArtifactShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	ok, err := store.RevokeArtifactShare(r.Context(), infra.DB, req.Token, requestUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "share not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"revoked": true})
}

func SharedArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	artifact, share, ok, err := store.GetSharedArtifact(r.Context(), infra.DB, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "share not found", http.StatusNotFound)
		return
	}
	resp := artifactResp(artifact)
	resp.UserID = ""
	resp.ThreadID = ""
	resp.RunID = ""
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.SharedArtifactResponse{
		Artifact: resp,
		Share:    artifactShareResp(share, ""),
	})
}

func artifactShareExpiresAt(hours int) sql.NullTime {
	if hours <= 0 {
		return sql.NullTime{}
	}
	if hours > maxArtifactShareTTLHours {
		hours = maxArtifactShareTTLHours
	}
	return sql.NullTime{Time: time.Now().Add(time.Duration(hours) * time.Hour), Valid: true}
}

func artifactShareResp(record store.ArtifactShareRecord, shareURL string) *model.ArtifactShareResp {
	resp := &model.ArtifactShareResp{
		Token:      record.Token,
		ArtifactID: record.ArtifactID,
		ShareURL:   shareURL,
	}
	if record.CreatedAt.Valid {
		resp.CreatedAt = record.CreatedAt.Time.Format(time.RFC3339)
	}
	if record.ExpiresAt.Valid {
		resp.ExpiresAt = record.ExpiresAt.Time.Format(time.RFC3339)
	}
	if record.RevokedAt.Valid {
		resp.RevokedAt = record.RevokedAt.Time.Format(time.RFC3339)
	}
	return resp
}

func artifactShareURL(r *http.Request, token string) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return "/share/artifacts?token=" + url.QueryEscape(token)
	}
	return scheme + "://" + host + "/share/artifacts?token=" + url.QueryEscape(token)
}
