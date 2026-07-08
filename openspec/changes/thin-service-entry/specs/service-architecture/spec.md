# Service Architecture Delta

## ADDED Requirements

### Requirement: Thin Transport Handlers

Transport handlers SHALL delegate application orchestration to an application service layer instead of directly coordinating agent runtimes, persistence, and reminder delivery.

#### Scenario: Chat stream handler delegates orchestration

- **WHEN** `/chat/stream` receives a valid request
- **THEN** the handler SHALL decode the request and create an event writer
- **AND** it SHALL delegate chat turn execution to an application service
- **AND** it SHALL NOT directly build the Eino graph

#### Scenario: Handler preserves protocol responsibility

- **WHEN** an application service emits an event
- **THEN** the transport handler SHALL write that event using the transport protocol
- **AND** HTTP-specific behavior such as headers and response status SHALL remain in the handler

### Requirement: Stable Stream Contract During Extraction

Service-layer extraction SHALL preserve the existing stream event contract.

#### Scenario: Existing frontend continues to work

- **WHEN** a research, check-in, image, interrupt, or reminder flow runs through `/chat/stream`
- **THEN** the backend SHALL continue to emit the existing event names and payload fields expected by the frontend

### Requirement: Application Service Boundary

The application layer SHALL expose chat-turn orchestration as a reusable boundary that can later be shared by SSE, WeChat, OpenAI-compatible, and CLI adapters.

#### Scenario: Chat turn service receives model request

- **WHEN** a transport adapter passes a chat request to the application layer
- **THEN** the application layer SHALL apply runtime defaults, select the correct agent path, run the agent flow, and emit normalized events

### Requirement: Incremental Migration

The architecture migration SHALL be incremental and SHALL NOT require replacing the existing Eino graph, storage engines, or frontend contract in the first phase.

#### Scenario: First phase implementation

- **WHEN** the first phase is complete
- **THEN** `/chat/stream` SHALL use the application service boundary
- **AND** existing Go tests and frontend build SHALL pass
- **AND** deeper extractions such as durable run events and removal of process-local check-in routing MAY remain as later tasks
