package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"deepAgent/conf"
)

var (
	ErrMissingCredential = errors.New("authentication credential is required")
	ErrInvalidCredential = errors.New("authentication credential is invalid")
)

type Principal struct {
	UserID      string   `json:"user_id"`
	Provider    string   `json:"provider"`
	ProviderID  string   `json:"provider_id"`
	DisplayName string   `json:"display_name,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Admin       bool     `json:"admin"`
	AuthType    string   `json:"auth_type"`
}

type TokenVerifier interface {
	Verify(context.Context, string) (Principal, error)
}

type apiKeyCredential struct {
	key       string
	principal Principal
}

type Authenticator struct {
	apiKeys       []apiKeyCredential
	tokenVerifier TokenVerifier
}

func NewAuthenticator(ctx context.Context, cfg conf.ServerConfig) (*Authenticator, error) {
	var verifier TokenVerifier
	if oidcConfigured(cfg.OIDC) {
		timeoutSeconds := cfg.OIDC.DiscoveryTimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 10
		}
		discoveryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
		var err error
		verifier, err = NewOIDCVerifier(discoveryCtx, cfg.OIDC)
		if err != nil {
			return nil, err
		}
	}
	return NewAuthenticatorWithVerifier(cfg, verifier)
}

func NewAuthenticatorWithVerifier(cfg conf.ServerConfig, verifier TokenVerifier) (*Authenticator, error) {
	if err := validateOIDCConfig(cfg.OIDC); err != nil {
		return nil, err
	}
	if oidcConfigured(cfg.OIDC) && verifier == nil {
		return nil, fmt.Errorf("oidc verifier is required when oidc is configured")
	}

	a := &Authenticator{tokenVerifier: verifier}
	for _, configured := range cfg.APIKeyPrincipals {
		key := strings.TrimSpace(configured.Key)
		if key == "" {
			return nil, fmt.Errorf("api_key_principals key is required")
		}
		userID := strings.TrimSpace(configured.UserID)
		if userID == "" {
			return nil, fmt.Errorf("api_key_principals user_id is required")
		}
		if a.hasAPIKey(key) {
			return nil, fmt.Errorf("api_key_principals contains a duplicate key")
		}
		a.apiKeys = append(a.apiKeys, apiKeyCredential{
			key: key,
			principal: Principal{
				UserID:      normalizeID(userID, 128),
				Provider:    "api_key",
				ProviderID:  providerIDForAPIKey(key),
				DisplayName: strings.TrimSpace(configured.DisplayName),
				Admin:       configured.Admin,
				AuthType:    "api_key",
			},
		})
	}
	for _, key := range cfg.APIKeys {
		a.addLegacyAPIKey(key, false)
	}
	for _, key := range cfg.AdminAPIKeys {
		a.addLegacyAPIKey(key, true)
	}
	return a, nil
}

func (a *Authenticator) hasAPIKey(key string) bool {
	for i := range a.apiKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(a.apiKeys[i].key)) == 1 {
			return true
		}
	}
	return false
}

func (a *Authenticator) Enabled() bool {
	return a != nil && (len(a.apiKeys) > 0 || a.tokenVerifier != nil)
}

func (a *Authenticator) Authenticate(r *http.Request) (Principal, error) {
	if a == nil || !a.Enabled() {
		return Principal{}, ErrMissingCredential
	}
	credential, explicitAPIKey := requestCredential(r)
	if credential == "" {
		return Principal{}, ErrMissingCredential
	}
	for _, configured := range a.apiKeys {
		if subtle.ConstantTimeCompare([]byte(credential), []byte(configured.key)) == 1 {
			return configured.principal, nil
		}
	}
	if explicitAPIKey || a.tokenVerifier == nil {
		return Principal{}, ErrInvalidCredential
	}
	principal, err := a.tokenVerifier.Verify(r.Context(), credential)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %v", ErrInvalidCredential, err)
	}
	return principal, nil
}

func (a *Authenticator) addLegacyAPIKey(raw string, admin bool) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return
	}
	for i := range a.apiKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(a.apiKeys[i].key)) == 1 {
			a.apiKeys[i].principal.Admin = a.apiKeys[i].principal.Admin || admin
			return
		}
	}
	hash := shortHash(key)
	a.apiKeys = append(a.apiKeys, apiKeyCredential{
		key: key,
		principal: Principal{
			UserID:     "service:" + hash,
			Provider:   "api_key",
			ProviderID: providerIDForAPIKey(key),
			Admin:      admin,
			AuthType:   "api_key",
		},
	})
}

func requestCredential(r *http.Request) (credential string, explicitAPIKey bool) {
	if key := strings.TrimSpace(r.Header.Get("X-DeepAgent-API-Key")); key != "" {
		return key, true
	}
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return "", false
	}
	scheme, token, ok := strings.Cut(raw, " ")
	if !ok || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return "", false
	}
	return strings.TrimSpace(token), false
}

func oidcConfigured(cfg conf.OIDCConfig) bool {
	return strings.TrimSpace(cfg.IssuerURL) != "" || strings.TrimSpace(cfg.Audience) != ""
}

func validateOIDCConfig(cfg conf.OIDCConfig) error {
	if !oidcConfigured(cfg) {
		return nil
	}
	if strings.TrimSpace(cfg.IssuerURL) == "" {
		return fmt.Errorf("server.oidc.issuer_url is required")
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return fmt.Errorf("server.oidc.audience is required")
	}
	return nil
}

func providerIDForAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

func normalizeID(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok && principal.UserID != ""
}
