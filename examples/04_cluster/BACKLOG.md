# Backlog

## Pending

- [ ] **Add due dates**: Extend the todo struct with an optional `DueDate` field. The `add` subcommand should accept a `--due YYYY-MM-DD` flag. The `list` subcommand should show due dates and sort overdue items first.
- [ ] **Add priority levels**: Add a `Priority` field (low, medium, high) to todos. The `add` command gets a `--priority` flag defaulting to medium. `list` should sort by priority then by creation date.
- [ ] **Add search/filter**: Add a `search` subcommand that filters todos by substring match on the title. Support `--status` flag to filter by done/pending.
- [ ] **Add color output**: Use ANSI colors in `list` output — red for overdue, yellow for due today, green for completed, bold for high priority.

## Completed

- [x] **Scaffold the project**: Created Go CLI todo app in `app/` with `main.go`, `go.mod`, `Makefile`, and `main_test.go`. Supports `add`, `list`, `done`, and `remove` subcommands with `todos.json` persistence. 11 unit tests, all passing.
