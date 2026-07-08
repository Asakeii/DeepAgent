# Service Architecture Spec

## ADDED Requirements

### Requirement: ChatService depends on runner interfaces

The chat application service MUST orchestrate research, check-in, and reminder streaming through interfaces instead of concrete service types.

#### Scenario: Default construction

- **GIVEN** production code calls `NewChatService`
- **WHEN** the chat service is created
- **THEN** it uses the existing real research, check-in, and reminder implementations

#### Scenario: Test construction

- **GIVEN** a test provides fake research and check-in runners
- **WHEN** the chat service is created through dependency injection
- **THEN** the service can execute routing logic without invoking the real Agent graph

### Requirement: CheckinService delegates Agent execution through a runner

The check-in application service MUST delegate direct Agent execution to a runner adapter.

#### Scenario: Checkin turn

- **GIVEN** a check-in turn is accepted for an owned thread
- **WHEN** the service runs the turn
- **THEN** it calls the configured check-in Agent runner
- **AND** the default runner preserves existing ReAct agent behavior

