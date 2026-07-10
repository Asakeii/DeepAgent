package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"deepAgent/conf"
)

type OIDCVerifier struct {
	verifier         *oidc.IDTokenVerifier
	issuer           string
	userIDClaim      string
	displayNameClaim string
	rolesClaim       string
	adminRoles       map[string]struct{}
}

func NewOIDCVerifier(ctx context.Context, cfg conf.OIDCConfig) (*OIDCVerifier, error) {
	if err := validateOIDCConfig(cfg); err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(ctx, strings.TrimSpace(cfg.IssuerURL))
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}
	return newOIDCVerifier(provider.VerifierContext(ctx, &oidc.Config{ClientID: strings.TrimSpace(cfg.Audience)}), cfg), nil
}

func newOIDCVerifier(verifier *oidc.IDTokenVerifier, cfg conf.OIDCConfig) *OIDCVerifier {
	adminRoles := make(map[string]struct{}, len(cfg.AdminRoles))
	for _, role := range cfg.AdminRoles {
		if role = strings.TrimSpace(role); role != "" {
			adminRoles[role] = struct{}{}
		}
	}
	return &OIDCVerifier{
		verifier:         verifier,
		issuer:           strings.TrimSpace(cfg.IssuerURL),
		userIDClaim:      defaultString(cfg.UserIDClaim, "sub"),
		displayNameClaim: defaultString(cfg.DisplayNameClaim, "name"),
		rolesClaim:       defaultString(cfg.RolesClaim, "groups"),
		adminRoles:       adminRoles,
	}
}

func (v *OIDCVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	if v == nil || v.verifier == nil {
		return Principal{}, fmt.Errorf("oidc verifier is not initialized")
	}
	token, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return Principal{}, err
	}
	var claims map[string]json.RawMessage
	if err := token.Claims(&claims); err != nil {
		return Principal{}, fmt.Errorf("decode oidc claims: %w", err)
	}
	providerSubject := token.Subject
	if v.userIDClaim != "sub" {
		providerSubject = stringClaim(claims[v.userIDClaim])
	}
	providerSubject = strings.TrimSpace(providerSubject)
	if providerSubject == "" {
		return Principal{}, fmt.Errorf("oidc claim %q is required", v.userIDClaim)
	}

	roles := stringListClaim(claims[v.rolesClaim])
	admin := false
	for _, role := range roles {
		if _, ok := v.adminRoles[role]; ok {
			admin = true
			break
		}
	}
	providerID := normalizeID(shortHash(v.issuer)+":"+providerSubject, 128)
	return Principal{
		UserID:      oidcUserID(v.issuer, providerSubject),
		Provider:    "oidc",
		ProviderID:  providerID,
		DisplayName: stringClaim(claims[v.displayNameClaim]),
		Roles:       roles,
		Admin:       admin,
		AuthType:    "oidc",
	}, nil
}

func oidcUserID(issuer, subject string) string {
	prefix := "oidc:" + shortHash(issuer) + ":"
	if len(prefix)+len(subject) <= 128 {
		return prefix + subject
	}
	sum := sha256.Sum256([]byte(subject))
	return prefix + "sha256:" + hex.EncodeToString(sum[:])
}

func stringClaim(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func stringListClaim(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return compactStrings(values)
	}
	var single string
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil
	}
	return compactStrings(strings.Fields(single))
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func defaultString(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}
