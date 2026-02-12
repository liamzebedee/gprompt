# Changelog

## 1.13.0 (`434a4f0`)
* Add tool details, content pane scrolling, and improve conversation rendering

## 1.12.0 (`3797bf0`)
* Fix SetNote not trimming whitespace — whitespace-only notes stored as non-empty

## 1.11.0 (`5137ad2`)
* Fix Archive nil-slice bug causing corrupted JSON when all items archived

## 1.10.0 (`f0baaf3`)
* Add `todo duplicate <id>` command to clone an existing item

## 1.9.0 (`1d9455d`)
* Add `todo swap <id1> <id2>` command and fix CSV note column
* Fix Export/Import CSV missing note column and add round-trip tests

## 1.8.0 (`f174cbc`)
* Add `todo note <id> <text>` command to set free-text notes on items

## 1.7.0 (`108163d`)
* Add `todo rename-tag <old> <new>` command to rename a tag across all items

## 1.6.0 (`711a39b`)
* Add `todo archive` command to move completed items to a separate archive file
* Improve iteration message rendering: assemble fragments, Claude Code style, animated spinner
* Refactor TUI: split cluster/tui.go into cluster/tui/ subpackage, simplify to match spec

## 1.5.0 (`5c56fae`)
* Add `todo overdue` and `todo upcoming [days]` commands for due date filtering
* Fix Upcoming accepting negative days without error — reject with validation error

## 1.4.0 (`c6ba846`)
* Add `todo show <id>` command to display detailed item info
* Fix Import discarding CSV timestamps, always using time.Now()
* Fix tags containing semicolons corrupting CSV export/import round-trip

## 1.3.0 (`249ef31`)
* Add `todo import` command to load items from CSV files
* Fix List with empty filter returning direct reference to internal Items slice
* Fix Export CSV tests missing "tags" column in expected headers
* Fix ClearDone producing nil Items slice that breaks subsequent Load

## 1.2.0 (`9efc88a`)
* Add `todo undo` command to revert the last change
* Add `todo sort` command to reorder items by priority, due, status, or created date
* Fix Search accepting empty/whitespace-only queries
* Fix Add/Edit not trimming whitespace from titles

## 1.1.0 (`0944c35`)
* Add `todo clear` command to bulk-remove completed items and trim whitespace on titles
* Add due date support to CLI: --due flag, due command, and list display
* Fix AddFull/AddWithPriority accepting invalid priority values
* Fix empty/whitespace title accepted by add and edit commands

## 0.4.0 (`b05b285`)
* Add todo search command to find items by title substring
* Fix ID reuse after delete by persisting a monotonic NextID counter
* Fix: validate status filter in todo list command

## 0.3.0 (`53667d3`)
* Add todo edit command to rename items by ID
* Fix multi-word title bug in todo add: join all remaining args

## 0.2.0 (`60fa77a`)
* Implement Go CLI todo app for 04_cluster example

## 0.1.0 (`10da2f9`)
* Scaffold 04_cluster example with Go CLI todo app
