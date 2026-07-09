## ADDED Requirements

### Requirement: Artifact share links are durable and revocable

The system MUST create artifact share links backed by shared storage so any service replica can resolve or revoke the same link.

#### Scenario: Create a share link

- **GIVEN** a user owns an artifact
- **WHEN** the user creates a share link
- **THEN** the service stores a hashed share token in MySQL
- **AND** returns the raw token and public share URL to the creator.

#### Scenario: Resolve a share link from another pod

- **GIVEN** pod A creates a share link
- **AND** pod B receives a public share request with the token
- **WHEN** the share is not expired or revoked
- **THEN** pod B returns the artifact content from shared storage.

#### Scenario: Revoke a share link

- **GIVEN** a user has created a share link
- **WHEN** the user revokes the share link
- **THEN** later public reads with the same token are rejected.

### Requirement: Public artifact shares do not expose internal ownership IDs

Public share responses MUST avoid exposing private user, thread, and run identifiers.

#### Scenario: Read public share

- **GIVEN** a valid artifact share token
- **WHEN** a public reader resolves the share
- **THEN** the artifact content is returned
- **AND** `user_id`, `thread_id`, and `run_id` are omitted or empty.
