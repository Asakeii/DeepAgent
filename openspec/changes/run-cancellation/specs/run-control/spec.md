# Run Control Spec

## ADDED Requirements

### Requirement: Users can cancel their own running runs

The service MUST provide an authenticated API for canceling a running Agent run owned by the caller.

#### Scenario: Owner cancels a running run

- **GIVEN** a run `r1` belongs to `user_id=u1`
- **AND** `r1` status is `running`
- **WHEN** `u1` posts to `/api/runs/cancel` with `run_id=r1`
- **THEN** the service marks `r1` as `cancelled`
- **AND** records `cancel_requested_at`
- **AND** appends a `run_cancelled` event

#### Scenario: Caller does not own the run

- **GIVEN** a run `r1` belongs to `user_id=u1`
- **WHEN** `user_id=u2` posts to `/api/runs/cancel` with `run_id=r1`
- **THEN** the service rejects the request
- **AND** does not cancel `r1`

### Requirement: Running services observe cancellation through shared state

The application layer MUST observe run cancellation through shared storage rather than process-local state.

#### Scenario: Cancellation is requested from another pod

- **GIVEN** pod A is executing run `r1`
- **AND** pod B handles `/api/runs/cancel` for `r1`
- **WHEN** pod B marks `r1` as `cancelled` in MySQL
- **THEN** pod A detects the shared state change
- **AND** cancels the current run context

#### Scenario: Run completion races with cancellation

- **GIVEN** a run has been marked `cancelled`
- **WHEN** deferred completion attempts to mark the run `succeeded` or `failed`
- **THEN** the run remains `cancelled`

