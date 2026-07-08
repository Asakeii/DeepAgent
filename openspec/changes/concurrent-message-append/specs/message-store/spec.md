# Message Store Spec

## ADDED Requirements

### Requirement: Message append is safe under concurrent writers

The message store MUST avoid read-before-write sequence allocation that can race across pods.

#### Scenario: Concurrent appends in one thread

- **GIVEN** multiple requests append messages to the same `thread_id` concurrently
- **WHEN** the messages are committed
- **THEN** each committed message has a distinct `turn_idx`
- **AND** recent-message queries can order messages deterministically

### Requirement: Message table initialization is idempotent

The service startup MUST ensure the messages table exists with a `BIGINT` turn index.

#### Scenario: Server starts with an existing database

- **GIVEN** the database already has a `messages` table
- **WHEN** the service initializes storage
- **THEN** startup ensures `turn_idx` is compatible with the auto-increment message id type
