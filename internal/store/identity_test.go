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
