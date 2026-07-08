package store

import (
	"context"
	"testing"
)

func TestMigrationVersion(t *testing.T) {
	got, err := migrationVersion("000123_add_runs.sql")
	if err != nil {
		t.Fatal(err)
	}
	if got != 123 {
		t.Fatalf("version=%d, want 123", got)
	}
	if _, err := migrationVersion("bad.sql"); err == nil {
		t.Fatal("expected invalid filename error")
	}
}

func TestSplitSQLStatementsIgnoresComments(t *testing.T) {
	stmts := splitSQLStatements(`
-- first table
CREATE TABLE a (id BIGINT);

-- second table
CREATE TABLE b (id BIGINT);
`)
	if len(stmts) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(stmts), stmts)
	}
}

func TestLoadMigrations(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no migrations loaded")
	}
	if migrations[0].Version != 1 || migrations[0].Checksum == "" || migrations[0].SQL == "" {
		t.Fatalf("unexpected first migration: %+v", migrations[0])
	}
}

func TestRunMigrationsWithMySQL(t *testing.T) {
	db := DBForTest(t)
	if db == nil {
		t.Skip("mysql not available")
	}
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration rows=%d, want 1", count)
	}
}
