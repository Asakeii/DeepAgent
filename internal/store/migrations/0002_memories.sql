CREATE TABLE IF NOT EXISTS memories (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL DEFAULT '',
    kind        VARCHAR(32) NOT NULL,
    content     TEXT NOT NULL,
    importance  INT NOT NULL DEFAULT 0,
    source      VARCHAR(64) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_kind (user_id, kind, importance),
    KEY idx_user_updated (user_id, updated_at),
    KEY idx_thread_updated (thread_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
