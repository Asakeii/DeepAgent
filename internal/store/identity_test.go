package store

import (
	"context"
	"testing"
)

func TestIdentityThreadOwnership(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureIdentityTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	threadID := "thread-owner-" + randomSuffix()
	userID := "user-owner-" + randomSuffix()

	if err := EnsureThread(ctx, db, threadID, userID, "hello", "test"); err != nil {
		t.Fatal(err)
	}
	ok, err := ThreadBelongsToUser(ctx, db, threadID, userID)
	if err != nil || !ok {
		t.Fatalf("ThreadBelongsToUser owner = %v err=%v, want true nil", ok, err)
	}
	ok, err = ThreadBelongsToUser(ctx, db, threadID, "other-user")
	if err != nil || ok {
		t.Fatalf("ThreadBelongsToUser other = %v err=%v, want false nil", ok, err)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM threads WHERE id=?", threadID)
	_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID)
}

func TestEnsureUserPreservesAuthenticatedProvider(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureIdentityTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	userID := "oidc-user-" + randomSuffix()
	defer db.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID)

	if err := EnsureUserProfile(ctx, db, userID, "oidc", "issuer:user-42", "Ada"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureUser(ctx, db, userID, "chat", userID); err != nil {
		t.Fatal(err)
	}

	var provider, providerID, displayName string
	if err := db.QueryRowContext(ctx,
		"SELECT provider, provider_id, display_name FROM users WHERE id=?", userID,
	).Scan(&provider, &providerID, &displayName); err != nil {
		t.Fatal(err)
	}
	if provider != "oidc" || providerID != "issuer:user-42" || displayName != "Ada" {
		t.Fatalf("provider=%q provider_id=%q display_name=%q", provider, providerID, displayName)
	}
}
