package store

import (
	"context"
	"errors"
	"testing"
)

func TestTeamThreadAccess(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureIdentityTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureMessageTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	ownerID := "team-owner-" + randomSuffix()
	memberID := "team-member-" + randomSuffix()
	otherID := "team-other-" + randomSuffix()
	threadID := "team-thread-" + randomSuffix()

	team, err := CreateTeam(ctx, db, ownerID, "Research Team")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(ctx, "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(ctx, "DELETE FROM team_members WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM teams WHERE id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id IN (?, ?, ?)", ownerID, memberID, otherID)
	})
	if team.ID == "" || team.Role != TeamRoleOwner {
		t.Fatalf("unexpected team: %+v", team)
	}
	if _, err := AddTeamMember(ctx, db, team.ID, ownerID, memberID, TeamRoleMember); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThreadWithTeam(ctx, db, threadID, ownerID, team.ID, "shared thread", "test"); err != nil {
		t.Fatal(err)
	}
	if err := AppendMessage(ctx, db, threadID, "user", "shared thread"); err != nil {
		t.Fatal(err)
	}

	ok, err := ThreadBelongsToUser(ctx, db, threadID, memberID)
	if err != nil || !ok {
		t.Fatalf("member thread access = %v err=%v, want true nil", ok, err)
	}
	ok, err = ThreadBelongsToUser(ctx, db, threadID, otherID)
	if err != nil || ok {
		t.Fatalf("other thread access = %v err=%v, want false nil", ok, err)
	}

	threads, err := SearchThreadsForUser(ctx, db, memberID, "shared", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 1 || threads[0].ThreadID != threadID || threads[0].TeamID != team.ID {
		t.Fatalf("member threads = %+v, want team thread", threads)
	}
}

func TestAddTeamMemberRequiresManager(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureIdentityTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	ownerID := "team-owner-" + randomSuffix()
	memberID := "team-member-" + randomSuffix()
	thirdID := "team-third-" + randomSuffix()

	team, err := CreateTeam(ctx, db, ownerID, "Ops")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM team_members WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM teams WHERE id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id IN (?, ?, ?)", ownerID, memberID, thirdID)
	})
	if _, err := AddTeamMember(ctx, db, team.ID, ownerID, memberID, TeamRoleMember); err != nil {
		t.Fatal(err)
	}
	if _, err := AddTeamMember(ctx, db, team.ID, memberID, thirdID, TeamRoleMember); !errors.Is(err, ErrTeamForbidden) {
		t.Fatalf("member add err=%v, want ErrTeamForbidden", err)
	}
}

func TestTeamArtifactAccess(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsureIdentityTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := EnsureArtifactTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	ownerID := "artifact-owner-" + randomSuffix()
	memberID := "artifact-member-" + randomSuffix()
	otherID := "artifact-other-" + randomSuffix()
	threadID := "artifact-team-thread-" + randomSuffix()

	team, err := CreateTeam(ctx, db, ownerID, "Artifact Team")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM artifacts WHERE thread_id=?", threadID)
		_, _ = db.ExecContext(ctx, "DELETE FROM threads WHERE id=?", threadID)
		_, _ = db.ExecContext(ctx, "DELETE FROM team_members WHERE team_id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM teams WHERE id=?", team.ID)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id IN (?, ?, ?)", ownerID, memberID, otherID)
	})
	if _, err := AddTeamMember(ctx, db, team.ID, ownerID, memberID, TeamRoleMember); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThreadWithTeam(ctx, db, threadID, ownerID, team.ID, "artifact thread", "test"); err != nil {
		t.Fatal(err)
	}
	artifactID, err := CreateArtifact(ctx, db, ArtifactRecord{
		UserID:   ownerID,
		ThreadID: threadID,
		Kind:     ArtifactKindReport,
		Title:    "Shared Report",
		Content:  "team report",
	})
	if err != nil {
		t.Fatal(err)
	}

	record, err := GetArtifact(ctx, db, artifactID, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != artifactID {
		t.Fatalf("member artifact id=%d, want %d", record.ID, artifactID)
	}
	record, err = GetArtifact(ctx, db, artifactID, otherID)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != 0 {
		t.Fatalf("other artifact = %+v, want empty", record)
	}
}
