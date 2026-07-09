# Cost Control Spec

## ADDED Requirements

### Requirement: Configurable model price table

The system MUST allow model token prices to be configured without hardcoding provider prices in code.

#### Scenario: Model prices are loaded from configuration

- **GIVEN** the service starts with `model.prices` configured
- **WHEN** code requests the price table
- **THEN** each configured model exposes input, output, cached input, and reasoning token prices
- **AND** the unit is USD per one million tokens.

#### Scenario: Missing model price fails cost budget checks

- **GIVEN** a cost budget is configured
- **AND** historical usage contains a model that is not present in `model.prices`
- **WHEN** the service checks the daily cost budget
- **THEN** the check fails closed
- **AND** the run is rejected with a budget check failure instead of bypassing cost control.

### Requirement: User daily cost budget

The system MUST allow a user daily cost budget to be stored in shared storage and enforced before an Agent run consumes model resources.

#### Scenario: User updates daily cost budget

- **GIVEN** a user calls `/api/settings`
- **WHEN** the request includes a positive `daily_cost_budget_micros`
- **THEN** the value is persisted in MySQL
- **AND** another pod can read the same value from shared storage.

#### Scenario: User clears daily cost budget

- **GIVEN** a user has a daily cost budget
- **WHEN** the request includes `daily_cost_budget_micros` less than or equal to zero
- **THEN** the stored budget is cleared.

#### Scenario: Run is rejected after user cost budget is exhausted

- **GIVEN** a user has a daily cost budget
- **AND** the user's model usage for the current user-local day is greater than or equal to the budget after applying `model.prices`
- **WHEN** the user starts a new Agent run
- **THEN** the service writes a `cost_budget_exceeded` error event
- **AND** the run is completed as failed
- **AND** Research, Checkin, and Vision model execution are not started.

### Requirement: Team daily cost budget

The system MUST allow a team daily cost budget to be stored in shared storage and enforced for team threads before an Agent run consumes model resources.

#### Scenario: Team manager updates daily cost budget

- **GIVEN** a team owner or admin calls `/api/team-settings`
- **WHEN** the request includes a positive `daily_cost_budget_micros`
- **THEN** the value is persisted in MySQL for the team
- **AND** another pod can read the same value from shared storage.

#### Scenario: Team member reads daily cost budget

- **GIVEN** a user is a team member
- **WHEN** the user calls `/api/team-settings?team_id=<team>`
- **THEN** the service returns the team's daily cost budget.

#### Scenario: Non-manager cannot update daily cost budget

- **GIVEN** a user is not a team owner or admin
- **WHEN** the user calls `PUT /api/team-settings`
- **THEN** the service rejects the request as forbidden.

#### Scenario: Run is rejected after team cost budget is exhausted

- **GIVEN** a team has a daily cost budget
- **AND** the team's model usage for the current user-local day is greater than or equal to the budget after applying `model.prices`
- **WHEN** a team member starts a new team Agent run
- **THEN** the service writes a `cost_budget_exceeded` error event
- **AND** Research, Checkin, and Vision model execution are not started.
