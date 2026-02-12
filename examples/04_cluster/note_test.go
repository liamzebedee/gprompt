package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetNote(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "todos.json"))
	item, err := s.Add("Buy milk")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Initially no note.
	got, err := s.Get(item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Note != "" {
		t.Fatalf("expected empty note, got %q", got.Note)
	}

	// Set a note.
	if err := s.SetNote(item.ID, "Remember to get whole milk"); err != nil {
		t.Fatalf("SetNote: %v", err)
	}
	got, _ = s.Get(item.ID)
	if got.Note != "Remember to get whole milk" {
		t.Fatalf("expected note %q, got %q", "Remember to get whole milk", got.Note)
	}

	// Updated timestamp should change.
	if got.UpdatedAt.Equal(got.CreatedAt) {
		t.Fatal("expected UpdatedAt to differ from CreatedAt after SetNote")
	}
}

func TestSetNoteClear(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "todos.json"))
	item, _ := s.Add("Task")
	_ = s.SetNote(item.ID, "Some note")
	_ = s.SetNote(item.ID, "") // clear

	got, _ := s.Get(item.ID)
	if got.Note != "" {
		t.Fatalf("expected note to be cleared, got %q", got.Note)
	}
}

func TestSetNoteNotFound(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "todos.json"))
	err := s.SetNote(999, "note")
	if err == nil {
		t.Fatal("expected error for non-existent item")
	}
}

func TestNotePersistence(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "todos.json")

	// Create store, add item with note, save.
	s := NewStore(file)
	item, _ := s.Add("Persist me")
	_ = s.SetNote(item.ID, "This is my note")
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload and verify note persists.
	s2 := NewStore(file)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := s2.Get(item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Note != "This is my note" {
		t.Fatalf("expected persisted note %q, got %q", "This is my note", got.Note)
	}
}

func TestNoteOmittedFromJSONWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "todos.json")

	s := NewStore(file)
	s.Add("No note item")
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(file)
	// The "note" field should not appear in JSON when empty (omitempty).
	if contains := string(data); len(data) > 0 {
		if got := contains; got != "" {
			// Just check the key isn't present.
			for _, line := range splitLines(contains) {
				if line == `      "note":` || line == `      "note": ""` {
					t.Fatal("expected note field to be omitted from JSON when empty")
				}
			}
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestSetNoteWhitespaceOnly(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "todos.json"))
	item, _ := s.Add("Task")

	// Setting a whitespace-only note should be treated the same as clearing
	// the note — it contains no useful content, just like whitespace-only
	// titles are rejected by Add/Edit.
	for _, ws := range []string{"   ", "\t", " \t\n "} {
		if err := s.SetNote(item.ID, ws); err != nil {
			t.Fatalf("SetNote(%q): unexpected error: %v", ws, err)
		}
		got, _ := s.Get(item.ID)
		if got.Note != "" {
			t.Errorf("SetNote(%q): expected note to be cleared (empty), got %q", ws, got.Note)
		}
	}
}

func TestSetNoteTrimsWhitespace(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "todos.json"))
	item, _ := s.Add("Task")

	// Notes with leading/trailing whitespace should be trimmed.
	if err := s.SetNote(item.ID, "  hello world  "); err != nil {
		t.Fatalf("SetNote: %v", err)
	}
	got, _ := s.Get(item.ID)
	if got.Note != "hello world" {
		t.Errorf("expected trimmed note %q, got %q", "hello world", got.Note)
	}
}

func TestNoteUndoRestore(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "todos.json")

	s := NewStore(file)
	item, _ := s.Add("Undo me")
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Snapshot, then set note.
	if err := s.Snapshot(); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	_ = s.SetNote(item.ID, "Added note")
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Undo should restore the empty note.
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	got, _ := s.Get(item.ID)
	if got.Note != "" {
		t.Fatalf("expected note to be empty after undo, got %q", got.Note)
	}
}
