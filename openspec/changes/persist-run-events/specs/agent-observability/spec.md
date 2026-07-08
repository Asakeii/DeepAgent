# Agent Observability Delta

## ADDED Requirements

### Requirement: Durable Agent Run Records

Every chat turn SHALL create a durable run record that can be inspected after the request completes.

#### Scenario: Chat stream starts a run

- **WHEN** `/chat/stream` receives a valid request
- **THEN** the application layer SHALL create a `runs` record with a `run_id`
- **AND** emitted stream payloads SHALL include that `run_id`

#### Scenario: Run completes successfully

- **WHEN** a chat turn exits without emitting an error event
- **THEN** the run SHALL be marked `succeeded`

#### Scenario: Run fails

- **WHEN** a chat turn emits an `error` event
- **THEN** the run SHALL be marked `failed`
- **AND** the run SHALL retain the error text when available

### Requirement: Durable Run Events

Agent stream events SHALL be persisted for replay and debugging.

#### Scenario: Event is emitted

- **WHEN** the application emits an event such as `agent`, `tool_calls`, `message_chunk`, `final_message`, `reminder`, or `error`
- **THEN** the event SHALL be appended to `run_events` with `run_id`, `thread_id`, event name, agent name when available, and JSON payload

### Requirement: Run Event Read API

The service SHALL expose a read endpoint for run events.

#### Scenario: Query events for a run

- **WHEN** a caller requests `/api/run-events?run_id=<id>`
- **THEN** the service SHALL return persisted events for that run ordered by event id
