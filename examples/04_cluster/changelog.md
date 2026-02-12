# Changelog

## 0.7.0 (`uncommitted`)
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
