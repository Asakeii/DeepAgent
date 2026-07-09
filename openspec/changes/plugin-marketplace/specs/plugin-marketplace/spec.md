## ADDED Requirements

### Requirement: Plugin catalog is backed by configured MCP servers

The system MUST expose configured MCP servers as the plugin catalog without returning server secrets or startup credentials.

#### Scenario: List plugins

- **GIVEN** MCP servers are configured
- **WHEN** a user requests `/api/plugins`
- **THEN** the response lists plugin server names, transport type, enabled state, and discovered tool metadata
- **AND** the response does not include environment variables, headers, or command arguments.

### Requirement: Plugin installs are persisted by user or team scope

Plugin enablement MUST be stored in shared storage so all service replicas enforce the same plugin selection.

#### Scenario: Disable user plugin

- **GIVEN** a configured MCP plugin
- **WHEN** a user posts to `/api/plugin-installs` with `enabled=false`
- **THEN** the disabled state is stored for that user scope.

#### Scenario: Team plugin management requires manager role

- **GIVEN** a team plugin install request includes `team_id`
- **WHEN** the requester is not a team owner or admin
- **THEN** the service rejects the request.

### Requirement: Agent runs filter MCP tools by plugin scope

Agent runtime MUST apply plugin enablement before exposing MCP tools to agent nodes.

#### Scenario: User plugin is disabled

- **GIVEN** a user has disabled an MCP plugin
- **WHEN** a chat run starts for that user
- **THEN** Coder and Background Investigator do not load tools from that plugin.

#### Scenario: Team plugin is disabled

- **GIVEN** a team has disabled an MCP plugin
- **AND** a chat request runs with that `team_id`
- **WHEN** Coder or Background Investigator loads MCP tools
- **THEN** tools from that plugin are omitted for the run.
