# Tool Governance Spec

## ADDED Requirements

### Requirement: Tool calls are audited by run

Every governed tool call MUST create an audit record that can be associated with a run, thread, and user when those identifiers are present in context.

#### Scenario: Successful tool call

- **GIVEN** a chat run with `run_id`, `thread_id`, and `user_id`
- **WHEN** an Agent invokes a governed Eino tool
- **THEN** the system records tool name, risk, arguments, result, status, and duration
- **AND** `/api/tool-audits?run_id=<run_id>` returns the record to the owning user

#### Scenario: Tool call fails

- **GIVEN** a governed Eino tool returns an error
- **WHEN** the Agent receives the tool result
- **THEN** the system records the audit status as `failed`
- **AND** the Agent receives a normalized tool error text instead of an unhandled runtime error

### Requirement: Tool policy classifies risk

The tool runtime MUST classify tools by risk and apply the configured policy before execution.

#### Scenario: Dangerous tool is not allowed

- **GIVEN** a tool is classified as `dangerous`
- **AND** the policy does not allow dangerous tools
- **WHEN** an Agent attempts to call the tool
- **THEN** the runtime blocks execution
- **AND** the block is recorded in the audit log

### Requirement: Tool governance reuses framework interfaces

The implementation MUST reuse Eino `BaseTool`-compatible interfaces for governance and MUST NOT replace the existing ReAct Agent runtime for this capability.

#### Scenario: Existing tool list is mounted

- **GIVEN** an Agent builds a list of Eino tools
- **WHEN** the tools are passed into the ReAct Agent
- **THEN** the list is wrapped by the tool runtime
- **AND** the original tool schema and invocation behavior remain compatible with Eino
