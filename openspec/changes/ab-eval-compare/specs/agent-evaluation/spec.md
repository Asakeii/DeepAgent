## ADDED Requirements

### Requirement: Offline A/B eval comparison

The project MUST provide an offline way to compare baseline and candidate eval observations using the same case set.

#### Scenario: Compare two observation sets

- **GIVEN** eval cases
- **AND** baseline observations
- **AND** candidate observations
- **WHEN** the comparison runner executes
- **THEN** it scores both suites
- **AND** reports pass rate delta and metric average deltas.

#### Scenario: Detect regressions

- **GIVEN** a case passes in the baseline suite
- **AND** the same case fails in the candidate suite
- **WHEN** the comparison is produced
- **THEN** the case is listed as a regression.

#### Scenario: CI gate on regressions

- **GIVEN** a maximum allowed regression count
- **WHEN** the candidate has more regressions than allowed
- **THEN** the comparison command exits non-zero.
