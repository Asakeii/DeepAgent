# Agent Observability Spec

## ADDED Requirements

### Requirement: Run metrics are queryable by user

The service MUST expose a protected API for recent run and tool metrics scoped to the requesting user.

#### Scenario: Query recent run metrics

- **GIVEN** the caller is authenticated or allowed by local configuration
- **WHEN** the caller requests `/api/metrics/runs?window_hours=24`
- **THEN** the service returns run totals, success rate, latency metrics, and tool error metrics for the resolved `user_id`

#### Scenario: Window is too large

- **GIVEN** a caller requests a very large `window_hours`
- **WHEN** the metrics API handles the request
- **THEN** the service caps the window to protect the database

#### Scenario: No recent runs

- **GIVEN** the user has no runs in the requested window
- **WHEN** the metrics API handles the request
- **THEN** the service returns zero counts and zero rates
