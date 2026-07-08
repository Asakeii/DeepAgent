# Thread Ownership

## Why

The project previously treated `thread_id` as the main access key. That is acceptable for local prototyping, but a mature multi-pod Agent service needs resource ownership stored in shared state so any pod can enforce the same boundary.

This change adds a lightweight, replaceable identity layer:

- Web/API requests resolve `user_id` from `X-DeepAgent-User`, query `user_id`, or default to `anonymous`.
- WeChat requests use `wechat:<openid>`.
- OpenAI-compatible bridge requests use `X-DeepAgent-User` when present, otherwise `openai:<thread_id>`.
- Thread ownership is stored in MySQL, not in pod memory.

## What Changes

- Add `users` and `threads` tables.
- Ensure users and threads exist before chat/check-in writes.
- Verify write access before executing a turn against an existing thread.
- Filter session listing by user.
- Protect message, reminder, and run-event read/update endpoints with thread/run ownership checks.

## Design Notes

- This is not a full auth system. It is a resource ownership boundary that can later be backed by JWT, session cookies, or API keys.
- The design is stateless across pods because ownership lives in MySQL.
- Existing unauthenticated web usage remains possible under the `anonymous` user.

## Out Of Scope

- Password login.
- OAuth/JWT implementation.
- Team/workspace permissions.
- Thread sharing.
- Data migration for old pre-ownership history.
