# Reminder Delivery Spec

## ADDED Requirements

### Requirement: Active reminder delivery supports multiple connections per thread

The in-process reminder connection registry MUST keep multiple active SSE connections for the same thread.

#### Scenario: Multiple tabs are connected

- **GIVEN** two SSE connections are registered for the same `thread_id`
- **WHEN** a reminder fires for that thread
- **THEN** both connections receive the reminder event

#### Scenario: One connection disconnects

- **GIVEN** two SSE connections are registered for the same `thread_id`
- **WHEN** one connection is unregistered
- **THEN** the other connection remains registered
- **AND** future reminder events are still delivered to it
