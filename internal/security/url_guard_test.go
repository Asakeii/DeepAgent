package security

import (
	"strings"
	"testing"
)

func TestValidateExternalURLRejectsPrivateHosts(t *testing.T) {
	cases := []string{
		"http://localhost/admin",
		"http://127.0.0.1/admin",
		"http://10.0.0.1/admin",
		"http://169.254.169.254/latest/meta-data",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			err := ValidateExternalURL(raw, URLPolicy{})
			if err == nil || !strings.Contains(err.Error(), "private or local") {
				t.Fatalf("err=%v, want private/local rejection", err)
			}
		})
	}
}

func TestValidateExternalURLAllowsPublicHTTPS(t *testing.T) {
	err := ValidateExternalURL("https://example.com/page", URLPolicy{})
	if err != nil {
		t.Fatalf("validate URL: %v", err)
	}
}

func TestValidateExternalURLRejectsUnsupportedScheme(t *testing.T) {
	err := ValidateExternalURL("file:///etc/passwd", URLPolicy{})
	if err == nil || !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Fatalf("err=%v, want unsupported scheme", err)
	}
}

func TestValidateExternalURLHonorsAllowAndDenyHosts(t *testing.T) {
	policy := URLPolicy{
		AllowedHosts: []string{"example.com", "*.trusted.example"},
		DeniedHosts:  []string{"blocked.trusted.example"},
	}
	if err := ValidateExternalURL("https://api.trusted.example/path", policy); err != nil {
		t.Fatalf("trusted subdomain rejected: %v", err)
	}
	if err := ValidateExternalURL("https://example.com/path", policy); err != nil {
		t.Fatalf("exact allow rejected: %v", err)
	}
	err := ValidateExternalURL("https://evil.example/path", policy)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("err=%v, want not allowed", err)
	}
	err = ValidateExternalURL("https://blocked.trusted.example/path", policy)
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err=%v, want denied", err)
	}
}

func TestValidateExternalURLCanAllowPrivateNetworks(t *testing.T) {
	err := ValidateExternalURL("http://127.0.0.1:8080/health", URLPolicy{AllowPrivateNetworks: true})
	if err != nil {
		t.Fatalf("private URL rejected: %v", err)
	}
}
