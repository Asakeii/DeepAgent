CREATE TABLE IF NOT EXISTS graph_checkpoint (
    thread_id   VARCHAR(128) NOT NULL PRIMARY KEY,
    data        LONGBLOB NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
