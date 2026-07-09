package store

import (
	"context"
	"database/sql"
	"testing"
)

func TestUserSettingsLifecycleWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	userID := "settings-user-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM user_settings WHERE user_id=?", userID)
		_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id=?", userID)
	})

	defaults, err := GetUserSettings(ctx, db, userID)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.Locale != DefaultLocale || defaults.Timezone != DefaultTimezone {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	if err := UpsertUserSettings(ctx, db, UserSettingsRecord{
		UserID:                        userID,
		Locale:                        "en-US",
		Timezone:                      "America/Los_Angeles",
		ModelProfile:                  "FAST",
		MaxPlanIterations:             sql.NullInt64{Int64: 99, Valid: true},
		MaxStepNum:                    sql.NullInt64{Int64: 0, Valid: true},
		DailyTokenBudget:              sql.NullInt64{Int64: 500000, Valid: true},
		DailyCostBudgetMicros:         sql.NullInt64{Int64: 2500000, Valid: true},
		EnableBackgroundInvestigation: sql.NullBool{Bool: true, Valid: true},
		AutoAcceptPlan:                sql.NullBool{Bool: false, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := GetUserSettings(ctx, db, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Locale != "en-US" || got.Timezone != "America/Los_Angeles" {
		t.Fatalf("unexpected locale/timezone: %+v", got)
	}
	if got.ModelProfile != "fast" {
		t.Fatalf("model_profile not normalized/persisted: %+v", got)
	}
	if got.MaxPlanIterations.Int64 != 10 || got.MaxStepNum.Int64 != 1 {
		t.Fatalf("settings should be clamped: %+v", got)
	}
	if !got.DailyTokenBudget.Valid || got.DailyTokenBudget.Int64 != 500000 {
		t.Fatalf("daily_token_budget not persisted: %+v", got)
	}
	if !got.DailyCostBudgetMicros.Valid || got.DailyCostBudgetMicros.Int64 != 2500000 {
		t.Fatalf("daily_cost_budget_micros not persisted: %+v", got)
	}
	if !got.EnableBackgroundInvestigation.Valid || !got.EnableBackgroundInvestigation.Bool {
		t.Fatalf("enable_background_investigation not persisted: %+v", got)
	}
	if !got.AutoAcceptPlan.Valid || got.AutoAcceptPlan.Bool {
		t.Fatalf("auto_accept_plan not persisted: %+v", got)
	}
}
