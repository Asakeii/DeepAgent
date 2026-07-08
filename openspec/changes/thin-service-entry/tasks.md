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

- [ ] Extract `ResearchService` from `ChatService`.
- [ ] Extract `CheckinService` from `ChatService`.
- [ ] Move WeChat and OpenAI-compatible endpoints onto the application service boundary.
- [ ] Remove process-local `agent.CheckinThreads` routing.
- [ ] Add service-level tests with fake event writers and fake agent runners.
