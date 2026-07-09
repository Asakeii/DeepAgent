CREATE TABLE IF NOT EXISTS plugin_installs (
    scope_type VARCHAR(16) NOT NULL,
    scope_id   VARCHAR(128) NOT NULL,
    server     VARCHAR(128) NOT NULL,
    enabled    TINYINT(1) NOT NULL DEFAULT 1,
    updated_by VARCHAR(128) NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (scope_type, scope_id, server),
    KEY idx_server_enabled (server, enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
