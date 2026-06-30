# Reminder Cards

## Requirements

### Requirement: Structured reminder events

The backend SHALL emit reminder-related SSE events with a structured `reminder` payload containing the reminder id, message, next fire time, recurrence metadata, and a status.

#### Scenario: Reminder scheduled

- **WHEN** the check-in agent successfully creates a Redis-backed reminder
- **THEN** the stream SHALL emit `event: reminder_scheduled`
- **AND** the payload SHALL include `reminder.status = "scheduled"`

#### Scenario: Reminder fired

- **WHEN** a due reminder is delivered through an active or newly reconnected SSE stream
- **THEN** the stream SHALL emit `event: reminder`
- **AND** the payload SHALL include `reminder.status = "fired"`

### Requirement: Animated frontend reminder cards

The frontend SHALL render structured reminder events as dedicated cards rather than plain assistant text.

#### Scenario: Scheduled task card

- **WHEN** the frontend receives `reminder_scheduled`
- **THEN** it SHALL append a transcript card showing the reminder message, next fire time, and whether it repeats
- **AND** the card SHALL use an animated visual state to distinguish it from normal chat bubbles.

#### Scenario: Fired task card

- **WHEN** the frontend receives `reminder`
- **THEN** it SHALL append a transcript card showing the fired reminder message
- **AND** the card SHALL use an active/ringing animation.
