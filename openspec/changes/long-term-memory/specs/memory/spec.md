# Memory Spec

## ADDED Requirements

### Requirement: Long-term memories are stored separately from messages

The service MUST persist long-term user memories in a dedicated table keyed by user.

#### Scenario: User creates a memory through API

- **GIVEN** a caller resolves to `user_id=u1`
- **WHEN** the caller posts a memory to `/api/memories`
- **THEN** the service stores it with `user_id=u1`

#### Scenario: Caller lists memories

- **GIVEN** a caller resolves to `user_id=u1`
- **WHEN** the caller requests `/api/memories`
- **THEN** only memories for `u1` are returned

### Requirement: Explicit user memory requests are captured

The chat service MUST capture explicit user requests to remember preferences or goals.

#### Scenario: User asks the Agent to remember a preference

- **GIVEN** a user message contains an explicit memory trigger such as `请记住`
- **WHEN** the chat turn is accepted for a thread owned by that user
- **THEN** the service writes a `preference` memory with source `explicit_user_message`

#### Scenario: Ordinary request

- **GIVEN** a user asks a normal research or check-in question
- **WHEN** the chat turn is accepted
- **THEN** the service does not infer or write a long-term memory from that message
