CREATE TABLE IF NOT EXISTS graph_checkpoint (
    thread_id   VARCHAR(128) NOT NULL PRIMARY KEY,
    data        LONGBLOB NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     BIGINT NOT NULL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    checksum    CHAR(64) NOT NULL,
    applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    provider     VARCHAR(32) NOT NULL,
    provider_id  VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_provider_user (provider, provider_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teams (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    created_by  VARCHAR(128) NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_created_by (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS team_members (
    team_id    VARCHAR(128) NOT NULL,
    user_id    VARCHAR(128) NOT NULL,
    role       VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (team_id, user_id),
    KEY idx_user_team (user_id, team_id),
    KEY idx_team_role (team_id, role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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

CREATE TABLE IF NOT EXISTS threads (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    team_id     VARCHAR(128) NOT NULL DEFAULT '',
    title       VARCHAR(255) NOT NULL DEFAULT '',
    source      VARCHAR(32) NOT NULL DEFAULT 'web',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_updated (user_id, updated_at),
    KEY idx_team_updated (team_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id   VARCHAR(128) NOT NULL,
    turn_idx    BIGINT NOT NULL,
    role        VARCHAR(32) NOT NULL,
    content     MEDIUMTEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_thread_turn (thread_id, turn_idx)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS runs (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL DEFAULT '',
    thread_id   VARCHAR(128) NOT NULL,
    mode        VARCHAR(32) NOT NULL,
    status      VARCHAR(32) NOT NULL,
    started_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at    TIMESTAMP NULL,
    error       TEXT,
    cancel_requested_at TIMESTAMP NULL,
    KEY idx_thread_started (thread_id, started_at),
    KEY idx_user_started (user_id, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS run_events (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    run_id      VARCHAR(128) NOT NULL,
    thread_id   VARCHAR(128) NOT NULL,
    user_id     VARCHAR(128) NOT NULL DEFAULT '',
    event_name  VARCHAR(64) NOT NULL,
    agent       VARCHAR(64) NOT NULL DEFAULT '',
    payload     JSON NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_run_id (run_id, id),
    KEY idx_thread_created (thread_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tool_audit_logs (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    run_id        VARCHAR(128) NOT NULL DEFAULT '',
    thread_id     VARCHAR(128) NOT NULL DEFAULT '',
    user_id       VARCHAR(128) NOT NULL DEFAULT '',
    tool_name     VARCHAR(128) NOT NULL,
    risk          VARCHAR(32) NOT NULL,
    status        VARCHAR(32) NOT NULL,
    arguments     JSON,
    result        MEDIUMTEXT,
    error         TEXT,
    duration_ms   BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at  TIMESTAMP NULL,
    KEY idx_run_tool (run_id, id),
    KEY idx_thread_created (thread_id, created_at),
    KEY idx_tool_status (tool_name, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS checkins (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id   VARCHAR(128) NOT NULL,
    category    VARCHAR(32) NOT NULL,
    content     TEXT NOT NULL,
    value       DOUBLE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_thread_cat (thread_id, category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS reminders (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id     VARCHAR(128) NOT NULL,
    cron          VARCHAR(64) NOT NULL,
    content       TEXT NOT NULL,
    next_fire_at  TIMESTAMP NOT NULL,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    KEY idx_fire (status, next_fire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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

CREATE TABLE IF NOT EXISTS user_settings (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
