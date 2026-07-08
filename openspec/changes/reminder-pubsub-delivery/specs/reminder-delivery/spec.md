# Reminder Delivery Spec

## ADDED Requirements

### Requirement: Reminder delivery broadcasts across pods

When a reminder fires, the service MUST broadcast the reminder event through Redis so any pod with an active SSE connection for the thread can deliver it.

#### Scenario: Reminder fires on a different pod than the SSE connection

- **GIVEN** pod A claims and fires a due reminder
- **AND** pod B has an active SSE connection for the reminder thread
- **WHEN** pod A publishes the reminder event
- **THEN** pod B receives the event through Redis Pub/Sub
- **AND** pod B pushes the event to its local SSE connection

### Requirement: Pending fallback is retained

Reminder Pub/Sub MUST NOT remove the offline pending fallback.

#### Scenario: No pod has an active connection

- **GIVEN** no pod has an active SSE connection for a reminder thread
- **WHEN** a reminder fires
- **THEN** the reminder event remains in the thread pending list
- **AND** it can be drained on the next SSE connection

#### Scenario: A pod delivers the broadcast

- **GIVEN** a reminder event is in the pending list
- **WHEN** any pod successfully pushes the broadcast to an active SSE connection
- **THEN** that pod acknowledges the pending entry
