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

ALTER TABLE threads ADD COLUMN IF NOT EXISTS team_id VARCHAR(128) NOT NULL DEFAULT '' AFTER user_id;
ALTER TABLE threads ADD INDEX idx_team_updated (team_id, updated_at);
