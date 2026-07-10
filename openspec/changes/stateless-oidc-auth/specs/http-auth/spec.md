# HTTP Authentication Spec

## ADDED Requirements

### Requirement: Protected routes use a verified principal

When authentication is configured, the service MUST reject protected requests without a valid credential and MUST derive the caller identity only from the verified credential.

#### Scenario: Valid OIDC bearer token

- **GIVEN** an OIDC issuer and audience are configured
- **WHEN** a protected route receives a signed, unexpired Bearer JWT with matching issuer and audience
- **THEN** the service validates the token with the provider JWKS
- **AND** stores the mapped principal in request context

#### Scenario: Caller tries to override verified identity

- **GIVEN** a request has a verified principal
- **WHEN** the same request contains a different `X-DeepAgent-User`, `user_id`, or chat body `user_id`
- **THEN** the service ignores the untrusted value
- **AND** authorizes data access as the verified principal

#### Scenario: Authentication is not configured for local development

- **GIVEN** OIDC, API keys, admin API keys, and API key principals are all empty
- **WHEN** a protected route is called
- **THEN** the service preserves the existing local development identity behavior

### Requirement: Machine credentials have stable identities

The service MUST map each configured API key to a deterministic or explicitly named service principal.

#### Scenario: Named API key principal

- **GIVEN** an API key is bound to `service:automation`
- **WHEN** the key calls a protected route
- **THEN** the request principal is `service:automation`
- **AND** caller-supplied user identity headers cannot change it

### Requirement: Administrator authorization is claim based

The service MUST allow an authenticated principal to access administrator routes only when the principal has administrator authority.

#### Scenario: OIDC user has configured administrator role

- **GIVEN** `deepagent-admin` is configured as an administrator role
- **WHEN** a valid token contains that role in the configured roles claim
- **THEN** the principal can access `/api/admin/*`

#### Scenario: Authenticated non-admin user

- **GIVEN** a valid principal without administrator authority
- **WHEN** the principal calls `/api/admin/*`
- **THEN** the service responds with HTTP 403

### Requirement: Authentication remains stateless across pods

The service MUST NOT require an in-memory login session to authenticate protected requests.

#### Scenario: Request reaches another pod

- **GIVEN** two Pods use the same OIDC issuer/audience configuration
- **WHEN** two requests with the same valid JWT reach different Pods
- **THEN** both Pods derive the same internal user identity
- **AND** enforce ownership using shared MySQL state
