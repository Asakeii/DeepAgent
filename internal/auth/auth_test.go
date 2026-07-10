package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"deepAgent/conf"
)

type fakeTokenVerifier struct {
	principal Principal
	err       error
	token     string
}

func (f *fakeTokenVerifier) Verify(_ context.Context, token string) (Principal, error) {
	f.token = token
	return f.principal, f.err
}

func TestAuthenticatorUsesNamedAPIKeyPrincipal(t *testing.T) {
	authenticator, err := NewAuthenticatorWithVerifier(conf.ServerConfig{
		APIKeyPrincipals: []conf.APIKeyPrincipalConfig{{
			Key:         "machine-secret",
			UserID:      "service:reporter",
			DisplayName: "Report worker",
			Admin:       true,
		}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("X-DeepAgent-API-Key", "machine-secret")
	req.Header.Set("X-DeepAgent-User", "spoofed-user")

	principal, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != "service:reporter" || !principal.Admin || principal.AuthType != "api_key" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestAuthenticatorFallsBackToOIDCVerifier(t *testing.T) {
	verifier := &fakeTokenVerifier{principal: Principal{UserID: "oidc:user-1", AuthType: "oidc"}}
	authenticator, err := NewAuthenticatorWithVerifier(conf.ServerConfig{
		OIDC: conf.OIDCConfig{IssuerURL: "https://issuer.example", Audience: "deepagent"},
	}, verifier)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer signed-token")

	principal, err := authenticator.Authenticate(req)
	if err != nil {
		t.Fatal(err)
	}
	if verifier.token != "signed-token" || principal.UserID != "oidc:user-1" {
		t.Fatalf("token=%q principal=%+v", verifier.token, principal)
	}
}

func TestAuthenticatorDoesNotTreatExplicitAPIKeyAsOIDCToken(t *testing.T) {
	verifier := &fakeTokenVerifier{err: errors.New("should not be called")}
	authenticator, err := NewAuthenticatorWithVerifier(conf.ServerConfig{
		OIDC: conf.OIDCConfig{IssuerURL: "https://issuer.example", Audience: "deepagent"},
	}, verifier)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("X-DeepAgent-API-Key", "unknown")

	_, err = authenticator.Authenticate(req)
	if !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("error=%v, want invalid credential", err)
	}
	if verifier.token != "" {
		t.Fatalf("oidc verifier unexpectedly received %q", verifier.token)
	}
}

func TestAuthenticatorRejectsPartialOIDCConfig(t *testing.T) {
	_, err := NewAuthenticatorWithVerifier(conf.ServerConfig{
		OIDC: conf.OIDCConfig{IssuerURL: "https://issuer.example"},
	}, nil)
	if err == nil {
		t.Fatal("expected invalid oidc config")
	}
}

func TestAuthenticatorRejectsDuplicateNamedAPIKeys(t *testing.T) {
	_, err := NewAuthenticatorWithVerifier(conf.ServerConfig{
		APIKeyPrincipals: []conf.APIKeyPrincipalConfig{
			{Key: "duplicate", UserID: "service:first"},
			{Key: "duplicate", UserID: "service:second"},
		},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate api key configuration error")
	}
}
