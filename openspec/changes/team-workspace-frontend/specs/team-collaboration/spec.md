# Team Collaboration Spec

## ADDED Requirements

### Requirement: Frontend team workspace switching

The frontend MUST let users switch between personal workspace and teams without relying on process-local state.

#### Scenario: User views personal sessions

- **GIVEN** the active workspace is personal
- **WHEN** the frontend loads sessions
- **THEN** it calls `/api/sessions` with an empty `team_id` scope
- **AND** the server returns only personal threads visible to the user.

#### Scenario: User views team sessions

- **GIVEN** the active workspace is a team
- **WHEN** the frontend loads sessions
- **THEN** it calls `/api/sessions` with that team's `team_id`
- **AND** the server returns only that team's threads visible to the user.

#### Scenario: User starts a team run

- **GIVEN** the active workspace is a team
- **WHEN** the user sends a prompt
- **THEN** the stream request includes `team_id`
- **AND** the backend creates or reuses a thread in that team scope.

### Requirement: Frontend team budget settings

The frontend MUST expose the team daily cost budget for team owners and admins.

#### Scenario: Manager saves team daily budget

- **GIVEN** the active workspace is a team
- **AND** the current user has owner or admin role
- **WHEN** the user saves a daily budget amount
- **THEN** the frontend sends `daily_cost_budget_micros` to `/api/team-settings`
- **AND** refreshes the displayed setting from the server response.

#### Scenario: Member reads team daily budget

- **GIVEN** the active workspace is a team
- **AND** the current user is a regular member
- **WHEN** the team settings panel is opened
- **THEN** the frontend displays the daily budget
- **AND** disables budget editing.
