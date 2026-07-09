## ADDED Requirements

### Requirement: Vision model usage is persisted when reported

The image analysis path MUST persist VisionModel token usage when the model response includes usage metadata.

#### Scenario: Vision model returns usage

- **GIVEN** an image analysis run has run, thread, and user context
- **AND** the VisionModel response includes token usage
- **WHEN** the food analysis tool handles the response
- **THEN** model usage is persisted with agent `vision`
- **AND** the record is associated with the current run, thread, and user identifiers
