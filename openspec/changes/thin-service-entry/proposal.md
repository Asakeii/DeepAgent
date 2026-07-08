# Thin Service Entry

## Why

The current service entry points are carrying too much application behavior.

`/chat/stream` currently parses the request, registers reminder delivery, handles image analysis, applies runtime defaults, builds and runs the Eino graph, resumes interrupts, switches to the check-in agent, emits reminder events, and persists final messages. WeChat and OpenAI-compatible endpoints also assemble their own agent execution paths.

This makes the product harder to mature:

- transport protocols cannot share one business execution path;
- handlers are difficult to unit test without running the full graph;
- check-in routing relies on process-local cross-layer state;
- adding auth, auditing, run events, retries, and evaluations would further bloat handlers;
- multi-instance deployment is harder to reason about.

## What Changes

- Introduce an application service layer under `internal/app`.
- Move chat turn orchestration out of HTTP handlers into `ChatService`.
- Keep current SSE event names and payloads stable.
- Keep existing Eino `State + Goto` graph shape unchanged.
- Make handlers responsible for protocol concerns only: decoding, encoding, HTTP/SSE/XML/JSON response behavior.
- Start with a low-risk Phase 1 implementation that extracts orchestration while preserving external behavior.

## Reflection and Corrections

The design document intentionally sketches a mature target architecture, but several parts should not be implemented all at once.

### Correction: Do not introduce microservices now

The project should remain a modular monolith. Splitting Research, Check-in, and Reminder into separate deployable services would add RPC, distributed tracing, deployment, and consistency costs before the boundaries are proven.

### Correction: Do not force complete dependency injection in Phase 1

Global infrastructure variables such as `infra.DB`, `infra.RDB`, and `infra.ChatModel` reduce testability, but replacing all of them immediately would create a large refactor with little product value. Phase 1 should only introduce the application layer and keep existing infra access where needed.

### Correction: Do not persist every run event immediately

Run/event persistence is useful for audit, replay, and evaluation, but adding `runs` and `run_events` while moving handler logic would increase migration and compatibility risk. The first implementation should normalize event emission in code and leave durable event storage for a later change.

### Correction: Do not change the frontend event contract

The frontend already depends on `agent`, `plan`, `interrupt`, `tool_calls`, `tool_call_result`, `message_chunk`, `final_message`, `reminder_scheduled`, `reminder`, `message`, and `error`. The service-layer extraction must preserve this contract.

### Correction: Check-in routing removal may need a second phase

The current route signal uses `agent.CheckinThreads`. Removing it cleanly requires either final state inspection from Eino or moving check-in into the graph as a first-class node. Phase 1 may keep behavior compatible while isolating the usage inside `ChatService`; a later phase should remove the process-local map.

## Out Of Scope

- Rewriting the Eino graph.
- Changing frontend event names or payload shape.
- Adding user authentication.
- Adding run event database tables.
- Replacing MySQL or Redis.
- Converting the modular monolith into microservices.

## Rollout Plan

1. Add OpenSpec requirements for thin service entries.
2. Add `internal/app` with a `ChatService` and small event writer boundary.
3. Refactor `/chat/stream` to delegate application orchestration to `ChatService`.
4. Keep WeChat and OpenAI-compatible endpoints unchanged in the first code slice unless the extraction is already stable.
5. Run Go tests and frontend build.
6. Follow up with `ResearchService`, `CheckinService`, and `ReminderService` extraction.
