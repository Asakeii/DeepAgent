# Research Workflow

## Requirements

### Requirement: Graph-Based Research Orchestration

Research tasks SHALL be routed through the Eino graph using shared state and explicit `Goto` transitions.

#### Scenario: Research task routing

- **WHEN** Coordinator classifies a request as research
- **THEN** it SHALL transition to Planner or Background Investigator according to runtime settings
