## ADDED Requirements

### Requirement: User settings are persisted outside the process

The system MUST persist user settings in shared storage so every stateless service replica can read the same preferences.

#### Scenario: Read default settings

- **GIVEN** a user has no persisted settings
- **WHEN** the user requests settings
- **THEN** the API returns stable default locale and timezone values

#### Scenario: Update settings

- **GIVEN** a user sends a settings update
- **WHEN** the update is accepted
- **THEN** the settings are stored under that user's id
- **AND** a later request from another pod can read the same values from the database

### Requirement: Chat runs use user settings as defaults

ChatService MUST apply user settings before running Agent orchestration when the request does not explicitly provide those fields.

#### Scenario: Run defaults

- **GIVEN** a user has max plan iterations, max step count, locale, and background investigation settings
- **WHEN** the user starts a chat run without those request fields
- **THEN** ChatService passes the user's settings to the runner
