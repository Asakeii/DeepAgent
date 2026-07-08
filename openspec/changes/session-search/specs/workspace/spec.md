## ADDED Requirements

### Requirement: Users can search their own sessions

The session API MUST support searching sessions within the requesting user's ownership boundary.

#### Scenario: Search by message content

- **GIVEN** a user has multiple sessions
- **WHEN** the user requests `/api/sessions?q=<keyword>`
- **THEN** the API returns only sessions owned by that user that match the keyword

### Requirement: The workspace exposes session search

The frontend workspace MUST provide a search input in the session sidebar.

#### Scenario: No matching sessions

- **GIVEN** the user enters a search query
- **WHEN** no sessions match
- **THEN** the sidebar shows an empty search result state
