package app

import (
	"encoding/base64"
	"strings"
	"testing"

	"deepAgent/conf"
)

const tinyPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/lD1V9wAAAABJRU5ErkJggg=="

func TestValidateImageInputAllowsDataURL(t *testing.T) {
	err := validateImageInputWithPolicy(tinyPNGDataURL, imageInputPolicy{
		MaxBytes:     1024,
		AllowedTypes: []string{"image/png"},
	})
	if err != nil {
		t.Fatalf("validate image: %v", err)
	}
}

func TestValidateImageInputRejectsUnsupportedType(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("not an image"))
	err := validateImageInputWithPolicy(raw, imageInputPolicy{
		MaxBytes:     1024,
		AllowedTypes: []string{"image/png"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported image content type") {
		t.Fatalf("err=%v, want unsupported image type", err)
	}
}

func TestValidateImageInputRejectsLargeImage(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString(make([]byte, 16))
	err := validateImageInputWithPolicy(raw, imageInputPolicy{
		MaxBytes:     4,
		AllowedTypes: []string{"image/png"},
	})
	if err == nil || !strings.Contains(err.Error(), "image too large") {
		t.Fatalf("err=%v, want image too large", err)
	}
}

func TestValidateImageInputRejectsLocalPath(t *testing.T) {
	err := validateImageInputWithPolicy("/etc/passwd", imageInputPolicy{
		MaxBytes:     1024,
		AllowedTypes: []string{"image/png"},
	})
	if err == nil || !strings.Contains(err.Error(), "image must be") {
		t.Fatalf("err=%v, want local path rejected", err)
	}
}

func TestValidateImageInputAllowsHTTPURL(t *testing.T) {
	prev := conf.App
	conf.App = &conf.Config{}
	t.Cleanup(func() { conf.App = prev })
	err := validateImageInputWithPolicy("https://example.com/a.png", imageInputPolicy{
		MaxBytes:     1024,
		AllowedTypes: []string{"image/png"},
	})
	if err != nil {
		t.Fatalf("validate image URL: %v", err)
	}
}

func TestValidateImageInputRejectsPrivateHTTPURL(t *testing.T) {
	prev := conf.App
	conf.App = &conf.Config{}
	t.Cleanup(func() { conf.App = prev })
	err := validateImageInputWithPolicy("http://127.0.0.1/a.png", imageInputPolicy{
		MaxBytes:     1024,
		AllowedTypes: []string{"image/png"},
	})
	if err == nil || !strings.Contains(err.Error(), "private or local") {
		t.Fatalf("err=%v, want private URL rejection", err)
	}
}
