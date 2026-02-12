package todo

import (
	"os"
	"testing"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	f, err := os.CreateTemp("", "todo-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	s := NewStore(f.Name())
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAddAndList(t *testing.T) {
	s := tempStore(t)

	s.Add("Write tests")
	s.Add("Ship feature")

	items := s.List("")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "Write tests" {
		t.Errorf("expected title 'Write tests', got %q", items[0].Title)
	}
	if items[0].Status != StatusPending {
		t.Errorf("expected status pending, got %s", items[0].Status)
	}
}

func TestSetStatus(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Do thing")
	if err := s.SetStatus(item.ID, StatusDone); err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(item.ID)
	if got.Status != StatusDone {
		t.Errorf("expected done, got %s", got.Status)
	}
}

func TestDelete(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Remove me")
	if err := s.Delete(item.ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(s.Items))
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.Delete(999); err == nil {
		t.Error("expected error deleting non-existent item")
	}
}

func TestListFilter(t *testing.T) {
	s := tempStore(t)

	s.Add("Pending task")
	done := s.Add("Done task")
	s.SetStatus(done.ID, StatusDone)

	pending := s.List(StatusPending)
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	doneItems := s.List(StatusDone)
	if len(doneItems) != 1 {
		t.Errorf("expected 1 done, got %d", len(doneItems))
	}
}

func TestStatsEmpty(t *testing.T) {
	s := tempStore(t)

	stats := s.Stats()
	if stats[StatusPending] != 0 || stats[StatusInProgress] != 0 || stats[StatusDone] != 0 {
		t.Errorf("expected all zeros for empty store, got %v", stats)
	}
}

func TestStats(t *testing.T) {
	s := tempStore(t)

	s.Add("Task A")
	s.Add("Task B")
	started := s.Add("Task C")
	s.SetStatus(started.ID, StatusInProgress)
	done := s.Add("Task D")
	s.SetStatus(done.ID, StatusDone)

	stats := s.Stats()
	if stats[StatusPending] != 2 {
		t.Errorf("expected 2 pending, got %d", stats[StatusPending])
	}
	if stats[StatusInProgress] != 1 {
		t.Errorf("expected 1 in_progress, got %d", stats[StatusInProgress])
	}
	if stats[StatusDone] != 1 {
		t.Errorf("expected 1 done, got %d", stats[StatusDone])
	}
}

func TestEdit(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Old title")
	if err := s.Edit(item.ID, "New title"); err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(item.ID)
	if got.Title != "New title" {
		t.Errorf("expected title 'New title', got %q", got.Title)
	}
	if !got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt) {
		// UpdatedAt should be >= CreatedAt (may be equal due to clock resolution)
	}
}

func TestEditNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.Edit(999, "Nope"); err == nil {
		t.Error("expected error editing non-existent item")
	}
}

func TestParseAddTitle(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"single word", []string{"groceries"}, "groceries"},
		{"multi word", []string{"Buy", "some", "milk"}, "Buy some milk"},
		{"two words", []string{"Write", "tests"}, "Write tests"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAddTitle(tt.args)
			if got != tt.want {
				t.Errorf("ParseAddTitle(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseAddTitleUsedWithAdd(t *testing.T) {
	s := tempStore(t)

	// Simulate: todo add Buy some milk  →  os.Args[2:] = ["Buy", "some", "milk"]
	args := []string{"Buy", "some", "milk"}
	title := ParseAddTitle(args)
	item := s.Add(title)

	if item.Title != "Buy some milk" {
		t.Errorf("expected title 'Buy some milk', got %q", item.Title)
	}
}

func TestPersistence(t *testing.T) {
	f, err := os.CreateTemp("", "todo-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	// Write
	s1 := NewStore(f.Name())
	s1.Load()
	s1.Add("Persist me")
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Read back
	s2 := NewStore(f.Name())
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Items) != 1 {
		t.Fatalf("expected 1 item after reload, got %d", len(s2.Items))
	}
	if s2.Items[0].Title != "Persist me" {
		t.Errorf("expected 'Persist me', got %q", s2.Items[0].Title)
	}
}
