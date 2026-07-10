package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"deepAgent/internal/auth"
)

func TestRequestUserIDPrefersAuthenticatedPrincipal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/me?user_id=query-spoof", nil)
	req.Header.Set("X-DeepAgent-User", "header-spoof")
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "trusted-user"}))

	if got := requestUserID(req); got != "trusted-user" {
		t.Fatalf("user id=%q, want trusted-user", got)
	}
}
