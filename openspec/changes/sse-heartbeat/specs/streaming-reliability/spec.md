# Streaming Reliability Spec

## ADDED Requirements

### Requirement: Chat stream emits heartbeat comments

The chat SSE endpoint MUST periodically emit heartbeat comment frames while the HTTP request is active.

#### Scenario: Chat stream stays open

- **GIVEN** `/chat/stream` accepts a request
- **AND** SSE heartbeat is enabled
- **WHEN** the stream remains open longer than the configured interval
- **THEN** the server writes an SSE comment frame
- **AND** the frame does not change the existing application event contract

#### Scenario: Request ends

- **GIVEN** a chat stream heartbeat is running
- **WHEN** the request context is canceled or the handler returns
- **THEN** the heartbeat stops without relying on shared process state

### Requirement: Heartbeat interval is configurable

The server MUST expose a configuration value for SSE heartbeat interval.

#### Scenario: Interval is omitted

- **GIVEN** server config omits `sse_heartbeat_seconds`
- **WHEN** configuration is loaded
- **THEN** the service uses a safe default heartbeat interval

#### Scenario: Interval is disabled

- **GIVEN** server config sets `sse_heartbeat_seconds` to a negative value
- **WHEN** a chat stream starts
- **THEN** heartbeat is not started

