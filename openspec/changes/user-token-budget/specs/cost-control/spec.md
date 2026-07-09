# Cost Control Spec

## ADDED Requirements

### Requirement: User daily token budget

The system MUST allow a user daily token budget to be stored in shared storage and enforced before an Agent run consumes model resources.

#### Scenario: User updates daily token budget

- **GIVEN** a user calls `/api/settings`
- **WHEN** the request includes a positive `daily_token_budget`
- **THEN** the value is persisted in MySQL
- **AND** another pod can read the same value from shared storage.

#### Scenario: User clears daily token budget

- **GIVEN** a user has a daily token budget
- **WHEN** the request includes `daily_token_budget` less than or equal to zero
- **THEN** the stored budget is cleared.

#### Scenario: Run is rejected after budget is exhausted

- **GIVEN** a user has a daily token budget
- **AND** the user's model usage for the current user-local day is greater than or equal to the budget
- **WHEN** the user starts a new Agent run
- **THEN** the service writes a `token_budget_exceeded` error event
- **AND** the run is completed as failed
- **AND** Research, Checkin, and Vision model execution are not started.

#### Scenario: Budget state is shared across pods

- **GIVEN** pod A records model usage for a user in `model_usage_logs`
- **AND** pod B handles the user's next Agent run
- **WHEN** pod B checks the daily token budget
- **THEN** pod B uses the shared MySQL token total and enforces the same budget.
