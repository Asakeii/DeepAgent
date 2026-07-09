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

### Requirement: Frontend exposes a read-only admin overview

The frontend MUST provide a read-only management view that consumes the admin overview API without changing normal workspace or reminder flows.

#### Scenario: Admin opens the management view

- **GIVEN** the frontend app is loaded
- **WHEN** the admin selects the management view from the top bar
- **THEN** the app renders an admin key field, an aggregation-window selector, and a refresh action.

#### Scenario: Admin refreshes overview metrics

- **GIVEN** the admin has entered an Admin API Key
- **WHEN** the admin refreshes the management view
- **THEN** the frontend calls `/api/admin/overview` with the selected `window_hours`
- **AND** sends the key as a bearer token
- **AND** renders user, thread, artifact, share, run, tool, and token metrics.

#### Scenario: Missing admin key

- **GIVEN** the admin key field is empty
- **WHEN** the management view is displayed
- **THEN** the frontend does not call `/api/admin/overview`
- **AND** the refresh action is disabled.
