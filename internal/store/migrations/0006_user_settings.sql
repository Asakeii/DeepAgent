CREATE TABLE IF NOT EXISTS user_settings (
    user_id                         VARCHAR(128) NOT NULL PRIMARY KEY,
    locale                          VARCHAR(32) NOT NULL DEFAULT 'zh-CN',
    timezone                        VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    max_plan_iterations             INT NULL,
    max_step_num                    INT NULL,
    enable_background_investigation TINYINT(1) NULL,
    auto_accept_plan                TINYINT(1) NULL,
    updated_at                      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
