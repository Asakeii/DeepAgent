package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCreateAndListArtifacts(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "artifact-user-" + randomSuffix()
	threadID := "artifact-thread-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM artifacts WHERE thread_id=?", threadID)
	})

	metadata, _ := json.Marshal(map[string]string{"source": "test"})
	id, err := CreateArtifact(ctx, db, ArtifactRecord{
		UserID:   userID,
		ThreadID: threadID,
		RunID:    "run-1",
		Kind:     ArtifactKindReport,
		Title:    "研究报告",
		Content:  "# 报告",
		Metadata: metadata,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("id=%d, want positive", id)
	}
	if _, err := CreateArtifact(ctx, db, ArtifactRecord{
		UserID:   userID,
		ThreadID: threadID,
		RunID:    "run-2",
		Kind:     ArtifactKindReport,
		Title:    "研究报告",
		Content:  "# 报告 v2",
	}); err != nil {
		t.Fatal(err)
	}

	records, err := ListArtifacts(ctx, db, userID, threadID, ArtifactKindReport, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("len=%d, want 2", len(records))
	}
	if records[0].Version != 2 || records[1].Version != 1 {
		t.Fatalf("versions=%d,%d want 2,1", records[0].Version, records[1].Version)
	}
	if records[0].Format != ArtifactFormatMD || records[0].Source != ArtifactSourceAgent {
		t.Fatalf("unexpected defaults: %+v", records[0])
	}
}

func TestCreateArtifactRejectsInvalidMetadata(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	_, err := CreateArtifact(context.Background(), db, ArtifactRecord{
		UserID:   "artifact-user-" + randomSuffix(),
		ThreadID: "artifact-thread-" + randomSuffix(),
		Content:  "content",
		Metadata: []byte(`not-json`),
	})
	if err == nil {
		t.Fatal("expected invalid metadata error")
	}
}
