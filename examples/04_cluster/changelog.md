# Changelog

## 1.6.0
* Add `overdue` command to list non-done items past their due date (`todo overdue`)
* Add `upcoming` command to list non-done items due today or within N days (`todo upcoming [days]`, default 7)
* Add `Overdue` and `Upcoming` methods on `Store` for due-date-based item filtering
* Change `Upcoming` return signature to `([]Item, error)` and reject negative `days` values
* Add tests for overdue/upcoming: empty store, done exclusion, boundary conditions, in-progress items, today-only mode, negative days rejection
* Update BACKLOG: mark overdue/upcoming commands as completed

## 1.5.1 (`4442c36`)
* Fix `Import` discarding `created_at`/`updated_at` timestamps from CSV — previously always used `time.Now()`, so export→import round-trip lost original timestamps
* Parse RFC3339 timestamps from CSV columns 6 and 7, falling back to `time.Now()` if missing or invalid
* Add tests for timestamp preservation on import and export→import round-trip
* Update BUG_BACKLOG: log import timestamp loss bug

## 1.5.0 (`c6ba846`)
* Add `show` command to display detailed info for a single item (`todo show <id>`)
* Shows all fields: ID, title, status, priority, due date, tags, created/updated timestamps
* Add `printItemDetail` helper for formatted single-item output with color support
* Update BACKLOG: mark `show` command as completed

## 1.4.1 (`1e9876d`)
* Fix tags containing semicolons corrupting CSV export/import round-trip — semicolons are the tag separator in CSV, so a tag like `"work;personal"` would silently split into two tags on import
* Reject semicolons in `ValidTag`, `AddTag`, and `AddFullWithTags` with clear error messages
* Add tests for semicolon rejection in `AddTag`, `AddFullWithTags`, `ValidTag`, and export→import tag round-trip
* Update BUG_BACKLOG: mark semicolon-in-tags bug as resolved

## 1.4.0 (`249ef31`)
* Add `import` command to load items from a CSV file (`todo import <file.csv>`), round-tripping with `export`
* Add `Import` method on `Store` that parses CSV rows, validates fields (title, status, priority, due date, tags), and assigns fresh IDs
* Add `tag` and `untag` commands to add/remove tags on items (`todo tag <id> <tag>`, `todo untag <id> <tag>`)
* Add `--tag` flag to `add` command for setting tags at creation time
* Add `--tag` filter to `list` command to show only items with a given tag
* Add `AddFullWithTags` method on `Store` for creating items with title, priority, due date, and tags
* Add `TAGS` column to `list` and `search` table output via `FormatTags` helper
* Display tags in `add` command confirmation output
* Add tests for import: basic parsing, new-ID assignment, empty CSV, bad header, invalid status, invalid priority, empty title, and export→import round-trip
* Update BACKLOG: mark `import` command as completed

## 1.3.3 (`eb11567`)
* Fix `List` with empty filter returning direct reference to internal `Items` slice — now returns a defensive copy, consistent with filtered list behavior
* Rename duplicate `TestExportIncludesTags` to fix compilation
* Add tests for `List` defensive copy behavior and tags functionality

## 1.3.2 (`475762c`)
* Fix `Export` CSV tests (`TestExportWithItems`, `TestExportIncludesDueDate`) using stale 7-column expected header missing "tags" — updated to 8-column header
* Add `TestExportIncludesTags` to validate tags column content in CSV export

## 1.3.1 (`54a5827`)
* Fix `ClearDone` producing nil `Items` slice that breaks subsequent `Load` — use `make([]Item, 0)` instead of `var kept []Item` so JSON serializes as `"items": []` rather than `"items": null`

## 1.3.0 (`b1c4624`)
* Add `undo` command to revert the last change (`todo undo`)
* Add `Snapshot` method on `Store` to save current state to an undo backup file
* Add `Undo` method on `Store` to restore from the last snapshot and remove the backup
* Add automatic snapshots before mutating commands: `add`, `done`, `start`, `delete`, `edit`, `priority`, `due`, `sort`, `clear`

## 1.2.1 (`8c871cd`)
* Mark `Search` empty/whitespace-only query bug as resolved in BUG_BACKLOG (fix was applied in 1.2.0)

## 1.2.0 (`9efc88a`)
* Add `sort` command to reorder items by priority, due date, status, or creation date (`todo sort <field>`)
* Add `Sort` method on `Store` with stable in-place reordering and persistence
* Add `SortField` type with `ValidSortField` helper for field validation
* Change `Search` return signature to `([]Item, error)` and reject empty/whitespace-only queries
* Add tests for sort by priority, due, status, created, invalid field, empty store, persistence, and `ValidSortField`
* Add test for `Search` rejecting empty queries
* Update BACKLOG: mark `sort` command as completed

## 1.1.1 (`96ba19d`)
* Mark whitespace-not-trimmed bug as resolved in BUG_BACKLOG (fix was applied in 1.1.0)

