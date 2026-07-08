# Operability Spec

## ADDED Requirements

### Requirement: Service exposes liveness and readiness endpoints

The HTTP server MUST expose separate liveness and readiness endpoints for deployment orchestration.

#### Scenario: Liveness probe

- **WHEN** `GET /healthz` is called
- **THEN** the service returns HTTP 200
- **AND** the response body reports `status=ok`

#### Scenario: Readiness probe fails before dependencies initialize

- **GIVEN** required dependencies are not initialized
- **WHEN** `GET /readyz` is called
- **THEN** the service returns HTTP 503
- **AND** the response body reports `status=not_ready`

#### Scenario: Readiness probe succeeds

- **GIVEN** MySQL and ChatModel are initialized
- **AND** Redis is either disabled or reachable
- **WHEN** `GET /readyz` is called
- **THEN** the service returns HTTP 200
- **AND** the response body reports `status=ready`
