package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	todo "github.com/example/todo"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: todo <command> [args]

Commands:
  add <title>         Add a new todo item
  list [status]       List items (optional filter: pending|in_progress|done)
  done <id>           Mark an item as done
  start <id>          Mark an item as in progress
  delete <id>         Delete an item
  edit <id> <title>   Rename an item
  search <query>      Search items by title substring
  stats               Show counts by status
  export              Output all items as CSV
  help                Show this message
`)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	store := todo.NewStore(todo.DefaultFile)
	if err := store.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading todos: %v\n", err)
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo add <title>")
			os.Exit(1)
		}
		title := todo.ParseAddTitle(os.Args[2:])
		item := store.Add(title)
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added #%d: %s\n", item.ID, item.Title)

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
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tTITLE")
		for _, item := range items {
			fmt.Fprintf(w, "%d\t%s\t%s\n", item.ID, item.Status, item.Title)
		}
		w.Flush()

	case "done":
		id := requireID()
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
		if err := store.Edit(id, newTitle); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Renamed #%d to %q.\n", id, newTitle)

	case "search":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: todo search <query>")
			os.Exit(1)
		}
		query := todo.ParseAddTitle(os.Args[2:])
		items := store.Search(query)
		if len(items) == 0 {
			fmt.Printf("No items matching %q.\n", query)
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tTITLE")
		for _, item := range items {
			fmt.Fprintf(w, "%d\t%s\t%s\n", item.ID, item.Status, item.Title)
		}
		w.Flush()

	case "stats":
		stats := store.Stats()
		total := 0
		for _, c := range stats {
			total += c
		}
		fmt.Printf("Total:       %d\n", total)
		fmt.Printf("Pending:     %d\n", stats[todo.StatusPending])
		fmt.Printf("In Progress: %d\n", stats[todo.StatusInProgress])
		fmt.Printf("Done:        %d\n", stats[todo.StatusDone])

	case "help":
		usage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
	}
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
