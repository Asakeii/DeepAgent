CREATE TABLE IF NOT EXISTS artifact_shares (
    token_hash  CHAR(64) NOT NULL PRIMARY KEY,
    artifact_id BIGINT NOT NULL,
    user_id     VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP NULL,
    revoked_at  TIMESTAMP NULL,
    KEY idx_artifact (artifact_id),
    KEY idx_user_created (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
