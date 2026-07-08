CREATE TABLE IF NOT EXISTS artifact_citations (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    artifact_id BIGINT NOT NULL,
    user_id     VARCHAR(128) NOT NULL DEFAULT '',
    thread_id   VARCHAR(128) NOT NULL DEFAULT '',
    run_id      VARCHAR(128) NOT NULL DEFAULT '',
    title       VARCHAR(512) NOT NULL DEFAULT '',
    url         TEXT NOT NULL,
    quote       TEXT,
    position    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_artifact_position (artifact_id, position),
    KEY idx_user_created (user_id, created_at),
    KEY idx_thread_created (thread_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
