# Stabilize Research Stream

## Why

The current research workflow has a sound graph design, but the frontend/backend boundary is unstable:

- model chunks are emitted both by Eino callbacks and by the handler stream loop;
- the frontend parses SSE line-by-line and guesses state from payload fields;
- planner output is streamed as partial JSON, so the frontend cannot reliably render the plan;
- server mode does not apply configured defaults for planning limits;
- research chats are not consistently stored in the message history;
- background investigation and researcher tool boundaries drift from the current native SearXNG search implementation.

## What Changes

- Define a stable SSE event contract for research runs.
- Emit full structured plan events when Planner output parses successfully.
- Use request defaults from configuration when omitted by clients.
- Persist user and final assistant research messages in `messages`.
- Avoid duplicate assistant text by making callbacks the sole streaming content path.
- Keep Researcher focused on web search/fetch and Coder focused on Python/code tools.
- Reuse the native SearXNG search tool for background investigation.

## Out Of Scope

- Reworking the full check-in/reminder architecture.
- Adding a new frontend framework.
- Changing the Eino graph routing model.
