CREATE TABLE IF NOT EXISTS graph_checkpoint (
    thread_id   VARCHAR(128) NOT NULL PRIMARY KEY,
    data        LONGBLOB NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(128) NOT NULL PRIMARY KEY,
    provider     VARCHAR(32) NOT NULL,
    provider_id  VARCHAR(128) NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_provider_user (provider, provider_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS threads (
    id          VARCHAR(128) NOT NULL PRIMARY KEY,
    user_id     VARCHAR(128) NOT NULL,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    source      VARCHAR(32) NOT NULL DEFAULT 'web',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user_updated (user_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS messages (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    thread_id   VARCHAR(128) NOT NULL,
    turn_idx    INT NOT NULL,
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
