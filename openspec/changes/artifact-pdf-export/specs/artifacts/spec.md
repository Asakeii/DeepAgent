## ADDED Requirements

### Requirement: Artifact PDF export

The system MUST allow a user to export an accessible artifact as a PDF document.

#### Scenario: Export owned artifact as PDF

- **GIVEN** a user can access an artifact
- **WHEN** the user calls `/api/artifact-exports` with `format=pdf`
- **THEN** the service renders the artifact markdown through the existing HTML template
- **AND** invokes the configured PDF renderer
- **AND** returns an `application/pdf` response with a download content disposition.

#### Scenario: Renderer unavailable

- **GIVEN** no PDF renderer command is configured or discoverable
- **WHEN** a user requests `format=pdf`
- **THEN** the service returns a clear renderer error
- **AND** HTML export remains available.

### Requirement: PDF renderer is configurable

PDF rendering MUST be configurable so deployment environments can provide Chrome or Chromium without changing application code.

#### Scenario: Renderer command is configured

- **GIVEN** `server.pdf_renderer_command` and `server.pdf_renderer_args` are configured
- **WHEN** PDF export runs
- **THEN** the service invokes that command with `{{input}}`, `{{input_path}}`, and `{{output}}` placeholders expanded.

#### Scenario: Renderer times out

- **GIVEN** the renderer exceeds `server.pdf_renderer_timeout_seconds`
- **WHEN** PDF export waits for completion
- **THEN** the service aborts the renderer and returns a timeout error.
