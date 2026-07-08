# Security Delta

## ADDED Requirements

### Requirement: Thread Ownership

Threads SHALL be owned by a user and ownership SHALL be stored in shared persistent storage.

#### Scenario: New chat turn creates owned thread

- **WHEN** a chat request starts with a new `thread_id`
- **THEN** the service SHALL ensure a `threads` record exists
- **AND** the record SHALL bind that thread to the resolved user id

#### Scenario: Existing thread belongs to another user

- **WHEN** a user attempts to run a chat/check-in turn against a thread owned by another user
- **THEN** the service SHALL reject the request before writing messages or invoking tools

### Requirement: User-Scoped Reads

Read APIs SHALL only return resources owned by the resolved user.

#### Scenario: List sessions

- **WHEN** a caller requests `/api/sessions`
- **THEN** the service SHALL return only threads owned by the resolved user

#### Scenario: Load messages

- **WHEN** a caller requests `/api/messages?thread_id=<id>`
- **THEN** the service SHALL verify thread ownership before returning messages

#### Scenario: Read reminders

- **WHEN** a caller lists, toggles, or cancels reminders for a thread
- **THEN** the service SHALL verify thread ownership before reading or mutating reminders

#### Scenario: Read run events

- **WHEN** a caller requests `/api/run-events?run_id=<id>`
- **THEN** the service SHALL verify run ownership before returning events
