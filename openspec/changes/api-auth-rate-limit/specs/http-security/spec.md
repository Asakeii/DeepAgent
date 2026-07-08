# HTTP Security Spec

## ADDED Requirements

### Requirement: Protected API routes can require API keys

The server MUST support optional API-key authentication for Agent and API routes.

#### Scenario: API keys are not configured

- **GIVEN** `server.api_keys` is empty
- **WHEN** a request reaches a protected API route
- **THEN** the request is allowed to continue without API-key authentication

#### Scenario: API key is configured and missing

- **GIVEN** `server.api_keys` is not empty
- **WHEN** a request reaches `/api/*`, `/chat/stream`, or `/v1/chat/completions` without a valid key
- **THEN** the server rejects it with HTTP 401

#### Scenario: Bearer API key is valid

- **GIVEN** `server.api_keys` contains `k1`
- **WHEN** a protected route receives `Authorization: Bearer k1`
- **THEN** the request is allowed to continue

### Requirement: Rate limiting is shared across pods

The server MUST support Redis-backed request rate limiting for protected API routes.

#### Scenario: Redis is available

- **GIVEN** `server.rate_limit_per_minute` is greater than zero
- **AND** Redis is initialized
- **WHEN** protected API requests arrive
- **THEN** the server increments a Redis counter keyed by caller identity and minute window
- **AND** returns HTTP 429 when the configured limit is exceeded

#### Scenario: Redis is unavailable

- **GIVEN** rate limiting is configured
- **AND** Redis is not initialized
- **WHEN** protected API requests arrive
- **THEN** the server allows the request to continue
