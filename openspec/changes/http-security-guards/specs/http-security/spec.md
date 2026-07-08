# HTTP Security Spec

## ADDED Requirements

### Requirement: CORS is restricted to configured origins

The HTTP server MUST NOT use wildcard CORS for browser requests.

#### Scenario: Allowed browser origin

- **GIVEN** the request has an `Origin` configured in `server.allowed_origins`
- **WHEN** the request reaches the HTTP guard middleware
- **THEN** the response includes `Access-Control-Allow-Origin` with that origin

#### Scenario: Forbidden browser origin

- **GIVEN** the request has an `Origin` not configured in `server.allowed_origins`
- **WHEN** the request reaches the HTTP guard middleware
- **THEN** the server rejects it with HTTP 403

#### Scenario: Same-origin or server-side request

- **GIVEN** the request has no `Origin` header
- **WHEN** the request reaches the HTTP guard middleware
- **THEN** the request is allowed to continue

### Requirement: Request body size is bounded

The HTTP server MUST apply a configured maximum request body size before calling route handlers.

#### Scenario: Request body exceeds the configured limit

- **GIVEN** `server.max_body_bytes` is configured
- **WHEN** a route handler reads a request body larger than that limit
- **THEN** the read fails and the handler can return a request-too-large response
