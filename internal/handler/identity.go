package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"deepAgent/internal/auth"
	"deepAgent/internal/store"
)

func requestUserID(r *http.Request) string {
	if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
		return store.NormalizeUserID(principal.UserID)
	}
	userID := strings.TrimSpace(r.Header.Get("X-DeepAgent-User"))
	if userID == "" {
		userID = strings.TrimSpace(r.URL.Query().Get("user_id"))
	}
	return store.NormalizeUserID(userID)
}

func Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		principal = auth.Principal{
			UserID:     requestUserID(r),
			Provider:   "local",
			ProviderID: requestUserID(r),
			AuthType:   "development",
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(principal)
}
