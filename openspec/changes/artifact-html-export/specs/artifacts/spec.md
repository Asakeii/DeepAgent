## ADDED Requirements

### Requirement: Artifact HTML export

The system MUST let a user export an owned artifact as a standalone HTML document.

#### Scenario: Export owned artifact

- **GIVEN** a user owns an artifact
- **WHEN** the user calls `/api/artifact-exports` with `format=html`
- **THEN** the service renders the artifact markdown as HTML
- **AND** returns a standalone HTML document with a download content disposition.

#### Scenario: Reject non-owned artifact export

- **GIVEN** an artifact belongs to another user
- **WHEN** a user calls `/api/artifact-exports` for that artifact
- **THEN** the service does not return the artifact content.

#### Scenario: Export works across pods

- **GIVEN** pod A created an artifact in MySQL
- **WHEN** pod B receives an export request for the same user's artifact
- **THEN** pod B reads the artifact from shared storage and renders the same HTML document.
