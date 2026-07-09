package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deepAgent/conf"
	"deepAgent/internal/infra"
)

func TestHTTPGuardsAllowConfiguredOrigin(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
	})
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow-origin=%q", got)
	}
}

func TestHTTPGuardsRejectForbiddenOrigin(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
	})
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHTTPGuardsLimitRequestBody(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   4,
	})
	req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader("12345"))
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHTTPGuardsRequireAPIKeyForProtectedRoutes(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHTTPGuardsAllowBearerAPIKey(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPGuardsRequireAdminAPIKeyForAdminRoutes(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"user-secret"},
		AdminAPIKeys:   []string{"admin-secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.Header.Set("Authorization", "Bearer user-secret")
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHTTPGuardsAllowAdminAPIKey(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"user-secret"},
		AdminAPIKeys:   []string{"admin-secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.Header.Set("Authorization", "Bearer admin-secret")
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPGuardsDoNotProtectStaticRoutes(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPGuardsDoNotProtectPublicShareRoutes(t *testing.T) {
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins: []string{"https://app.example.com"},
		MaxBodyBytes:   1024,
		APIKeys:        []string{"secret"},
	})
	req := httptest.NewRequest(http.MethodGet, "/share/artifacts?token=as_test", nil)
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHTTPGuardsAllowWhenRateLimitConfiguredWithoutRedis(t *testing.T) {
	prevRDB := infra.RDB
	infra.RDB = nil
	t.Cleanup(func() {
		infra.RDB = prevRDB
	})
	withTestServerConfig(t, conf.ServerConfig{
		AllowedOrigins:     []string{"https://app.example.com"},
		MaxBodyBytes:       1024,
		RateLimitPerMinute: 1,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rec := httptest.NewRecorder()

	withHTTPGuards(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
}

func withTestServerConfig(t *testing.T, server conf.ServerConfig) {
	t.Helper()
	prev := conf.App
	conf.App = &conf.Config{Server: server}
	t.Cleanup(func() {
		conf.App = prev
	})
}
