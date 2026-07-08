package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deepAgent/conf"
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

func withTestServerConfig(t *testing.T, server conf.ServerConfig) {
	t.Helper()
	prev := conf.App
	conf.App = &conf.Config{Server: server}
	t.Cleanup(func() {
		conf.App = prev
	})
}
