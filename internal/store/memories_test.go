package store

import (
	"context"
	"testing"
)

func TestMemoriesLifecycleWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	userID := "memory-user-" + randomSuffix()
	id, err := CreateMemory(ctx, db, MemoryRecord{
		UserID:     userID,
		ThreadID:   "memory-thread",
		Kind:       MemoryKindPreference,
		Content:    "我喜欢早上收到提醒",
		Importance: 5,
		Source:     "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("id=%d, want positive", id)
	}

	memories, err := ListMemories(ctx, db, userID, MemoryKindPreference, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(memories) != 1 || memories[0].Content != "我喜欢早上收到提醒" {
		t.Fatalf("unexpected memories: %+v", memories)
	}

	_, _ = db.ExecContext(ctx, "DELETE FROM memories WHERE user_id=?", userID)
}
