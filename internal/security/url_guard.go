package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"deepAgent/conf"
)

type URLPolicy struct {
	AllowedHosts         []string
	DeniedHosts          []string
	AllowPrivateNetworks bool
}

func URLPolicyFromConfig() URLPolicy {
	if conf.App == nil {
		return URLPolicy{}
	}
	return URLPolicy{
		AllowedHosts:         append([]string(nil), conf.App.Server.URLAllowedHosts...),
		DeniedHosts:          append([]string(nil), conf.App.Server.URLDeniedHosts...),
		AllowPrivateNetworks: conf.App.Server.URLAllowPrivateNetworks,
	}
}

func ValidateExternalURL(raw string, policy URLPolicy) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return fmt.Errorf("URL host is required")
	}
	if hostMatchesAny(host, policy.DeniedHosts) {
		return fmt.Errorf("URL host %q is denied", host)
	}
	if len(normalizeHostPatterns(policy.AllowedHosts)) > 0 && !hostMatchesAny(host, policy.AllowedHosts) {
		return fmt.Errorf("URL host %q is not allowed", host)
	}
	if !policy.AllowPrivateNetworks && isPrivateHost(host) {
		return fmt.Errorf("URL host %q resolves to a private or local network", host)
	}
	return nil
}

func hostMatchesAny(host string, patterns []string) bool {
	for _, pattern := range normalizeHostPatterns(patterns) {
		if pattern == "*" {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(suffix, ".") {
				return true
			}
			continue
		}
		if host == pattern {
			return true
		}
	}
	return false
}

func normalizeHostPatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern != "" {
			out = append(out, pattern)
		}
	}
	return out
}

func isPrivateHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
