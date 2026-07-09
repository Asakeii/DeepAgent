## ADDED Requirements

### Requirement: Model token usage is persisted by run

The system MUST persist model token usage emitted by Eino callbacks with run, thread, user, agent, and model dimensions.

#### Scenario: Eino callback reports token usage

- **GIVEN** a model callback output contains token usage
- **WHEN** the callback is handled during a run
- **THEN** the system records prompt, completion, total, cached, and reasoning token counts
- **AND** the record is associated with the current run, thread, and user

### Requirement: Run metrics include model usage totals

The run metrics API MUST include aggregated model token usage for the requested user and time window.

#### Scenario: Query run metrics

- **GIVEN** model usage records exist for a user
- **WHEN** the user queries `/api/metrics/runs`
- **THEN** the response includes prompt, completion, total, cached, and reasoning token totals
