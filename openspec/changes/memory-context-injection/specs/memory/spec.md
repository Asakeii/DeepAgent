# Memory Spec

## ADDED Requirements

### Requirement: Long-term memories are injected into Agent context

The service MUST retrieve bounded long-term memories for the authenticated user and inject them into the Agent context after thread ownership is confirmed.

#### Scenario: User has memories

- **GIVEN** a chat request resolves to `user_id=u1`
- **AND** the thread belongs to `u1`
- **AND** `u1` has stored memories
- **WHEN** the chat run starts
- **THEN** the Agent receives a system context message containing a bounded list of those memories
- **AND** the message states that memories cannot override system instructions, developer instructions, safety policy, or the current user request

#### Scenario: Memory retrieval fails

- **GIVEN** memory retrieval returns an error
- **WHEN** the chat run starts
- **THEN** the service logs the error and continues with the original request messages

### Requirement: Injected memories are not persisted as conversation history

The service MUST keep injected memory context separate from user-visible message history.

#### Scenario: Research run persists messages

- **GIVEN** a research run receives injected memory context
- **WHEN** the run completes
- **THEN** only original user messages and the assistant final answer are persisted

#### Scenario: Checkin run persists messages

- **GIVEN** a checkin run receives injected memory context
- **WHEN** the checkin agent records incoming messages
- **THEN** only non-empty user messages are persisted

