# Changelog

## 2.5.0
* Add `group-by-tag` command to display items grouped by their tags (`todo group-by-tag`)
* Add `GroupByTag` method on `Store` that collects items into `TagGroup` slices sorted alphabetically by tag name
* Items with multiple tags appear in each relevant group; untagged items are collected in a group with an empty tag label at the end
* CLI renders untagged group with "Untagged" header and uses color-coded labels
* Add tests for GroupByTag: empty store, basic grouping, multi-tag items, untagged-at-end, all-untagged, alphabetical ordering
* Update BACKLOG: mark `group-by-tag` command as completed

## 2.4.0 (`3797bf0`)
* Fix `SetNote` not trimming whitespace — whitespace-only notes were stored as non-empty instead of being treated as cleared, and notes with surrounding spaces kept accidental whitespace (same class of bug as the `Add`/`Edit` whitespace trim fix)
* Add `Timeline` method on `Store` that groups non-done items by due-date urgency into ordered buckets: Overdue, Today, This Week, Later, and No Due Date
* Add `TimelineBucket` struct for timeline grouping
* Add `GroupByTag` method on `Store` that groups items by tag, sorted alphabetically, with multi-tagged items appearing in each relevant group and untagged items collected separately
* Add `TagGroup` struct for tag-based grouping
* Add tests for `SetNote` whitespace-only clearing and leading/trailing whitespace trimming
* Update BUG_BACKLOG: document resolved `SetNote` whitespace trimming bug

## 2.3.1 (`5137ad2`)
* Fix `Archive` nil-slice bug causing corrupted JSON when all items are archived — `var kept []Item` (nil) replaced with `make([]Item, 0)` so JSON serializes as `"items": []` instead of `"items": null`
* Same bug pattern as the previously fixed `ClearDone` nil-slice issue (1.3.1)
* Update BUG_BACKLOG: document resolved Archive nil-slice bug

## 2.3.0 (`f0baaf3`)
* Add `duplicate` command to clone an existing item as a new pending copy (`todo duplicate <id>`)
* Add `Duplicate` method on `Store` that copies title, priority, due date, tags, and note while assigning a fresh ID and timestamps
* Add `bulk-done` command to mark multiple items as done in one atomic operation (`todo bulk-done <id1> <id2> ...`)
* Add `BulkDone` method on `Store` with duplicate-ID detection and all-or-nothing validation — all IDs are checked before any changes are made
* Create undo snapshot before duplicating and bulk-done for safe rollback
* Add tests for duplicate: basic field copying, status reset to pending, not found, independent tag slices, new timestamps, no-tags case, store count increment
* Add test for archive-all-done persistence: reload after archiving all items succeeds and preserves NextID
* Update BACKLOG: mark `duplicate` command as completed

## 2.2.1 (`2eced6c`)
* Fix Export/Import CSV missing `note` column — export→import round-trips silently dropped notes
* Add `note` as column 7 in CSV format, update `Import` to read it
* Add tests for note CSV round-trip preservation
* Add `ListWithTag` tests
* Update BUG_BACKLOG: document resolved CSV note column bug

## 2.2.0 (`1d9455d`)
* Add `swap` command to exchange the position of two items in the list for manual reordering (`todo swap <id1> <id2>`)
* Add `Swap` method on `Store` with validation for same-ID and not-found cases
* Create undo snapshot before swapping for safe rollback
* Add `note` column to CSV export/import — notes now survive export→import round-trips
* Update CSV header from 8 to 9 columns (`id, title, status, priority, due_date, tags, note, created_at, updated_at`)
* Update `Import` to parse `note` field from column 6 and shift timestamp columns accordingly
* Add tests for swap: basic position exchange, same-ID rejection, not-found error
* Add tests for note CSV round-trip: export includes note column, export→import preserves notes
* Update BACKLOG: mark `swap` command as completed
* Update BUG_BACKLOG: mark `list --tag` empty tag validation bug as resolved

