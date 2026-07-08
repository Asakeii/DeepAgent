package app

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"deepAgent/conf"
	"deepAgent/internal/security"
)

type imageInputPolicy struct {
	MaxBytes     int64
	AllowedTypes []string
}

func currentImageInputPolicy() imageInputPolicy {
	if conf.App == nil {
		return imageInputPolicy{
			MaxBytes:     conf.DefaultImageMaxBytes,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
		}
	}
	return imageInputPolicy{
		MaxBytes:     conf.App.Server.ImageMaxBytes,
		AllowedTypes: append([]string(nil), conf.App.Server.ImageAllowedTypes...),
	}
}

func validateImageInput(raw string) error {
	return validateImageInputWithPolicy(raw, currentImageInputPolicy())
}

func validateImageInputWithPolicy(raw string, policy imageInputPolicy) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = conf.DefaultImageMaxBytes
	}
	allowed := allowedImageTypes(policy.AllowedTypes)

	if strings.HasPrefix(raw, "data:") {
		mimeType, b64, err := splitImageDataURL(raw)
		if err != nil {
			return err
		}
		if !allowed[mimeType] {
			return fmt.Errorf("unsupported image type %q", mimeType)
		}
		data, err := decodeBoundedBase64(b64, policy.MaxBytes)
		if err != nil {
			return err
		}
		if detected := http.DetectContentType(data); !allowed[detected] && !strings.HasPrefix(detected, mimeType) {
			return fmt.Errorf("unsupported image content type %q", detected)
		}
		return nil
	}

	if isHTTPURL(raw) {
		return security.ValidateExternalURL(raw, security.URLPolicyFromConfig())
	}

	data, err := decodeBoundedBase64(raw, policy.MaxBytes)
	if err != nil {
		return fmt.Errorf("image must be a data URL, raw base64, or HTTP(S) URL: %w", err)
	}
	detected := http.DetectContentType(data)
	if !allowed[detected] {
		return fmt.Errorf("unsupported image content type %q", detected)
	}
	return nil
}

func splitImageDataURL(dataURL string) (string, string, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid image data URL")
	}
	header := parts[0]
	if !strings.Contains(header, ";base64") {
		return "", "", fmt.Errorf("image data URL must be base64 encoded")
	}
	mimeType := strings.TrimPrefix(strings.SplitN(header, ";", 2)[0], "data:")
	if mimeType == "" {
		return "", "", fmt.Errorf("image data URL missing MIME type")
	}
	return strings.ToLower(mimeType), parts[1], nil
}

func decodeBoundedBase64(raw string, maxBytes int64) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	estimated := base64.StdEncoding.DecodedLen(len(raw))
	if int64(estimated) > maxBytes+2 {
		return nil, fmt.Errorf("image too large: estimated %d bytes, max %d", estimated, maxBytes)
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("image too large: %d bytes, max %d", len(data), maxBytes)
	}
	return data, nil
}

func allowedImageTypes(types []string) map[string]bool {
	out := map[string]bool{}
	for _, typ := range types {
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ != "" {
			out[typ] = true
		}
	}
	if len(out) == 0 {
		out["image/jpeg"] = true
		out["image/png"] = true
		out["image/webp"] = true
		out["image/gif"] = true
	}
	return out
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
