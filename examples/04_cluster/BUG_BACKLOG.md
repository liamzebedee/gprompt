# Bug Backlog

## Open

- [ ] `Add` and `Edit` do not trim leading/trailing whitespace from titles — stored as-is with accidental spaces

## Resolved

- [x] Empty or whitespace-only titles accepted by `add` and `edit` — should reject with error
- [x] IDs are not stable after delete — deleted IDs can be reused on next add
- [x] No validation that status filter in `todo list` is a valid status value
- [x] `todo add` with multi-word title requires quoting — should join all remaining args
- [x] `AddFull`/`AddWithPriority` accept invalid priority values — should reject with error like `SetPriority` does
