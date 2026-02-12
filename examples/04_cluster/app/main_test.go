package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func setupTestFile(t *testing.T, todos []Todo) {
	t.Helper()
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(todosFile, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func cleanupTestFile(t *testing.T) {
	t.Helper()
	os.Remove(todosFile)
}

func TestLoadTodos_NoFile(t *testing.T) {
	cleanupTestFile(t)
	defer cleanupTestFile(t)

	todos, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}
}

func TestLoadTodos_WithFile(t *testing.T) {
	defer cleanupTestFile(t)

	expected := []Todo{
		{ID: 1, Title: "Test todo", Done: false, CreatedAt: time.Now()},
	}
	setupTestFile(t, expected)

	todos, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "Test todo" {
		t.Fatalf("expected title 'Test todo', got '%s'", todos[0].Title)
	}
}

func TestSaveTodos(t *testing.T) {
	defer cleanupTestFile(t)

	todos := []Todo{
		{ID: 1, Title: "Save test", Done: false, CreatedAt: time.Now()},
		{ID: 2, Title: "Another", Done: true, CreatedAt: time.Now()},
	}

	if err := saveTodos(todos); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(loaded))
	}
	if loaded[1].Done != true {
		t.Fatal("expected second todo to be done")
	}
}

func TestNextID(t *testing.T) {
	todos := []Todo{
		{ID: 1}, {ID: 5}, {ID: 3},
	}
	if got := nextID(todos); got != 6 {
		t.Fatalf("expected nextID=6, got %d", got)
	}
}

func TestNextID_Empty(t *testing.T) {
	if got := nextID(nil); got != 1 {
		t.Fatalf("expected nextID=1, got %d", got)
	}
}

func TestCmdAdd(t *testing.T) {
	cleanupTestFile(t)
	defer cleanupTestFile(t)

	if err := cmdAdd([]string{"Buy", "groceries"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todos, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "Buy groceries" {
		t.Fatalf("expected 'Buy groceries', got '%s'", todos[0].Title)
	}
	if todos[0].Done {
		t.Fatal("new todo should not be done")
	}
}

func TestCmdAdd_NoArgs(t *testing.T) {
	if err := cmdAdd(nil); err == nil {
		t.Fatal("expected error for empty args")
	}
}

func TestCmdDone(t *testing.T) {
	defer cleanupTestFile(t)
	setupTestFile(t, []Todo{
		{ID: 1, Title: "Test", Done: false, CreatedAt: time.Now()},
	})

	if err := cmdDone([]string{"1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todos, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !todos[0].Done {
		t.Fatal("todo should be marked done")
	}
}

func TestCmdDone_NotFound(t *testing.T) {
	defer cleanupTestFile(t)
	setupTestFile(t, []Todo{})

	if err := cmdDone([]string{"99"}); err == nil {
		t.Fatal("expected error for missing todo")
	}
}

func TestCmdRemove(t *testing.T) {
	defer cleanupTestFile(t)
	setupTestFile(t, []Todo{
		{ID: 1, Title: "Keep", Done: false, CreatedAt: time.Now()},
		{ID: 2, Title: "Remove me", Done: false, CreatedAt: time.Now()},
	})

	if err := cmdRemove([]string{"2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	todos, err := loadTodos()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "Keep" {
		t.Fatalf("wrong todo remaining: %s", todos[0].Title)
	}
}

func TestCmdRemove_NotFound(t *testing.T) {
	defer cleanupTestFile(t)
	setupTestFile(t, []Todo{})

	if err := cmdRemove([]string{"99"}); err == nil {
		t.Fatal("expected error for missing todo")
	}
}
