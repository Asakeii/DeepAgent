# Agent Observability Spec

## ADDED Requirements

### Requirement: Run logs include stable correlation fields

The application MUST emit structured logs for the main Agent run lifecycle with stable correlation fields.

#### Scenario: Chat run starts

- **GIVEN** a chat run has `run_id`, `thread_id`, and `user_id`
- **WHEN** the run is accepted by ChatService
- **THEN** the service logs a structured record containing `run_id`, `thread_id`, `user_id`, and `mode`

#### Scenario: Chat run completes

- **GIVEN** a chat run reaches terminal status
- **WHEN** completion is persisted
- **THEN** the service logs a structured record containing `run_id`, `thread_id`, `user_id`, `status`, and `duration_ms`

### Requirement: Observability uses standard library logging

The service MUST provide a standard structured logging entry point based on Go's standard library.

#### Scenario: Logger is configured

- **GIVEN** the service process starts
- **WHEN** logging is configured
- **THEN** logs are emitted as JSON records compatible with log aggregation systems

