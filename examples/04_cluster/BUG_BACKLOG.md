# Bug Backlog

## Open

## Resolved

- [x] `Swap` does not update `UpdatedAt` timestamps on swapped items — every other mutation operation (`Edit`, `SetNote`, `SetStatus`, `SetPriority`, `SetDueDate`, `AddTag`, `RemoveTag`, `RenameTag`, `BulkDone`) updates `UpdatedAt`, but `Swap` silently changes item positions without recording when the change occurred

- [x] `SetNote` does not trim leading/trailing whitespace — whitespace-only notes are stored as non-empty instead of being treated as cleared, and notes with surrounding spaces keep the accidental whitespace (same class of bug as the `Add`/`Edit` whitespace trim fix)

- [x] `Archive` produces nil `Items` slice when all items are archived — causes `Save` to write `"items": null`, which fails on subsequent `Load` (same class of bug as the `ClearDone` nil-slice fix)

- [x] `Export`/`Import` CSV missing `note` column — export→import round-trip silently drops notes added via `SetNote`

- [x] `list --tag` accepts empty or whitespace-only tag filter without validation — silently returns no items instead of rejecting with an error like `ListByTag` and other validated inputs

- [x] `Upcoming` accepts negative `days` values without error — silently returns empty results instead of rejecting with an error like other validated inputs

- [x] `List` with empty filter returns direct reference to internal `Items` slice — callers can mutate store state; filtered lists return safe copies

- [x] `Export` CSV tests (`TestExportWithItems`, `TestExportIncludesDueDate`) use stale 7-column expected header missing "tags" — tests fail because `Export` now writes 8 columns including tags

- [x] `ClearDone` produces nil `Items` slice when all items are removed — causes `Save` to write `"items": null`, which fails on subsequent `Load`

- [x] `Search` accepts empty or whitespace-only query — returns all items instead of rejecting with error

- [x] `Add` and `Edit` do not trim leading/trailing whitespace from titles — stored as-is with accidental spaces

- [x] Empty or whitespace-only titles accepted by `add` and `edit` — should reject with error
- [x] IDs are not stable after delete — deleted IDs can be reused on next add
- [x] No validation that status filter in `todo list` is a valid status value
- [x] `todo add` with multi-word title requires quoting — should join all remaining args
- [x] `AddFull`/`AddWithPriority` accept invalid priority values — should reject with error like `SetPriority` does
- [x] Tags containing semicolons corrupt CSV export/import round-trip — semicolons are used as tag separator in CSV but not rejected as tag content
- [x] `Import` discards `created_at`/`updated_at` timestamps from CSV — always uses `time.Now()`, so export→import round-trip loses original timestamps
