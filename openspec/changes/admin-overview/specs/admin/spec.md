## ADDED Requirements

### Requirement: Admin APIs require admin credentials

The system MUST protect admin APIs with admin-specific credentials.

#### Scenario: Normal API key cannot access admin API

- **GIVEN** normal API keys and admin API keys are configured
- **WHEN** a request to `/api/admin/overview` uses only a normal API key
- **THEN** the service rejects the request.

#### Scenario: Admin API key can access admin API

- **GIVEN** admin API keys are configured
- **WHEN** a request to `/api/admin/overview` uses an admin API key
- **THEN** the service allows the request to reach the handler.

### Requirement: Admin overview aggregates shared-state metrics

The system MUST provide a read-only admin overview sourced from shared storage.

#### Scenario: Query admin overview

- **GIVEN** runs, tool audits, model usage, users, threads, artifacts, and artifact shares exist in MySQL
- **WHEN** an admin queries `/api/admin/overview`
- **THEN** the response includes aggregate counts and rates across users
- **AND** the response is independent of which pod handles the request.
