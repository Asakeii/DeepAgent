ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS daily_token_budget INT NULL AFTER max_step_num;
