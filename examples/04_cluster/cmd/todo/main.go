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
		title := os.Args[2]
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
		items := store.List(filter)
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
