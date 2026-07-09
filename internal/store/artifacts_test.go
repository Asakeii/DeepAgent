package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	got, err := GetArtifact(ctx, db, records[0].ID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != records[0].ID || got.UserID != userID {
		t.Fatalf("unexpected artifact lookup: %+v", got)
	}
	other, err := GetArtifact(ctx, db, records[0].ID, "other-user-"+randomSuffix())
	if err != nil {
		t.Fatal(err)
	}
	if other.ID != 0 {
		t.Fatalf("artifact should not be visible to another user: %+v", other)
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

func TestArtifactShareLifecycle(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "artifact-share-user-" + randomSuffix()
	threadID := "artifact-share-thread-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM artifact_shares WHERE user_id=?", userID)
		_, _ = db.ExecContext(ctx, "DELETE FROM artifacts WHERE thread_id=?", threadID)
	})

	artifactID, err := CreateArtifact(ctx, db, ArtifactRecord{
		UserID:   userID,
		ThreadID: threadID,
		Kind:     ArtifactKindReport,
		Title:    "可分享报告",
		Content:  "# 可分享报告",
	})
	if err != nil {
		t.Fatal(err)
	}
	share, err := CreateArtifactShare(ctx, db, artifactID, userID, sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true})
	if err != nil {
		t.Fatal(err)
	}
	if share.Token == "" || !strings.HasPrefix(share.Token, "as_") {
		t.Fatalf("unexpected token: %+v", share)
	}
	if share.TokenHash == share.Token {
		t.Fatal("share token should be stored as a hash")
	}

	artifact, loadedShare, ok, err := GetSharedArtifact(ctx, db, share.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("share should be readable")
	}
	if artifact.ID != artifactID || artifact.Content != "# 可分享报告" {
		t.Fatalf("unexpected shared artifact: %+v", artifact)
	}
	if loadedShare.ArtifactID != artifactID || loadedShare.UserID != userID {
		t.Fatalf("unexpected loaded share: %+v", loadedShare)
	}

	revoked, err := RevokeArtifactShare(ctx, db, share.Token, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("share should be revoked")
	}
	_, _, ok, err = GetSharedArtifact(ctx, db, share.Token)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("revoked share should not be readable")
	}
}
