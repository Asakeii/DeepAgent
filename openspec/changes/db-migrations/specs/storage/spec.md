# Storage Spec

## ADDED Requirements

### Requirement: Database schema changes are versioned

The service MUST record applied schema migrations in a database table.

#### Scenario: First startup

- **GIVEN** the database has no `schema_migrations` table
- **WHEN** the service initializes the database
- **THEN** the service creates `schema_migrations`
- **AND** applies embedded SQL migrations in version order

#### Scenario: Restart after migrations applied

- **GIVEN** a migration version is already recorded with the same checksum
- **WHEN** the service initializes the database again
- **THEN** the migration is skipped

#### Scenario: Applied migration was edited

- **GIVEN** a migration version is recorded with a checksum
- **WHEN** the embedded migration file has a different checksum
- **THEN** initialization fails instead of silently applying a modified migration

### Requirement: Migration execution is safe for multiple pods

The service MUST serialize migration execution across pods sharing the same MySQL database.

#### Scenario: Two pods start concurrently

- **GIVEN** two service instances call database initialization at the same time
- **WHEN** migrations run
- **THEN** a MySQL advisory lock allows only one instance to apply migrations at a time
