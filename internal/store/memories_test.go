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

func TestNormalizeMemoryKind(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: MemoryKindPreference},
		{in: "偏好", want: MemoryKindPreference},
		{in: "目标", want: MemoryKindGoal},
		{in: "profile", want: MemoryKindFact},
		{in: "经历", want: MemoryKindEpisodic},
		{in: "业务", want: MemoryKindBusiness},
	}
	for _, tc := range cases {
		if got := NormalizeMemoryKind(tc.in); got != tc.want {
			t.Fatalf("NormalizeMemoryKind(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestListMemoriesOrdersByLayer(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	userID := "memory-layer-user-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM memories WHERE user_id=?", userID)
	})
	for _, record := range []MemoryRecord{
		{UserID: userID, Kind: MemoryKindEpisodic, Content: "上次讨论过边界", Importance: 9},
		{UserID: userID, Kind: MemoryKindGoal, Content: "目标是每周跑步三次", Importance: 1},
		{UserID: userID, Kind: MemoryKindPreference, Content: "喜欢早上提醒", Importance: 1},
	} {
		if _, err := CreateMemory(ctx, db, record); err != nil {
			t.Fatal(err)
		}
	}

	memories, err := ListMemories(ctx, db, userID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(memories) != 3 {
		t.Fatalf("len=%d, want 3", len(memories))
	}
	if memories[0].Kind != MemoryKindPreference || memories[1].Kind != MemoryKindGoal || memories[2].Kind != MemoryKindEpisodic {
		t.Fatalf("unexpected order: %+v", memories)
	}
}
