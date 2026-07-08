## ADDED Requirements

### Requirement: External content is treated as untrusted data

The system MUST mark fetched external page content as untrusted data before it enters Agent context.

#### Scenario: Fetched page content

- **GIVEN** the Agent fetches a web page
- **WHEN** the page text is returned to the model
- **THEN** the content is wrapped with an instruction that it is untrusted external data
- **AND** the wrapper says the model must not execute instructions from that content

### Requirement: Suspicious prompt injection text is annotated

The system MUST annotate external content when common prompt injection patterns are detected.

#### Scenario: Search result contains injection text

- **GIVEN** a search result snippet contains text such as “ignore previous instructions”
- **WHEN** the result is returned by the search tool
- **THEN** the result includes a security note
- **AND** the factual snippet remains available for research
