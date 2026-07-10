package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"deepAgent/conf"
)

func TestOIDCVerifierValidatesAndMapsClaims(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "https://issuer.example"
	audience := "deepagent-api"
	baseVerifier := oidc.NewVerifier(issuer, &oidc.StaticKeySet{
		PublicKeys: []crypto.PublicKey{&privateKey.PublicKey},
	}, &oidc.Config{ClientID: audience})
	verifier := newOIDCVerifier(baseVerifier, conf.OIDCConfig{
		IssuerURL:        issuer,
		Audience:         audience,
		AdminRoles:       []string{"deepagent-admin"},
		RolesClaim:       "roles",
		DisplayNameClaim: "preferred_username",
	})
	token := signRS256Token(t, privateKey, map[string]any{
		"iss":                issuer,
		"aud":                audience,
		"sub":                "user-42",
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Add(-time.Minute).Unix(),
		"preferred_username": "Ada",
		"roles":              []string{"reader", "deepagent-admin"},
	})

	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(principal.UserID, ":user-42") || principal.DisplayName != "Ada" || !principal.Admin {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if len(principal.Roles) != 2 || principal.Provider != "oidc" || principal.AuthType != "oidc" {
		t.Fatalf("unexpected mapped claims: %+v", principal)
	}
}

func TestOIDCVerifierRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "https://issuer.example"
	baseVerifier := oidc.NewVerifier(issuer, &oidc.StaticKeySet{
		PublicKeys: []crypto.PublicKey{&privateKey.PublicKey},
	}, &oidc.Config{ClientID: "deepagent-api"})
	verifier := newOIDCVerifier(baseVerifier, conf.OIDCConfig{IssuerURL: issuer})
	token := signRS256Token(t, privateKey, map[string]any{
		"iss": issuer,
		"aud": "another-api",
		"sub": "user-42",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), token); err == nil {
		t.Fatal("expected audience validation failure")
	}
}

func TestNewOIDCVerifierUsesDiscoveryAndJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const audience = "deepagent-api"
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                                issuer,
				"jwks_uri":                              issuer + "/keys",
				"authorization_endpoint":                issuer + "/authorize",
				"token_endpoint":                        issuer + "/token",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
				"kty": "RSA",
				"kid": "test-key",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL

	verifier, err := NewOIDCVerifier(context.Background(), conf.OIDCConfig{
		IssuerURL: issuer,
		Audience:  audience,
	})
	if err != nil {
		t.Fatal(err)
	}
	token := signRS256Token(t, privateKey, map[string]any{
		"iss": issuer,
		"aud": audience,
		"sub": "discovered-user",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(principal.UserID, ":discovered-user") {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func signRS256Token(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	encoding := base64.RawURLEncoding
	unsigned := encoding.EncodeToString(header) + "." + encoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + encoding.EncodeToString(signature)
}
