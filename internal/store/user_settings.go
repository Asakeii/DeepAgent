package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultLocale   = "zh-CN"
	DefaultTimezone = "Asia/Shanghai"
)

type UserSettingsRecord struct {
	UserID                        string
	Locale                        string
	Timezone                      string
	ModelProfile                  string
	MaxPlanIterations             sql.NullInt64
	MaxStepNum                    sql.NullInt64
	DailyTokenBudget              sql.NullInt64
	EnableBackgroundInvestigation sql.NullBool
	AutoAcceptPlan                sql.NullBool
	UpdatedAt                     time.Time
}

func EnsureUserSettingsTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS user_settings (
		user_id                         VARCHAR(128) NOT NULL PRIMARY KEY,
		locale                          VARCHAR(32) NOT NULL DEFAULT 'zh-CN',
		timezone                        VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
		model_profile                   VARCHAR(64) NOT NULL DEFAULT '',
		max_plan_iterations             INT NULL,
		max_step_num                    INT NULL,
		daily_token_budget              INT NULL,
		enable_background_investigation TINYINT(1) NULL,
		auto_accept_plan                TINYINT(1) NULL,
		updated_at                      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure user settings table: %w", err)
	}
	return nil
}

func DefaultUserSettings(userID string) UserSettingsRecord {
	return UserSettingsRecord{
		UserID:   NormalizeUserID(userID),
		Locale:   DefaultLocale,
		Timezone: DefaultTimezone,
	}
}

func GetUserSettings(ctx context.Context, db *sql.DB, userID string) (UserSettingsRecord, error) {
	if db == nil {
		return UserSettingsRecord{}, fmt.Errorf("db is nil")
	}
	record := DefaultUserSettings(userID)
	err := db.QueryRowContext(ctx,
		`SELECT user_id, locale, timezone, model_profile, max_plan_iterations, max_step_num,
		 daily_token_budget, enable_background_investigation, auto_accept_plan, updated_at
		 FROM user_settings WHERE user_id=?`,
		record.UserID,
	).Scan(&record.UserID, &record.Locale, &record.Timezone, &record.ModelProfile, &record.MaxPlanIterations, &record.MaxStepNum, &record.DailyTokenBudget, &record.EnableBackgroundInvestigation, &record.AutoAcceptPlan, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return record, nil
	}
	if err != nil {
		return UserSettingsRecord{}, fmt.Errorf("get user settings: %w", err)
	}
	normalizeUserSettings(&record)
	return record, nil
}

func UpsertUserSettings(ctx context.Context, db *sql.DB, record UserSettingsRecord) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	normalizeUserSettings(&record)
	if err := EnsureUser(ctx, db, record.UserID, "local", record.UserID); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO user_settings
		 (user_id, locale, timezone, model_profile, max_plan_iterations, max_step_num, daily_token_budget, enable_background_investigation, auto_accept_plan)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		 locale=VALUES(locale),
		 timezone=VALUES(timezone),
		 model_profile=VALUES(model_profile),
		 max_plan_iterations=VALUES(max_plan_iterations),
		 max_step_num=VALUES(max_step_num),
		 daily_token_budget=VALUES(daily_token_budget),
		 enable_background_investigation=VALUES(enable_background_investigation),
		 auto_accept_plan=VALUES(auto_accept_plan)`,
		record.UserID, record.Locale, record.Timezone, record.ModelProfile, record.MaxPlanIterations, record.MaxStepNum, record.DailyTokenBudget, record.EnableBackgroundInvestigation, record.AutoAcceptPlan,
	)
	if err != nil {
		return fmt.Errorf("upsert user settings: %w", err)
	}
	return nil
}

func normalizeUserSettings(record *UserSettingsRecord) {
	record.UserID = NormalizeUserID(record.UserID)
	record.Locale = strings.TrimSpace(record.Locale)
	if record.Locale == "" {
		record.Locale = DefaultLocale
	}
	if len(record.Locale) > 32 {
		record.Locale = record.Locale[:32]
	}
	record.Timezone = strings.TrimSpace(record.Timezone)
	if record.Timezone == "" {
		record.Timezone = DefaultTimezone
	}
	if len(record.Timezone) > 64 {
		record.Timezone = record.Timezone[:64]
	}
	record.ModelProfile = strings.ToLower(strings.TrimSpace(record.ModelProfile))
	if len(record.ModelProfile) > 64 {
		record.ModelProfile = record.ModelProfile[:64]
	}
	record.MaxPlanIterations = normalizeNullableInt(record.MaxPlanIterations, 1, 10)
	record.MaxStepNum = normalizeNullableInt(record.MaxStepNum, 1, 20)
	record.DailyTokenBudget = normalizeNullableInt(record.DailyTokenBudget, 1000, 100000000)
}

func normalizeNullableInt(value sql.NullInt64, min, max int64) sql.NullInt64 {
	if !value.Valid {
		return value
	}
	if value.Int64 < min {
		value.Int64 = min
	}
	if value.Int64 > max {
		value.Int64 = max
	}
	return value
}
