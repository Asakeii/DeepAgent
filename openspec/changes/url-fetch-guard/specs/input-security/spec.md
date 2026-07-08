# Input Security Spec

## ADDED Requirements

### Requirement: External fetch URLs are validated before HTTP requests

The service MUST validate user/model supplied external fetch URLs before issuing HTTP requests.

#### Scenario: Private network URL

- **GIVEN** a URL points to localhost, loopback, private, link-local, or unspecified address
- **WHEN** `web_fetch` or image URL fetching attempts to use it
- **THEN** the service rejects the URL before issuing the HTTP request

#### Scenario: Unsupported scheme

- **GIVEN** a URL uses a scheme other than `http` or `https`
- **WHEN** a fetch path validates it
- **THEN** the service rejects the URL

### Requirement: Host allow and deny lists are configurable

The service MUST support configurable host allow and deny lists for external fetch URLs.

#### Scenario: Allowlist is configured

- **GIVEN** `server.url_allowed_hosts` is non-empty
- **WHEN** a fetch URL host does not match the allowlist
- **THEN** the service rejects the URL

#### Scenario: Denylist match

- **GIVEN** a fetch URL host matches `server.url_denied_hosts`
- **WHEN** a fetch path validates it
- **THEN** the service rejects the URL even if it would otherwise be allowed

