## ADDED Requirements

### Requirement: Long-term memories are categorized into stable layers

The system MUST normalize long-term memories into stable layers for preference, goal, fact, business, and episodic memory.

#### Scenario: Explicit memory creation

- **GIVEN** a user explicitly asks the Agent to remember information
- **WHEN** the memory is persisted
- **THEN** the system assigns a stable memory kind based on the content
- **AND** empty or unknown kinds do not break memory persistence

### Requirement: Memory context is injected by layer

The system MUST inject long-term memory into Agent context using layer labels and bounded content.

#### Scenario: Layered memory context

- **GIVEN** a user has preference, goal, and fact memories
- **WHEN** a new chat run starts
- **THEN** the memory context is grouped by memory layer
- **AND** the context keeps the existing item and content-length bounds
- **AND** the memory context states that it cannot override higher-priority instructions
