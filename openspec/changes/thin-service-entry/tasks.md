# Tasks

- [x] Add service architecture OpenSpec requirements.
- [x] Document reflection and corrections to the initial design.
- [x] Add `internal/app` service boundary for chat stream orchestration.
- [x] Refactor `/chat/stream` handler to delegate business flow to `ChatService`.
- [x] Preserve current SSE event contract and frontend behavior.
- [x] Run Go tests.
- [x] Run frontend build.
- [x] Identify remaining Phase 2 extraction work.

## Follow-up Tasks

- [x] Extract `ResearchService` from `ChatService`.
- [x] Extract `CheckinService` from `ChatService`.
- [x] Move WeChat and OpenAI-compatible endpoints onto the application service boundary.
- [x] Remove process-local `agent.CheckinThreads` routing.
- [x] Add service-level tests for event capture and route detection.

## Remaining Hardening

- [ ] Add injectable graph/check-in runner interfaces for deeper service tests.
- [ ] Add auth/user ownership checks before exposing thread histories.
- [ ] Persist run events for audit and replay.