## 1.1.0 (`2648f42`)
* Add `clear` command to bulk-remove all completed items (`todo clear`)
* Add `ClearDone` method on `Store` — removes all done items and returns count removed
* Fix `Add`/`AddFull`/`Edit` to trim leading/trailing whitespace from titles before storing
* Add tests for `ClearDone` (empty store, only-done removed, all done, none done)
* Add tests for whitespace trimming in `Add`, `AddFull`, and `Edit`
* Log whitespace-not-trimmed bug in BUG_BACKLOG as resolved by trimming fix
* Update BACKLOG: mark `clear` command, color-coded output, `stats` command, and `--file` flag as completed

## 1.0.1 (`d4abdc5`)
* Mark `AddFull`/`AddWithPriority` invalid-priority bug as resolved in BUG_BACKLOG

## 1.0.0 (`0944c35`)
* Add `ColorDueDate` helper — overdue dates shown in red+bold, today in yellow
* Validate priority in `AddFull` — reject invalid priority values with descriptive error
* Add tests for `AddFull` and `AddWithPriority` rejecting invalid priorities
* Fix unchecked `(Item, error)` return values across test suite

## 0.9.0 (`1f9bb11`)
* Validate titles in `Add`, `AddWithPriority`, `AddFull`, and `Edit` — reject empty or whitespace-only strings
* Change `Add`, `AddWithPriority`, and `AddFull` return signatures to `(Item, error)` for validation support
* Add tests for `ParseDueDate` (valid, empty, invalid inputs)
* Add tests for empty/whitespace-only title rejection in `AddFull` and `Edit`
* Add tests for `AddFull` with due date, `SetDueDate` (set, clear, not found), due date persistence, and due date JSON round-trip

## 0.8.0 (`4ace21a`)
* Refactor `add` command flag parsing to accept `--priority` and `--due` flags in any order
* Switch `add` command from `AddWithPriority` to `AddFull` to support setting priority and due date together
* Improve `add` output to display both priority and due date when present
* Add `due` command to set or clear an item's due date (`todo due <id> <YYYY-MM-DD|none>`)
* Add `DUE` column to `list` and `search` table output
* Add `ColorDueDate` helper — overdue dates shown in red+bold, today in yellow

## 0.7.0 (`3542b0c`)
* Add `DueDate` field to items with `YYYY-MM-DD` format and JSON serialisation support
* Add `--due` flag to `add` command for setting a due date at creation time
* Add `AddFull` method on `Store` for creating items with title, priority, and due date
* Add `SetDueDate` method on `Store` to update or clear an item's due date
* Add `ParseDueDate` helper to parse and validate date strings
* Add colored terminal output — statuses, priorities, and labels are color-coded when stdout is a TTY
* Add `color.go` with `ColorEnabled`, `ColorStatus`, `ColorPriority`, `ColorLabel`, and `Colorf` helpers
* Add `--file` global flag to specify a custom todo file path
* Add shared `printItems` helper used by `list` and `search` commands (includes priority column)
* Include `due_date` column in CSV export output
* Add `golang.org/x/term` dependency for terminal detection
* Add tests for due date export (with and without due dates)

## 0.6.0 (`2b9ba3b`)
* Add `Priority` field to items with `low`, `medium`, `high` levels
* Add `priority` command to set or clear an item's priority (`todo priority <id> <low|medium|high|none>`)
* Add `--priority` flag to `add` command for setting priority at creation time
* Add `AddWithPriority` method and `SetPriority` method on `Store`
* Add `ValidPriority` helper to check whether a priority value is valid
* Add `export` command to output all items as CSV (`todo export`)
* Include priority column in `list` and `search` output
* Include priority column in CSV export output
* Fix ID reuse bug — IDs are now monotonically increasing even after deletions
* Change persistence format from bare JSON array to `{next_id, items}` envelope (backward-compatible with legacy format)
* Add tests for stable IDs after delete and reload, CSV export (empty, with items, escaping)

## 0.5.0 (`26d51c8`)
* Add `ValidStatus` helper to check whether a status string is valid
* Return error from `Store.List` when given an invalid status filter
* Add tests for `ValidStatus`, invalid/valid list filters

## 0.4.0 (`b05b285`)
* Add `search` command to find items by title substring (`todo search <query>`)
* Add `Store.Search` method with case-insensitive title matching
* Add `ParseAddTitle` helper to join CLI args into a single title string
* Fix `todo add` requiring quotes around multi-word titles — now joins all remaining args automatically

## 0.3.0 (`53667d3`)
* Add `edit` command to rename existing todo items by ID (`todo edit <id> <new-title>`)
* Add `Store.Edit` method with title validation and `UpdatedAt` tracking

## 0.2.0 (`60fa77a`)
* Restructure project from `app/` monolith to Go package with separate `todo` library and `cmd/todo` CLI
* Add `start` command to mark items as in-progress
* Replace `remove` command with `delete`
* Add three-state status model (pending, in_progress, done) replacing boolean done field
* Add `UpdatedAt` timestamp tracking on status changes
* Use tabwriter for formatted list output
* Rewrite tests to use temp files instead of shared state

## 0.1.0 (`initial`)
* Initial implementation of CLI todo app
* Add, list, done, start, delete commands
* JSON file persistence with todos.json
* Status filtering on list command
* Tab-formatted output
