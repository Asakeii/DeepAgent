CREATE TABLE IF NOT EXISTS artifacts (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL DEFAULT '',
    thread_id   VARCHAR(128) NOT NULL DEFAULT '',
    run_id      VARCHAR(128) NOT NULL DEFAULT '',
    kind        VARCHAR(32) NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    format      VARCHAR(32) NOT NULL DEFAULT 'markdown',
    content     MEDIUMTEXT NOT NULL,
    metadata    JSON,
    version     BIGINT NOT NULL DEFAULT 1,
    source      VARCHAR(64) NOT NULL DEFAULT 'agent',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_created (user_id, created_at),
    KEY idx_thread_kind (thread_id, kind, created_at),
    KEY idx_run (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
