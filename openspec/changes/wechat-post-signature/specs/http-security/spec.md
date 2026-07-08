# HTTP Security Spec

## ADDED Requirements

### Requirement: WeChat POST callbacks verify platform signature

The WeChat message callback MUST verify `signature`, `timestamp`, and `nonce` before reading the message body or invoking Agent logic.

#### Scenario: Invalid POST signature

- **GIVEN** a WeChat POST callback has an invalid signature
- **WHEN** the callback is received
- **THEN** the server rejects the request
- **AND** does not read the XML message body

#### Scenario: Valid POST signature with invalid XML

- **GIVEN** a WeChat POST callback has a valid signature
- **AND** the body is invalid XML
- **WHEN** the callback is received
- **THEN** signature verification succeeds
- **AND** the server returns the existing safe `success` response without invoking Agent logic
