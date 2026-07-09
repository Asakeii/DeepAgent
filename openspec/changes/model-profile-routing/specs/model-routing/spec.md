## ADDED Requirements

### Requirement: Named model profiles

The system MUST support named model profiles configured at startup.

#### Scenario: Configure model profiles

- **GIVEN** `model.profiles` defines a profile
- **WHEN** the service initializes models
- **THEN** the profile gets its own chat and plan model instances
- **AND** missing endpoint credentials inherit from the default model config.

### Requirement: User or request can select a model profile

The system MUST allow a model profile to be selected from the request or persisted user settings.

#### Scenario: Request selects profile

- **GIVEN** a request contains `model_profile`
- **WHEN** the Agent run starts
- **THEN** all model calls in that run use the selected profile through request context.

#### Scenario: User setting supplies profile

- **GIVEN** a user has a persisted `model_profile`
- **WHEN** the request does not specify a profile
- **THEN** ChatService applies the persisted profile as the run default.

### Requirement: Unknown model profiles are rejected safely

The system MUST reject unknown model profiles before invoking model execution.

#### Scenario: Unknown profile

- **GIVEN** a request selects a profile that is not configured
- **WHEN** the Agent run starts
- **THEN** the service emits an `invalid_model_profile` error
- **AND** completes the run as failed
- **AND** does not call Research, Checkin, or Vision model execution.
