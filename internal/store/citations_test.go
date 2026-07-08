package store

import (
	"context"
	"testing"
)

func TestCreateAndListArtifactCitations(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "citation-user-" + randomSuffix()
	threadID := "citation-thread-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM artifact_citations WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(ctx, "DELETE FROM artifacts WHERE thread_id=?", threadID)
	})

	artifactID, err := CreateArtifact(ctx, db, ArtifactRecord{
		UserID:   userID,
		ThreadID: threadID,
		RunID:    "run-1",
		Content:  "report",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateArtifactCitations(ctx, db, []CitationRecord{
		{ArtifactID: artifactID, UserID: userID, ThreadID: threadID, RunID: "run-1", Title: "B", URL: "https://b.example", Position: 2},
		{ArtifactID: artifactID, UserID: userID, ThreadID: threadID, RunID: "run-1", Title: "A", URL: "https://a.example", Position: 1},
	}); err != nil {
		t.Fatal(err)
	}

	records, err := ListArtifactCitations(ctx, db, userID, artifactID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("len=%d, want 2", len(records))
	}
	if records[0].Title != "A" || records[1].Title != "B" {
		t.Fatalf("unexpected order: %+v", records)
	}
	ok, err := ArtifactBelongsToUser(ctx, db, artifactID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("artifact should belong to user")
	}
}
