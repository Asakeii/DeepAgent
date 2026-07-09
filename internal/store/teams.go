package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TeamRoleOwner  = "owner"
	TeamRoleAdmin  = "admin"
	TeamRoleMember = "member"
)

var (
	ErrValidation    = errors.New("validation error")
	ErrTeamForbidden = errors.New("team forbidden")
)

type TeamRecord struct {
	ID        string
	Name      string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeamMemberRecord struct {
	TeamID    string
	UserID    string
	Role      string
	CreatedAt time.Time
}

func EnsureTeamTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS teams (
			id          VARCHAR(128) NOT NULL PRIMARY KEY,
			name        VARCHAR(128) NOT NULL,
			created_by  VARCHAR(128) NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			KEY idx_created_by (created_by, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE IF NOT EXISTS team_members (
			team_id    VARCHAR(128) NOT NULL,
			user_id    VARCHAR(128) NOT NULL,
			role       VARCHAR(32) NOT NULL DEFAULT 'member',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (team_id, user_id),
			KEY idx_user_team (user_id, team_id),
			KEY idx_team_role (team_id, role)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure team tables: %w", err)
		}
	}
	if err := ensureColumn(ctx, db, "threads", "team_id", "VARCHAR(128) NOT NULL DEFAULT '' AFTER user_id"); err != nil {
		return fmt.Errorf("ensure threads team_id: %w", err)
	}
	return nil
}

func CreateTeam(ctx context.Context, db *sql.DB, userID, name string) (TeamRecord, error) {
	if db == nil {
		return TeamRecord{}, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	name = normalizeTeamName(name)
	if name == "" {
		return TeamRecord{}, fmt.Errorf("%w: team name is required", ErrValidation)
	}
	if err := EnsureUser(ctx, db, userID, "local", userID); err != nil {
		return TeamRecord{}, err
	}
	record := TeamRecord{ID: newTeamID(), Name: name, Role: TeamRoleOwner}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return TeamRecord{}, fmt.Errorf("begin create team: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO teams (id, name, created_by) VALUES (?, ?, ?)`,
		record.ID, record.Name, userID,
	); err != nil {
		return TeamRecord{}, fmt.Errorf("create team: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		record.ID, userID, TeamRoleOwner,
	); err != nil {
		return TeamRecord{}, fmt.Errorf("create owner membership: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return TeamRecord{}, fmt.Errorf("commit create team: %w", err)
	}
	return record, nil
}

func ListTeamsForUser(ctx context.Context, db *sql.DB, userID string) ([]TeamRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	userID = NormalizeUserID(userID)
	rows, err := db.QueryContext(ctx,
		`SELECT t.id, t.name, tm.role, t.created_at, t.updated_at
		 FROM team_members tm
		 JOIN teams t ON t.id=tm.team_id
		 WHERE tm.user_id=?
		 ORDER BY t.updated_at DESC, t.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()
	var out []TeamRecord
	for rows.Next() {
		var record TeamRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Role, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func AddTeamMember(ctx context.Context, db *sql.DB, teamID, actorUserID, memberUserID, role string) (TeamMemberRecord, error) {
	if db == nil {
		return TeamMemberRecord{}, fmt.Errorf("db is nil")
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return TeamMemberRecord{}, fmt.Errorf("%w: team id is required", ErrValidation)
	}
	actorUserID = NormalizeUserID(actorUserID)
	memberUserID = NormalizeUserID(memberUserID)
	role = NormalizeTeamRole(role)
	if ok, err := UserCanManageTeam(ctx, db, teamID, actorUserID); err != nil {
		return TeamMemberRecord{}, err
	} else if !ok {
		return TeamMemberRecord{}, ErrTeamForbidden
	}
	if err := EnsureUser(ctx, db, memberUserID, "local", memberUserID); err != nil {
		return TeamMemberRecord{}, err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE role=VALUES(role)`,
		teamID, memberUserID, role,
	)
	if err != nil {
		return TeamMemberRecord{}, fmt.Errorf("add team member: %w", err)
	}
	return TeamMemberRecord{TeamID: teamID, UserID: memberUserID, Role: role}, nil
}

func ListTeamMembers(ctx context.Context, db *sql.DB, teamID, userID string) ([]TeamMemberRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return nil, fmt.Errorf("%w: team id is required", ErrValidation)
	}
	userID = NormalizeUserID(userID)
	if ok, err := UserIsTeamMember(ctx, db, teamID, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrTeamForbidden
	}
	rows, err := db.QueryContext(ctx,
		`SELECT team_id, user_id, role, created_at
		 FROM team_members
		 WHERE team_id=?
		 ORDER BY created_at ASC, user_id ASC`,
		teamID,
	)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()
	var out []TeamMemberRecord
	for rows.Next() {
		var record TeamMemberRecord
		if err := rows.Scan(&record.TeamID, &record.UserID, &record.Role, &record.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func UserIsTeamMember(ctx context.Context, db *sql.DB, teamID, userID string) (bool, error) {
	return userHasTeamRole(ctx, db, teamID, userID, nil)
}

func UserCanManageTeam(ctx context.Context, db *sql.DB, teamID, userID string) (bool, error) {
	return userHasTeamRole(ctx, db, teamID, userID, map[string]bool{
		TeamRoleOwner: true,
		TeamRoleAdmin: true,
	})
}

func NormalizeTeamRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case TeamRoleOwner:
		return TeamRoleOwner
	case TeamRoleAdmin:
		return TeamRoleAdmin
	default:
		return TeamRoleMember
	}
}

func normalizeTeamName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 128 {
		return name[:128]
	}
	return name
}

func userHasTeamRole(ctx context.Context, db *sql.DB, teamID, userID string, allowed map[string]bool) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("db is nil")
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return false, nil
	}
	userID = NormalizeUserID(userID)
	var role string
	err := db.QueryRowContext(ctx,
		`SELECT role FROM team_members WHERE team_id=? AND user_id=?`,
		teamID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query team member: %w", err)
	}
	if allowed == nil {
		return true, nil
	}
	return allowed[role], nil
}

func newTeamID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "team"
	}
	return "team_" + hex.EncodeToString(b[:])
}
