## ADDED Requirements

### Requirement: Checkin model calls record token usage

Checkin ReAct model calls MUST use the same callback-based token usage recording path as research graph model calls.

#### Scenario: Checkin agent generates a response

- **GIVEN** a checkin run has run, thread, and user context
- **WHEN** the Checkin ReAct agent invokes the model and Eino reports token usage
- **THEN** model usage is persisted with the current run, thread, and user identifiers