## 2.1.0 (`f174cbc`)
* Show tool argument summaries in TUI iteration view — e.g. `Read(BACKLOG.md)`, `Bash(git status)`, `Grep(TODO)`
* Add `Detail` field to `ConvoMessage` for short tool-input summaries
* Add `toolDetail` helper to extract meaningful fields (file_path, command, pattern, query, url) from streamed tool input JSON
* Accumulate `input_json_delta` and `partial_json` during streaming to build complete tool input for summary extraction
* Change `AppendLiveMessage` to upsert semantics — updates existing messages by ID instead of always appending
* Refactor iteration view: extract `renderConversation` from `renderIteration` for clearer separation of data lookup and rendering
* Suppress empty tool results in conversation display instead of showing blank `⎿` lines
* Handle `content_block_stop` events to finalise tool-use detail and tool-result content via upsert
* Add semicolon-prefixed comment support (`;`) to parser alongside existing `#` comments
* Add `TestParseSemicolonComments` for semicolon comment parsing
* Add `note` command to set, view, or clear a free-text note on an item (`todo note <id> <text>`, `todo note <id> --clear`)
* Add `Note` field to `Item` struct with JSON serialisation support
* Add `SetNote` method on `Store` to set or clear a note with `UpdatedAt` tracking
* Display note in `show` command detail output
* Create undo snapshot before setting/clearing notes for safe rollback
* Add `ListWithTag` method on `Store` combining status and tag filtering with validation in a single call
* Add tests for note: set, clear, not found, persistence, JSON `omitempty` for empty note, undo restore
* Add tests for ListWithTag: empty tag rejection, status+tag filtering, invalid status rejection, no matches
* Update BUG_BACKLOG: mark `list --tag` empty tag validation bug as resolved

## 2.0.0 (`108163d`)
* Add `rename-tag` command to rename a tag across all items (`todo rename-tag <old> <new>`)
* Add `RenameTag` method on `Store` with normalisation (trim + lowercase), semicolon rejection, same-name check, and deduplication when target tag already exists
* Create undo snapshot before renaming for safe rollback
* Add tests for rename-tag: basic rename, deduplication, empty old/new, same name, not found, semicolon rejection, case-insensitive matching
* Add regression test files for edge cases: priority/status case sensitivity, `HasTag` with empty string, `Overdue` excluding today, sort-by-due stability, tag display consistency

## 1.9.0 (`6fe5764`)
* Add `archive` command to move completed items to a separate archive file (`todo archive`), preserving history unlike `clear`
* Add `Archive` method on `Store` that moves done items to a `.archive` JSON file with the same envelope format
* Add `archiveFile` helper to derive archive path from the main store file
* Archive appends to existing archive file, preserving previously archived items
* Sync `NextID` to archive file to prevent ID collisions if archive is ever imported back
* Create undo snapshot before archiving for safe rollback
* Add tests for archive: empty store, no done items, moves only done, appends to existing, preserves item data (priority, due date, tags, IDs)
* Update BACKLOG: mark `archive` command as completed

## 1.8.0 (`711a39b`)
* Improve iteration message rendering: assemble consecutive text fragments into blocks instead of rendering each message individually
* Adopt Claude Code-style tool display: `● ToolName` for tool use, `⎿  result` for tool results
* Add animated braille spinner (`⠋⠙⠹…`) shown while an iteration is still running
* Add `tickMsg` subscription with 300ms interval to drive spinner animation
* Add `SpinFrame` field to `Model` for tracking spinner frame index
* Remove inline duration/timing display from iteration view in favor of cleaner layout

## 1.7.0 (`0f673d9`)
* Refactor TUI: split monolithic `cluster/tui.go` into `cluster/tui/` subpackage with 6 focused source files
* Reduce codebase from 17 files (1989 lines) to 6 files (1133 lines) — 43% reduction
* Simplify focus model to 2 targets (sidebar + input) per spec instead of 3
* Delete `ContentScroll`/`SidebarState` structs in favor of flat `Model` fields
* Simplify `AgentView` to empty placeholder per spec ("reserved for future metadata")
* Redesign `LoopView` as two-column layout (Prompt | Stats) matching spec
* Change scroll behavior to always `ScrollToBottom` with offset measured as lines from bottom
* Update `gcluster` main entry point to use new `tui` subpackage

## 1.6.1 (`2272786`)
* Fix `Upcoming` accepting negative `days` values without error — silently returned empty results instead of rejecting with a validation error
* Change `Upcoming` return signature to `([]Item, error)` and add input validation for negative days
* Update BUG_BACKLOG: mark `Upcoming` negative days bug as resolved

## 1.6.0 (`5c56fae`)
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
