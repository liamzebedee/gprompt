package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	todo "github.com/example/todo"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: todo [--file <path>] <command> [args]

Global flags:
  --file <path>       Use a specific todo file (default: %s)

Commands:
  add [--priority low|medium|high] [--due YYYY-MM-DD] <title>
                      Add a new todo item
  list [status]       List items (optional filter: pending|in_progress|done)
  done <id>           Mark an item as done
  start <id>          Mark an item as in progress
  delete <id>         Delete an item
  edit <id> <title>   Rename an item
  priority <id> <low|medium|high|none>
                      Set or clear an item's priority
  due <id> <YYYY-MM-DD|none>
                      Set or clear an item's due date
  search <query>      Search items by title substring
  stats               Show counts by status
  sort <field>        Sort items by: priority, due, status, created
  clear               Remove all completed items
  undo                Revert the last change
  export              Output all items as CSV
  help                Show this message
`, todo.DefaultFile)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	// Parse global --file flag before the command.
	args := os.Args[1:]
	file := todo.DefaultFile
	if len(args) >= 2 && args[0] == "--file" {
		file = args[1]
		args = args[2:]
	}
	if len(args) == 0 {
		usage()
	}

	store := todo.NewStore(file)
	if err := store.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading todos: %v\n", err)
		os.Exit(1)
	}

	color := todo.ColorEnabled()
	cmd := args[0]
	// Rewrite os.Args so sub-command handlers see the right positional args.
	os.Args = append([]string{os.Args[0], cmd}, args[1:]...)

	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo add [--priority low|medium|high] [--due YYYY-MM-DD] <title>")
			os.Exit(1)
		}
		args := os.Args[2:]
		var priority todo.Priority
		var due todo.DueDate
		// Parse optional flags in any order before the title.
		for len(args) >= 2 {
			if args[0] == "--priority" {
				priority = todo.Priority(args[1])
				if !todo.ValidPriority(priority) || priority == "" {
					fmt.Fprintf(os.Stderr, "Invalid priority: %q (valid values: low, medium, high)\n", args[1])
					os.Exit(1)
				}
				args = args[2:]
			} else if args[0] == "--due" {
				var err error
				due, err = todo.ParseDueDate(args[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				args = args[2:]
			} else {
				break
			}
		}
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: todo add [--priority low|medium|high] [--due YYYY-MM-DD] <title>")
			os.Exit(1)
		}
		title := todo.ParseAddTitle(args)
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		item, err := store.AddFull(title, priority, due)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		suffix := ""
		if priority != "" {
			suffix += fmt.Sprintf(" [%s]", item.Priority)
		}
		if item.DueDate.Valid {
			suffix += fmt.Sprintf(" (due %s)", item.DueDate)
		}
		fmt.Printf("Added #%d: %s%s\n", item.ID, item.Title, suffix)

	case "list":
		var filter todo.Status
		if len(os.Args) >= 3 {
			filter = todo.Status(os.Args[2])
		}
		items, err := store.List(filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(items) == 0 {
			fmt.Println("No items.")
			return
		}
		printItems(items, color)

	case "done":
		id := requireID()
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.SetStatus(id, todo.StatusDone); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Marked #%d as done.\n", id)

	case "start":
		id := requireID()
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.SetStatus(id, todo.StatusInProgress); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Marked #%d as in progress.\n", id)

	case "delete":
		id := requireID()
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.Delete(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted #%d.\n", id)

	case "edit":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: todo edit <id> <new-title>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid ID: %s\n", os.Args[2])
			os.Exit(1)
		}
		newTitle := todo.ParseAddTitle(os.Args[3:])
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.Edit(id, newTitle); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Renamed #%d to %q.\n", id, newTitle)

	case "priority":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: todo priority <id> <low|medium|high|none>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid ID: %s\n", os.Args[2])
			os.Exit(1)
		}
		priStr := os.Args[3]
		var pri todo.Priority
		if priStr == "none" {
			pri = todo.PriorityNone
		} else {
			pri = todo.Priority(priStr)
			if !todo.ValidPriority(pri) || pri == "" {
				fmt.Fprintf(os.Stderr, "Invalid priority: %q (valid values: low, medium, high, none)\n", priStr)
				os.Exit(1)
			}
		}
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.SetPriority(id, pri); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		if pri == "" {
			fmt.Printf("Cleared priority on #%d.\n", id)
		} else {
			fmt.Printf("Set #%d priority to %s.\n", id, pri)
		}

	case "search":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo search <query>")
			os.Exit(1)
		}
		query := todo.ParseAddTitle(os.Args[2:])
		items, err := store.Search(query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(items) == 0 {
			fmt.Printf("No items matching %q.\n", query)
			return
		}
		printItems(items, color)

	case "stats":
		stats := store.Stats()
		total := 0
		for _, c := range stats {
			total += c
		}
		fmt.Printf("%s %d\n", todo.ColorLabel("Total:      ", color), total)
		fmt.Printf("%s %d\n", todo.ColorStatus(todo.StatusPending, color)+":    ", stats[todo.StatusPending])
		fmt.Printf("%s %d\n", todo.ColorStatus(todo.StatusInProgress, color)+": ", stats[todo.StatusInProgress])
		fmt.Printf("%s %d\n", todo.ColorStatus(todo.StatusDone, color)+":        ", stats[todo.StatusDone])

	case "due":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: todo due <id> <YYYY-MM-DD|none>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid ID: %s\n", os.Args[2])
			os.Exit(1)
		}
		dateStr := os.Args[3]
		var due todo.DueDate
		if dateStr == "none" {
			due = todo.DueDate{} // Valid=false clears the date
		} else {
			due, err = todo.ParseDueDate(dateStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.SetDueDate(id, due); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		if due.Valid {
			fmt.Printf("Set #%d due date to %s.\n", id, due)
		} else {
			fmt.Printf("Cleared due date on #%d.\n", id)
		}

	case "sort":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo sort <priority|due|status|created>")
			os.Exit(1)
		}
		field := todo.SortField(os.Args[2])
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		if err := store.Sort(field); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Sorted items by %s.\n", field)
		items, _ := store.List("")
		if len(items) > 0 {
			printItems(items, color)
		}

	case "clear":
		if err := store.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating undo snapshot: %v\n", err)
			os.Exit(1)
		}
		removed := store.ClearDone()
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		if removed == 0 {
			fmt.Println("No completed items to clear.")
		} else {
			fmt.Printf("Cleared %d completed item(s).\n", removed)
		}

	case "undo":
		if err := store.Undo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Undo successful.")
		items, _ := store.List("")
		if len(items) > 0 {
			printItems(items, color)
		}

	case "export":
		if err := store.Export(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting: %v\n", err)
			os.Exit(1)
		}

	case "help":
		usage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
	}
}

func printItems(items []todo.Item, color bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, todo.ColorLabel("ID\tSTATUS\tPRIORITY\tDUE\tTITLE", color))
	for _, item := range items {
		dueStr := "-"
		if item.DueDate.Valid {
			dueStr = todo.ColorDueDate(item.DueDate, color)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			item.ID,
			todo.ColorStatus(item.Status, color),
			todo.ColorPriority(item.Priority, color),
			dueStr,
			item.Title,
		)
	}
	w.Flush()
}

func requireID() int {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: todo %s <id>\n", os.Args[1])
		os.Exit(1)
	}
	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid ID: %s\n", os.Args[2])
		os.Exit(1)
	}
	return id
}
