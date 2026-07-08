# Input Security Spec

## ADDED Requirements

### Requirement: Image inputs are bounded by size and type

The chat service MUST validate direct image inputs before invoking the image analysis runner.

#### Scenario: Valid data URL image

- **GIVEN** a chat request contains a base64 data URL image
- **AND** the decoded image size is within `server.image_max_bytes`
- **AND** the image MIME type is allowed by `server.image_allowed_types`
- **WHEN** the image branch runs
- **THEN** the service may invoke image analysis

#### Scenario: Oversized image

- **GIVEN** a chat request contains a base64 image larger than `server.image_max_bytes`
- **WHEN** the image branch runs
- **THEN** the service rejects the request before invoking image analysis
- **AND** emits an error event with `finish_reason=invalid_image`

#### Scenario: Local file path

- **GIVEN** a chat request provides a local filesystem path in `image_base64`
- **WHEN** the image branch runs
- **THEN** the service rejects the request before invoking image analysis

### Requirement: Image validation failures complete the run as failed

Invalid image input MUST be visible in run observability.

#### Scenario: Invalid image request has run id

- **GIVEN** a chat request has an invalid image input
- **WHEN** the service rejects the image
- **THEN** the error is emitted through the run event writer
- **AND** normal run completion marks the run as failed

