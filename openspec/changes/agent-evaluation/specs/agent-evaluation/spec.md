# Agent Evaluation Spec

## ADDED Requirements

### Requirement: Eval cases define expected Agent behavior

The project MUST provide a machine-readable eval case format for expected route, tool calls, forbidden tools, and required final-answer content.

#### Scenario: Routing eval case

- **GIVEN** an eval case with `expected_route`
- **WHEN** the runner compares it with an observation
- **THEN** the runner reports `routing_accuracy`

#### Scenario: Tool eval case

- **GIVEN** an eval case with `expected_tools` and `forbidden_tools`
- **WHEN** the runner compares it with observed tool calls
- **THEN** the runner reports `tool_call_accuracy`

### Requirement: Offline eval runner scores observations

The project MUST provide an offline eval runner that reads cases and observations without requiring a live model call.

#### Scenario: Missing observation

- **GIVEN** an eval case has no matching observation
- **WHEN** the runner evaluates the suite
- **THEN** the case is marked failed

#### Scenario: Regression gate

- **GIVEN** a minimum pass rate
- **WHEN** the suite pass rate is below that threshold
- **THEN** the runner exits with a non-zero status
