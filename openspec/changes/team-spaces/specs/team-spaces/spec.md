## ADDED Requirements

### Requirement: Team spaces are persisted in shared storage

The system MUST persist team spaces and team membership in MySQL so all service replicas enforce the same collaboration boundary.

#### Scenario: Create team

- **GIVEN** an authenticated user
- **WHEN** the user creates a team
- **THEN** the system stores a `teams` record
- **AND** stores the creator as an `owner` in `team_members`.

#### Scenario: List teams

- **GIVEN** a user belongs to one or more teams
- **WHEN** the user requests `/api/teams`
- **THEN** the response includes only teams where the user is a member.

### Requirement: Team membership controls shared threads

Team threads MUST be readable by team members and rejected for non-members.

#### Scenario: Create team thread

- **GIVEN** a chat request includes `team_id`
- **AND** the requesting user is a team member
- **WHEN** the chat run starts
- **THEN** the created thread is stored with that `team_id`.

#### Scenario: Reject non-member team thread

- **GIVEN** a chat request includes `team_id`
- **AND** the requesting user is not a team member
- **WHEN** the chat run starts
- **THEN** the service rejects the run before Agent execution.

#### Scenario: Team member reads thread messages

- **GIVEN** a thread belongs to a team
- **AND** another user is a member of that team
- **WHEN** the member requests messages for the thread
- **THEN** the service allows access.

### Requirement: Team members can discover shared work

Session and artifact reads MUST include resources in teams where the user is a member.

#### Scenario: List team sessions

- **GIVEN** a user is a member of a team
- **AND** the team has threads
- **WHEN** the user requests `/api/sessions`
- **THEN** the response includes matching team threads with `team_id`.

#### Scenario: Read team artifact

- **GIVEN** an artifact belongs to a team thread
- **AND** the requester is a member of that team
- **WHEN** the requester reads or exports the artifact
- **THEN** the service returns the artifact.
