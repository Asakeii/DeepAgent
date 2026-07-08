# Run Control Spec

## ADDED Requirements

### Requirement: Agent run timeout is configurable

The service MUST support a configurable whole-run timeout for Agent execution.

#### Scenario: Timeout is disabled

- **GIVEN** `setting.run_timeout_seconds` is omitted or set to `0`
- **WHEN** a chat run starts
- **THEN** the service does not add a whole-run timeout beyond the request context

#### Scenario: Timeout is configured

- **GIVEN** `setting.run_timeout_seconds` is a positive number
- **WHEN** a chat run starts
- **THEN** the service creates a deadline context for that run
- **AND** passes the deadline context to research, check-in, and tool execution

### Requirement: Timed-out runs produce a clear user-visible error

The service MUST convert whole-run deadline expiry into a clear stream error.

#### Scenario: Research run exceeds timeout

- **GIVEN** a research run exceeds the configured run timeout
- **WHEN** the run context reaches its deadline
- **THEN** the service emits an error event with `finish_reason=timeout`
- **AND** the run is completed as failed through existing run completion behavior

