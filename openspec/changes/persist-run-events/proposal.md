# Persist Run Events

## Why

Mature Agent systems need durable run records. Before this change, the service could stream Agent events to the frontend, but a run's timeline was not persisted as a first-class artifact. That made multi-pod operation, replay, audit, debugging, evaluation, and cost analysis harder.

The project already uses Eino callbacks to turn graph/model/tool activity into normalized stream events. Eino also provides checkpoint support for graph interruption. Instead of inventing another runtime, this change reuses those existing callback events and records them into MySQL.

## What Changes

- Add `runs` and `run_events` tables.
- Generate a `run_id` for every chat turn when the client does not provide one.
- Add `run_id` to stream payloads.
- Wrap the existing app `EventWriter` with a recorder that writes events to MySQL and forwards them to the transport writer.
- Mark runs as `succeeded` or `failed` when the chat turn exits.
- Add a read-only `/api/run-events?run_id=...` endpoint for replay/debugging.

## Design Notes

- The design is stateless across pods: run state is stored in MySQL, not in process memory.
- This intentionally reuses Eino callback output and the existing `LoggerCallback` event contract.
- Event recording is best-effort from the user-facing stream perspective; write failures are logged and do not block token streaming.
- Full auth/user ownership checks are still a separate maturity phase.

## References

- Temporal's durable execution model relies on persisted event history so workflows can resume after crashes or server failures: https://docs.temporal.io/temporal
- OpenTelemetry's semantic conventions define a common vocabulary for telemetry data, and its GenAI conventions emphasize traces, metrics, and events for model/tool observability: https://opentelemetry.io/docs/specs/semconv/
- Eino already provides callbacks and checkpoint stores in this project, so this change records the existing callback-derived event stream instead of replacing the framework runtime.

## Out Of Scope

- Token/cost accounting.
- Full distributed tracing.
- User ownership checks.
- Event retention policies.
- UI for run replay.
