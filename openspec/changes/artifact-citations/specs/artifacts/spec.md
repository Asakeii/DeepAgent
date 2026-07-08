## ADDED Requirements

### Requirement: Report citations are stored separately from artifact content

When a report artifact contains markdown links, the system MUST persist those links as citation records associated with the artifact.

#### Scenario: Report with markdown citations

- **GIVEN** a completed research report contains markdown links
- **WHEN** the report is persisted as an artifact
- **THEN** each unique HTTP(S) markdown link is stored as a citation
- **AND** each citation records the artifact, run, thread, user, title, URL, and position

### Requirement: Artifact citations are protected by artifact ownership

The citation API MUST only return citations for artifacts owned by the requesting user.

#### Scenario: Owned artifact citations

- **GIVEN** a user owns an artifact with citations
- **WHEN** the user requests `/api/artifact-citations?artifact_id=<artifact>`
- **THEN** the API returns citations ordered by position

#### Scenario: Foreign artifact citations

- **GIVEN** a user does not own an artifact
- **WHEN** the user requests citations for that artifact
- **THEN** the API rejects the request
