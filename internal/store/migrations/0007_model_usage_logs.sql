CREATE TABLE IF NOT EXISTS model_usage_logs (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    run_id            VARCHAR(128) NOT NULL DEFAULT '',
    thread_id         VARCHAR(128) NOT NULL DEFAULT '',
    user_id           VARCHAR(128) NOT NULL DEFAULT '',
    agent             VARCHAR(64) NOT NULL DEFAULT '',
    model             VARCHAR(128) NOT NULL DEFAULT '',
    prompt_tokens     INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens      INT NOT NULL DEFAULT 0,
    cached_tokens     INT NOT NULL DEFAULT 0,
    reasoning_tokens  INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_run_model (run_id, id),
    KEY idx_user_created (user_id, created_at),
    KEY idx_agent_created (agent, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
