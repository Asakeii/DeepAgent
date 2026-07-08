## ADDED Requirements

### Requirement: Research reports are persisted as artifacts

When a research run completes with a non-empty final report, the system MUST persist the report as a markdown artifact associated with the run, thread, and user.

#### Scenario: Completed research run

- **GIVEN** a chat request runs through the research path
- **AND** the final research response is non-empty
- **WHEN** the run completes
- **THEN** the final response is stored as an artifact with kind `report`
- **AND** the artifact records the `run_id`, `thread_id`, and `user_id`

### Requirement: Artifacts are queryable within user ownership boundaries

The API MUST return artifacts only for the requesting user, and MUST reject thread-filtered queries for threads the user does not own.

#### Scenario: Query artifacts by owned thread

- **GIVEN** a user owns a thread with report artifacts
- **WHEN** the user requests `/api/artifacts?thread_id=<thread>`
- **THEN** the API returns artifacts for that thread

#### Scenario: Query artifacts by foreign thread

- **GIVEN** a user does not own a thread
- **WHEN** the user requests `/api/artifacts?thread_id=<thread>`
- **THEN** the API rejects the request
