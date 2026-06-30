# Project Overview

deepAgent is a Go service built around CloudWeGo Eino graphs. The research workflow routes user requests through coordinator, planner, human feedback, research execution, and reporter nodes, then streams progress to a static frontend over SSE.

## Conventions

- Specs live under `openspec/specs/<capability>/spec.md`.
- Proposed changes live under `openspec/changes/<change-id>/`.
- Use `SHALL` for requirements and keep scenarios observable from HTTP/API behavior.
- Code changes should preserve the existing Eino `State + Goto` graph shape unless the spec requires otherwise.
