package handler

import (
	"net/http"
	"strings"

	"deepAgent/internal/store"
)

func requestUserID(r *http.Request) string {
	userID := strings.TrimSpace(r.Header.Get("X-DeepAgent-User"))
	if userID == "" {
		userID = strings.TrimSpace(r.URL.Query().Get("user_id"))
	}
	return store.NormalizeUserID(userID)
}
