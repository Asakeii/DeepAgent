# Reminder Event Cards

## Why

Reminder tasks currently appear as plain assistant text. This makes scheduled tasks feel indistinguishable from normal chat replies, and fired reminders are pushed over SSE without enough structured data for the frontend to render a dedicated card.

## What Changes

- Add a structured reminder event payload shared by scheduled and fired reminder events.
- Emit `reminder_scheduled` when the check-in agent creates a Redis-backed reminder.
- Emit `reminder` with structured reminder data when the scheduler fires a reminder or drains pending reminders.
- Render animated frontend cards for reminder creation and firing instead of plain assistant bubbles.

## Out Of Scope

- Replacing the existing scheduler storage model.
- Fully removing legacy MySQL reminder tools.
- Adding push notifications outside the active SSE session.
