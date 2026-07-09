package infra

import (
	"context"
	"testing"
)

func TestModelBundleForUsesContextProfile(t *testing.T) {
	prevDefault := defaultModels
	prevProfiles := modelProfiles
	t.Cleanup(func() {
		defaultModels = prevDefault
		modelProfiles = prevProfiles
	})
	defaultBundle := &ModelBundle{Profile: ""}
	fastBundle := &ModelBundle{Profile: "fast"}
	defaultModels = defaultBundle
	modelProfiles = map[string]*ModelBundle{"fast": fastBundle}

	if got := modelBundleFor(context.Background()); got != defaultBundle {
		t.Fatalf("default bundle=%p, want %p", got, defaultBundle)
	}
	ctx := WithModelProfile(context.Background(), " FAST ")
	if got := ActiveModelProfile(ctx); got != "fast" {
		t.Fatalf("active profile=%q, want fast", got)
	}
	if got := modelBundleFor(ctx); got != fastBundle {
		t.Fatalf("profile bundle=%p, want %p", got, fastBundle)
	}
	if !HasModelProfile("fast") || HasModelProfile("missing") {
		t.Fatalf("unexpected profile availability")
	}
}
