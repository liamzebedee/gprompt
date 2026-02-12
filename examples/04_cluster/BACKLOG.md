# Backlog

## Pending

## Completed

- [x] Add `todo move <id> <position>` command to move an item to a specific position in the list (1-based index, shifts other items accordingly)

- [x] Add `todo group-by-tag` command to display items grouped by their tags (alphabetically sorted, multi-tag items appear in each group, untagged items shown last)

- [x] Add `todo duplicate <id>` command to create a new pending copy of an existing item (preserves title, priority, due date, tags, and note)

- [x] Add `todo swap <id1> <id2>` command to swap the position of two items in the list (for manual reordering)

- [x] Add `todo note <id> <text>` command to set a free-text note on an item (displayed in `show`, clearable with `todo note <id> --clear`)

- [x] Add `todo rename-tag <old> <new>` command to rename a tag across all items (case-insensitive, deduplicates)

- [x] Add `todo archive` command to move completed items to a separate archive file (preserves history unlike `clear`)

- [x] Add `todo overdue` and `todo upcoming [days]` commands to filter items by due date urgency

- [x] Add `todo show <id>` command to display detailed information about a single item

- [x] Add `todo import <file.csv>` command to import items from CSV (round-trips with export)

- [x] Add `todo undo` command to revert the last change (snapshot-based single-level undo)

- [x] Add `todo sort <field>` command to reorder items by priority, due date, status, or creation date

- [x] Add `todo clear` command to bulk-remove all completed items

- [x] Basic CRUD operations (add, list, done, start, delete)
- [x] JSON file persistence
- [x] Status filtering on list command
- [x] Add `todo edit <id> <new-title>` command to rename items
- [x] Add `todo search <query>` to find items by title substring
- [x] Add `todo export` to output items as CSV
- [x] Add priority levels (low, medium, high) with `todo add --priority high "Task"`
- [x] Add due dates to todo items with `todo add --due 2025-03-01 "Task"`
- [x] Add color-coded output for different statuses
- [x] Add `todo stats` command showing counts by status
- [x] Support multiple todo files with `--file` flag
