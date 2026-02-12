package todo

import (
	"bytes"
	"encoding/csv"
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

	items, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
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

	pending, err := s.List(StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	doneItems, err := s.List(StatusDone)
	if err != nil {
		t.Fatal(err)
	}
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

func TestSearchFindsMatches(t *testing.T) {
	s := tempStore(t)

	s.Add("Buy groceries")
	s.Add("Buy a new book")
	s.Add("Write tests")

	results := s.Search("buy")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, item := range results {
		if item.Title != "Buy groceries" && item.Title != "Buy a new book" {
			t.Errorf("unexpected item in results: %q", item.Title)
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	s := tempStore(t)

	s.Add("Deploy To Production")

	results := s.Search("deploy to production")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Deploy To Production" {
		t.Errorf("expected 'Deploy To Production', got %q", results[0].Title)
	}
}

func TestSearchNoMatches(t *testing.T) {
	s := tempStore(t)

	s.Add("Write docs")
	s.Add("Fix bug")

	results := s.Search("deploy")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyStore(t *testing.T) {
	s := tempStore(t)

	results := s.Search("anything")
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty store, got %d", len(results))
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

func TestValidStatus(t *testing.T) {
	valid := []Status{StatusPending, StatusInProgress, StatusDone}
	for _, s := range valid {
		if !ValidStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []Status{"foo", "bar", "completed", ""}
	for _, s := range invalid {
		if ValidStatus(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestListInvalidFilter(t *testing.T) {
	s := tempStore(t)
	s.Add("Task A")

	_, err := s.List(Status("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status filter, got nil")
	}
}

func TestListValidFilter(t *testing.T) {
	s := tempStore(t)
	s.Add("Task A")
	done := s.Add("Task B")
	s.SetStatus(done.ID, StatusDone)

	// Empty filter returns all
	all, err := s.List("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 items, got %d", len(all))
	}

	// Valid filter returns matching
	pending, err := s.List(StatusPending)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
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

func TestIDsStableAfterDelete(t *testing.T) {
	s := tempStore(t)

	a := s.Add("Task A") // ID 1
	b := s.Add("Task B") // ID 2
	c := s.Add("Task C") // ID 3

	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}

	// New item must NOT reuse the deleted ID 3; it should get ID 4.
	d := s.Add("Task D")
	if d.ID == c.ID {
		t.Errorf("new item reused deleted ID %d", c.ID)
	}
	if d.ID <= c.ID {
		t.Errorf("expected new ID > %d, got %d", c.ID, d.ID)
	}

	// Sanity: earlier IDs unchanged
	if a.ID != 1 || b.ID != 2 {
		t.Errorf("earlier IDs changed: a=%d b=%d", a.ID, b.ID)
	}
}

func TestIDsStableAfterDeleteAndReload(t *testing.T) {
	f, err := os.CreateTemp("", "todo-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s1 := NewStore(f.Name())
	s1.Load()
	s1.Add("Task A") // ID 1
	s1.Add("Task B") // ID 2
	c := s1.Add("Task C") // ID 3
	s1.Delete(c.ID)
	s1.Save()

	// Reload and add — must not reuse ID 3
	s2 := NewStore(f.Name())
	s2.Load()
	d := s2.Add("Task D")
	if d.ID <= c.ID {
		t.Errorf("after reload, expected new ID > %d, got %d", c.ID, d.ID)
	}
}

func TestExportEmpty(t *testing.T) {
	s := tempStore(t)

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// Should have only the header row
	if len(records) != 1 {
		t.Fatalf("expected 1 row (header), got %d", len(records))
	}
	if records[0][0] != "id" {
		t.Errorf("expected first header column 'id', got %q", records[0][0])
	}
}

func TestExportWithItems(t *testing.T) {
	s := tempStore(t)

	s.Add("Buy groceries")
	done := s.Add("Write tests")
	s.SetStatus(done.ID, StatusDone)

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}

	// Check header
	expectedHeader := []string{"id", "title", "status", "priority", "due_date", "created_at", "updated_at"}
	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("header[%d]: expected %q, got %q", i, col, records[0][i])
		}
	}

	// Check first data row
	if records[1][1] != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %q", records[1][1])
	}
	if records[1][2] != "pending" {
		t.Errorf("expected status 'pending', got %q", records[1][2])
	}

	// Check second data row
	if records[2][1] != "Write tests" {
		t.Errorf("expected title 'Write tests', got %q", records[2][1])
	}
	if records[2][2] != "done" {
		t.Errorf("expected status 'done', got %q", records[2][2])
	}
}

func TestExportCSVEscaping(t *testing.T) {
	s := tempStore(t)

	s.Add("Task with, comma")
	s.Add("Task with \"quotes\"")

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}
	if records[1][1] != "Task with, comma" {
		t.Errorf("expected 'Task with, comma', got %q", records[1][1])
	}
	if records[2][1] != "Task with \"quotes\"" {
		t.Errorf("expected 'Task with \"quotes\"', got %q", records[2][1])
	}
}

func TestAddWithPriority(t *testing.T) {
	s := tempStore(t)

	item := s.AddWithPriority("Urgent task", PriorityHigh)
	if item.Priority != PriorityHigh {
		t.Errorf("expected priority high, got %q", item.Priority)
	}
	if item.Title != "Urgent task" {
		t.Errorf("expected title 'Urgent task', got %q", item.Title)
	}
	if item.Status != StatusPending {
		t.Errorf("expected status pending, got %s", item.Status)
	}
}

func TestAddDefaultPriority(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Normal task")
	if item.Priority != PriorityNone {
		t.Errorf("expected empty priority, got %q", item.Priority)
	}
}

func TestSetPriority(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Task")
	if err := s.SetPriority(item.ID, PriorityMedium); err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(item.ID)
	if got.Priority != PriorityMedium {
		t.Errorf("expected priority medium, got %q", got.Priority)
	}
}

func TestSetPriorityClear(t *testing.T) {
	s := tempStore(t)

	item := s.AddWithPriority("Task", PriorityHigh)
	if err := s.SetPriority(item.ID, PriorityNone); err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(item.ID)
	if got.Priority != PriorityNone {
		t.Errorf("expected empty priority, got %q", got.Priority)
	}
}

func TestSetPriorityNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.SetPriority(999, PriorityHigh); err == nil {
		t.Error("expected error setting priority on non-existent item")
	}
}

func TestValidPriority(t *testing.T) {
	valid := []Priority{PriorityNone, PriorityLow, PriorityMedium, PriorityHigh}
	for _, p := range valid {
		if !ValidPriority(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}

	invalid := []Priority{"critical", "urgent", "1"}
	for _, p := range invalid {
		if ValidPriority(p) {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestExportWithPriority(t *testing.T) {
	s := tempStore(t)

	s.AddWithPriority("High task", PriorityHigh)
	s.Add("Normal task")

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}
	// priority is column index 3
	if records[1][3] != "high" {
		t.Errorf("expected priority 'high', got %q", records[1][3])
	}
	if records[2][3] != "" {
		t.Errorf("expected empty priority, got %q", records[2][3])
	}
}

func TestExportIncludesDueDate(t *testing.T) {
	s := tempStore(t)

	due, err := ParseDueDate("2025-06-15")
	if err != nil {
		t.Fatal(err)
	}
	s.AddFull("Task with due date", PriorityHigh, due)
	s.Add("Task without due date")

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}

	// Check header includes due_date column
	expectedHeader := []string{"id", "title", "status", "priority", "due_date", "created_at", "updated_at"}
	if len(records[0]) != len(expectedHeader) {
		t.Fatalf("expected %d columns, got %d: %v", len(expectedHeader), len(records[0]), records[0])
	}
	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("header[%d]: expected %q, got %q", i, col, records[0][i])
		}
	}

	// Check first data row has due date
	if records[1][4] != "2025-06-15" {
		t.Errorf("expected due_date '2025-06-15', got %q", records[1][4])
	}

	// Check second data row has empty due date
	if records[2][4] != "" {
		t.Errorf("expected empty due_date, got %q", records[2][4])
	}
}

func TestMultipleFiles(t *testing.T) {
	// Create two separate todo files to verify --file flag use case.
	f1, err := os.CreateTemp("", "todo-work-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f1.Close()
	defer os.Remove(f1.Name())

	f2, err := os.CreateTemp("", "todo-personal-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f2.Close()
	defer os.Remove(f2.Name())

	// Add items to first file
	s1 := NewStore(f1.Name())
	if err := s1.Load(); err != nil {
		t.Fatal(err)
	}
	s1.Add("Work task A")
	s1.Add("Work task B")
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Add items to second file
	s2 := NewStore(f2.Name())
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	s2.Add("Personal task X")
	if err := s2.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload and verify they are independent
	r1 := NewStore(f1.Name())
	if err := r1.Load(); err != nil {
		t.Fatal(err)
	}
	if len(r1.Items) != 2 {
		t.Errorf("expected 2 items in work file, got %d", len(r1.Items))
	}
	if r1.Items[0].Title != "Work task A" {
		t.Errorf("expected 'Work task A', got %q", r1.Items[0].Title)
	}

	r2 := NewStore(f2.Name())
	if err := r2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(r2.Items) != 1 {
		t.Errorf("expected 1 item in personal file, got %d", len(r2.Items))
	}
	if r2.Items[0].Title != "Personal task X" {
		t.Errorf("expected 'Personal task X', got %q", r2.Items[0].Title)
	}
}

func TestSetStatusRejectsInvalid(t *testing.T) {
	s := tempStore(t)

	item := s.Add("Task")

	// Attempting to set an invalid status should return an error.
	err := s.SetStatus(item.ID, Status("garbage"))
	if err == nil {
		t.Fatal("expected error when setting invalid status, got nil")
	}

	// The item's status should remain unchanged.
	got, _ := s.Get(item.ID)
	if got.Status != StatusPending {
		t.Errorf("expected status to remain pending after invalid SetStatus, got %s", got.Status)
	}
}

func TestSetPriorityRejectsInvalid(t *testing.T) {
	s := tempStore(t)

	item := s.AddWithPriority("Task", PriorityHigh)

	// Attempting to set an invalid priority should return an error.
	err := s.SetPriority(item.ID, Priority("critical"))
	if err == nil {
		t.Fatal("expected error when setting invalid priority, got nil")
	}

	// The item's priority should remain unchanged.
	got, _ := s.Get(item.ID)
	if got.Priority != PriorityHigh {
		t.Errorf("expected priority to remain high after invalid SetPriority, got %q", got.Priority)
	}
}

func TestPriorityPersistence(t *testing.T) {
	f, err := os.CreateTemp("", "todo-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s1 := NewStore(f.Name())
	s1.Load()
	s1.AddWithPriority("Important", PriorityHigh)
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	s2 := NewStore(f.Name())
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(s2.Items))
	}
	if s2.Items[0].Priority != PriorityHigh {
		t.Errorf("expected priority high after reload, got %q", s2.Items[0].Priority)
	}
}
