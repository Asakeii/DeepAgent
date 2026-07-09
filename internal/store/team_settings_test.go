package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestTeamSettingsLifecycleWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	teamID := "settings-team-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM team_settings WHERE team_id=?", teamID)
	})

	defaults, err := GetTeamSettings(ctx, db, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.TeamID != teamID || defaults.DailyCostBudgetMicros.Valid {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if err := UpsertTeamSettings(ctx, db, TeamSettingsRecord{
		TeamID:                teamID,
		DailyCostBudgetMicros: sql.NullInt64{Int64: 1234567, Valid: true},
		UpdatedBy:             "owner",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := GetTeamSettings(ctx, db, teamID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.DailyCostBudgetMicros.Valid || got.DailyCostBudgetMicros.Int64 != 1234567 || got.UpdatedBy != "owner" {
		t.Fatalf("unexpected team settings: %+v", got)
	}
}
