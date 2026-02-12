package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

const todosFile = "todos.json"

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

func loadTodos() ([]Todo, error) {
	data, err := os.ReadFile(todosFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Todo{}, nil
		}
		return nil, err
	}
	var todos []Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

func saveTodos(todos []Todo) error {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(todosFile, data, 0644)
}

func nextID(todos []Todo) int {
	max := 0
	for _, t := range todos {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}

func cmdAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: todo add <title>")
	}
	title := args[0]
	for _, a := range args[1:] {
		title += " " + a
	}

	todos, err := loadTodos()
	if err != nil {
		return err
	}

	todo := Todo{
		ID:        nextID(todos),
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}
	todos = append(todos, todo)

	if err := saveTodos(todos); err != nil {
		return err
	}
	fmt.Printf("Added: [%d] %s\n", todo.ID, todo.Title)
	return nil
}

func cmdList() error {
	todos, err := loadTodos()
	if err != nil {
		return err
	}
	if len(todos) == 0 {
		fmt.Println("No todos.")
		return nil
	}
	for _, t := range todos {
		status := " "
		if t.Done {
			status = "x"
		}
		fmt.Printf("[%s] %d: %s\n", status, t.ID, t.Title)
	}
	return nil
}

func cmdDone(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: todo done <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %s", args[0])
	}

	todos, err := loadTodos()
	if err != nil {
		return err
	}

	found := false
	for i, t := range todos {
		if t.ID == id {
			todos[i].Done = true
			found = true
			fmt.Printf("Completed: [%d] %s\n", t.ID, t.Title)
			break
		}
	}
	if !found {
		return fmt.Errorf("todo %d not found", id)
	}

	return saveTodos(todos)
}

func cmdRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: todo remove <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %s", args[0])
	}

	todos, err := loadTodos()
	if err != nil {
		return err
	}

	newTodos := make([]Todo, 0, len(todos))
	found := false
	for _, t := range todos {
		if t.ID == id {
			found = true
			fmt.Printf("Removed: [%d] %s\n", t.ID, t.Title)
			continue
		}
		newTodos = append(newTodos, t)
	}
	if !found {
		return fmt.Errorf("todo %d not found", id)
	}

	return saveTodos(newTodos)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: todo <add|list|done|remove> [args...]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "add":
		err = cmdAdd(os.Args[2:])
	case "list":
		err = cmdList()
	case "done":
		err = cmdDone(os.Args[2:])
	case "remove":
		err = cmdRemove(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
