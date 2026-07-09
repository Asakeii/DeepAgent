package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	PluginScopeUser = "user"
	PluginScopeTeam = "team"
)

type PluginInstallRecord struct {
	ScopeType string
	ScopeID   string
	Server    string
	Enabled   bool
	UpdatedBy string
	UpdatedAt time.Time
}

func EnsurePluginTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS plugin_installs (
		scope_type VARCHAR(16) NOT NULL,
		scope_id   VARCHAR(128) NOT NULL,
		server     VARCHAR(128) NOT NULL,
		enabled    TINYINT(1) NOT NULL DEFAULT 1,
		updated_by VARCHAR(128) NOT NULL DEFAULT '',
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (scope_type, scope_id, server),
		KEY idx_server_enabled (server, enabled)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return fmt.Errorf("ensure plugin tables: %w", err)
	}
	return nil
}

func UpsertPluginInstall(ctx context.Context, db *sql.DB, record PluginInstallRecord) (PluginInstallRecord, error) {
	if db == nil {
		return PluginInstallRecord{}, fmt.Errorf("db is nil")
	}
	record.ScopeType = NormalizePluginScopeType(record.ScopeType)
	record.ScopeID = strings.TrimSpace(record.ScopeID)
	record.Server = strings.TrimSpace(record.Server)
	record.UpdatedBy = NormalizeUserID(record.UpdatedBy)
	if record.ScopeID == "" {
		return PluginInstallRecord{}, fmt.Errorf("%w: plugin scope id is required", ErrValidation)
	}
	if record.Server == "" {
		return PluginInstallRecord{}, fmt.Errorf("%w: plugin server is required", ErrValidation)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO plugin_installs (scope_type, scope_id, server, enabled, updated_by)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE enabled=VALUES(enabled), updated_by=VALUES(updated_by), updated_at=CURRENT_TIMESTAMP`,
		record.ScopeType, record.ScopeID, record.Server, record.Enabled, record.UpdatedBy,
	)
	if err != nil {
		return PluginInstallRecord{}, fmt.Errorf("upsert plugin install: %w", err)
	}
	return record, nil
}

func ListPluginInstalls(ctx context.Context, db *sql.DB, scopeType, scopeID string) ([]PluginInstallRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	scopeType = NormalizePluginScopeType(scopeType)
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return nil, fmt.Errorf("%w: plugin scope id is required", ErrValidation)
	}
	rows, err := db.QueryContext(ctx,
		`SELECT scope_type, scope_id, server, enabled, updated_by, updated_at
		 FROM plugin_installs
		 WHERE scope_type=? AND scope_id=?
		 ORDER BY server ASC`,
		scopeType, scopeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list plugin installs: %w", err)
	}
	defer rows.Close()
	var out []PluginInstallRecord
	for rows.Next() {
		var record PluginInstallRecord
		if err := rows.Scan(&record.ScopeType, &record.ScopeID, &record.Server, &record.Enabled, &record.UpdatedBy, &record.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func EnabledPluginServers(ctx context.Context, db *sql.DB, scopeType, scopeID string, configured []string) (map[string]bool, error) {
	available := make(map[string]bool, len(configured))
	for _, server := range configured {
		server = strings.TrimSpace(server)
		if server != "" {
			available[server] = true
		}
	}
	if len(available) == 0 {
		return map[string]bool{}, nil
	}
	records, err := ListPluginInstalls(ctx, db, scopeType, scopeID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(available))
	for server := range available {
		out[server] = true
	}
	for _, record := range records {
		if available[record.Server] {
			out[record.Server] = record.Enabled
		}
	}
	return out, nil
}

func NormalizePluginScopeType(scopeType string) string {
	if strings.EqualFold(strings.TrimSpace(scopeType), PluginScopeTeam) {
		return PluginScopeTeam
	}
	return PluginScopeUser
}

func PluginScopeFor(userID, teamID string) (string, string) {
	teamID = strings.TrimSpace(teamID)
	if teamID != "" {
		return PluginScopeTeam, teamID
	}
	return PluginScopeUser, NormalizeUserID(userID)
}

func SortedPluginServers(servers map[string]bool) []string {
	out := make([]string, 0, len(servers))
	for server, enabled := range servers {
		if enabled {
			out = append(out, server)
		}
	}
	sort.Strings(out)
	return out
}
