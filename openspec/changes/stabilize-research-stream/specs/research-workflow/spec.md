# Research Workflow Delta

## ADDED Requirements

### Requirement: Stable Research SSE Contract

The research HTTP endpoint SHALL emit named SSE events with JSON payloads and clients SHALL be able to route behavior by event name.

#### Scenario: Stream text tokens once

- **WHEN** a research run produces assistant text
- **THEN** the server SHALL emit `message_chunk` events for incremental text
- **AND** the server SHALL NOT emit a second generic `message` event containing the same streamed text

#### Scenario: Emit structured plan

- **WHEN** Planner successfully parses a `Plan`
- **THEN** the server SHALL emit a `plan` event containing the complete plan JSON
- **AND** the frontend SHALL render plan steps from that event instead of parsing partial token chunks

#### Scenario: Emit completion

- **WHEN** Reporter completes a research run
- **THEN** the server SHALL emit `final_message` with the final assistant content
- **AND** the server SHALL persist that final content as the assistant history entry

### Requirement: Request Defaults

The research endpoint SHALL apply configured defaults for `max_plan_iterations`, `max_step_num`, and `enable_background_investigation` when the request omits those values.

#### Scenario: Frontend omits plan limits

- **WHEN** the frontend sends a research request without plan limit fields
- **THEN** Planner SHALL receive the configured non-zero defaults

### Requirement: Research History Persistence

The research endpoint SHALL append user requests and final assistant research responses to `messages` using `thread_id`.

#### Scenario: Research run appears in history

- **WHEN** a research run completes with a non-empty `thread_id`
- **THEN** `/api/sessions` SHALL list the thread
- **AND** `/api/messages?thread_id=...` SHALL include the user request and final assistant response

### Requirement: Research Tool Boundaries

Researcher SHALL use web search/fetch tools only. Coder SHALL retain Python/code execution tools. Background investigation SHALL use the same native web search capability as Researcher.

#### Scenario: Background search enabled

- **WHEN** background investigation is enabled
- **THEN** the background node SHALL use native SearXNG-backed search when available
- **AND** failure to search SHALL continue to Planner without blocking the workflow
