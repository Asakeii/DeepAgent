ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS daily_cost_budget_micros BIGINT NULL AFTER daily_token_budget;

CREATE TABLE IF NOT EXISTS team_settings (
    team_id                    VARCHAR(128) NOT NULL PRIMARY KEY,
    daily_cost_budget_micros   BIGINT NULL,
    updated_by                 VARCHAR(128) NOT NULL DEFAULT '',
    updated_at                 TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
