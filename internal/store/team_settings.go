package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type TeamSettingsRecord struct {
	TeamID                string
	DailyCostBudgetMicros sql.NullInt64
	UpdatedBy             string
	UpdatedAt             time.Time
}

func EnsureTeamSettingsTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS team_settings (
		team_id                    VARCHAR(128) NOT NULL PRIMARY KEY,
		daily_cost_budget_micros   BIGINT NULL,
		updated_by                 VARCHAR(128) NOT NULL DEFAULT '',
		updated_at                 TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure team settings table: %w", err)
	}
	return nil
}

func DefaultTeamSettings(teamID string) TeamSettingsRecord {
	return TeamSettingsRecord{TeamID: strings.TrimSpace(teamID)}
}

func GetTeamSettings(ctx context.Context, db *sql.DB, teamID string) (TeamSettingsRecord, error) {
	if db == nil {
		return TeamSettingsRecord{}, fmt.Errorf("db is nil")
	}
	record := DefaultTeamSettings(teamID)
	if record.TeamID == "" {
		return TeamSettingsRecord{}, fmt.Errorf("%w: team id is required", ErrValidation)
	}
	err := db.QueryRowContext(ctx,
		`SELECT team_id, daily_cost_budget_micros, updated_by, updated_at
		 FROM team_settings WHERE team_id=?`,
		record.TeamID,
	).Scan(&record.TeamID, &record.DailyCostBudgetMicros, &record.UpdatedBy, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return record, nil
	}
	if err != nil {
		return TeamSettingsRecord{}, fmt.Errorf("get team settings: %w", err)
	}
	normalizeTeamSettings(&record)
	return record, nil
}

func UpsertTeamSettings(ctx context.Context, db *sql.DB, record TeamSettingsRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	normalizeTeamSettings(&record)
	if record.TeamID == "" {
		return fmt.Errorf("%w: team id is required", ErrValidation)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO team_settings (team_id, daily_cost_budget_micros, updated_by)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 daily_cost_budget_micros=VALUES(daily_cost_budget_micros),
		 updated_by=VALUES(updated_by),
		 updated_at=CURRENT_TIMESTAMP`,
		record.TeamID, record.DailyCostBudgetMicros, record.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("upsert team settings: %w", err)
	}
	return nil
}

func normalizeTeamSettings(record *TeamSettingsRecord) {
	record.TeamID = strings.TrimSpace(record.TeamID)
	record.UpdatedBy = NormalizeUserID(record.UpdatedBy)
	record.DailyCostBudgetMicros = normalizeNullableInt(record.DailyCostBudgetMicros, 1, 1_000_000_000_000)
}
