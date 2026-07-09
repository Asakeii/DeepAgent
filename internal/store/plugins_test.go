package store

import (
	"context"
	"testing"
)

func TestEnabledPluginServersDefaultsToConfigured(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsurePluginTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	userID := "plugin-user-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM plugin_installs WHERE scope_id=?", userID)
	})

	enabled, err := EnabledPluginServers(ctx, db, PluginScopeUser, userID, []string{"python", "tavily"})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled["python"] || !enabled["tavily"] {
		t.Fatalf("enabled=%v, want all configured enabled by default", enabled)
	}
}

func TestEnabledPluginServersHonorsExplicitInstalls(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := EnsurePluginTables(ctx, db); err != nil {
		t.Fatal(err)
	}
	userID := "plugin-user-" + randomSuffix()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM plugin_installs WHERE scope_id=?", userID)
	})
	if _, err := UpsertPluginInstall(ctx, db, PluginInstallRecord{
		ScopeType: PluginScopeUser,
		ScopeID:   userID,
		Server:    "python",
		Enabled:   true,
		UpdatedBy: userID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertPluginInstall(ctx, db, PluginInstallRecord{
		ScopeType: PluginScopeUser,
		ScopeID:   userID,
		Server:    "tavily",
		Enabled:   false,
		UpdatedBy: userID,
	}); err != nil {
		t.Fatal(err)
	}

	enabled, err := EnabledPluginServers(ctx, db, PluginScopeUser, userID, []string{"python", "tavily"})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled["python"] || enabled["tavily"] {
		t.Fatalf("enabled=%v, want only python enabled", enabled)
	}
}

func TestPluginScopeForTeam(t *testing.T) {
	scopeType, scopeID := PluginScopeFor("user-1", "team-1")
	if scopeType != PluginScopeTeam || scopeID != "team-1" {
		t.Fatalf("team scope=%s:%s", scopeType, scopeID)
	}
	scopeType, scopeID = PluginScopeFor("user-1", "")
	if scopeType != PluginScopeUser || scopeID != "user-1" {
		t.Fatalf("user scope=%s:%s", scopeType, scopeID)
	}
}
