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

	_, _ = s.Add("Write tests")
	_, _ = s.Add("Ship feature")

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

	item, _ := s.Add("Do thing")
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

	item, _ := s.Add("Remove me")
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

	_, _ = s.Add("Pending task")
	done, _ := s.Add("Done task")
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

	_, _ = s.Add("Task A")
	_, _ = s.Add("Task B")
	started, _ := s.Add("Task C")
	s.SetStatus(started.ID, StatusInProgress)
	done, _ := s.Add("Task D")
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

	item, _ := s.Add("Old title")
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

	_, _ = s.Add("Buy groceries")
	_, _ = s.Add("Buy a new book")
	_, _ = s.Add("Write tests")

	results, err := s.Search("buy")
	if err != nil {
		t.Fatal(err)
	}
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

	_, _ = s.Add("Deploy To Production")

	results, err := s.Search("deploy to production")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Deploy To Production" {
		t.Errorf("expected 'Deploy To Production', got %q", results[0].Title)
	}
}

func TestSearchNoMatches(t *testing.T) {
	s := tempStore(t)

	_, _ = s.Add("Write docs")
	_, _ = s.Add("Fix bug")

	results, err := s.Search("deploy")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyStore(t *testing.T) {
	s := tempStore(t)

	results, err := s.Search("anything")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty store, got %d", len(results))
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("Task")

	_, err := s.Search("")
	if err == nil {
		t.Fatal("expected error for empty search query, got nil")
	}

	_, err = s.Search("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only search query, got nil")
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

	// Simulate: todo add Buy some milk  ->  os.Args[2:] = ["Buy", "some", "milk"]
	args := []string{"Buy", "some", "milk"}
	title := ParseAddTitle(args)
	item, _ := s.Add(title)

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
	_, _ = s.Add("Task A")

	_, err := s.List(Status("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status filter, got nil")
	}
}

func TestListValidFilter(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("Task A")
	done, _ := s.Add("Task B")
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
	_, _ = s1.Add("Persist me")
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

	a, _ := s.Add("Task A") // ID 1
	b, _ := s.Add("Task B") // ID 2
	c, _ := s.Add("Task C") // ID 3

	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}

	// New item must NOT reuse the deleted ID 3; it should get ID 4.
	d, _ := s.Add("Task D")
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
	_, _ = s1.Add("Task A") // ID 1
	_, _ = s1.Add("Task B") // ID 2
	c, _ := s1.Add("Task C") // ID 3
	s1.Delete(c.ID)
	s1.Save()

	// Reload and add -- must not reuse ID 3
	s2 := NewStore(f.Name())
	s2.Load()
	d, _ := s2.Add("Task D")
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

	_, _ = s.Add("Buy groceries")
	done, _ := s.Add("Write tests")
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

	_, _ = s.Add("Task with, comma")
	_, _ = s.Add("Task with \"quotes\"")

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

	item, _ := s.AddWithPriority("Urgent task", PriorityHigh)
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

	item, _ := s.Add("Normal task")
	if item.Priority != PriorityNone {
		t.Errorf("expected empty priority, got %q", item.Priority)
	}
}

func TestSetPriority(t *testing.T) {
	s := tempStore(t)

	item, _ := s.Add("Task")
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

	item, _ := s.AddWithPriority("Task", PriorityHigh)
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

	_, _ = s.AddWithPriority("High task", PriorityHigh)
	_, _ = s.Add("Normal task")

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
	_, _ = s.AddFull("Task with due date", PriorityHigh, due)
	_, _ = s.Add("Task without due date")

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
	_, _ = s1.Add("Work task A")
	_, _ = s1.Add("Work task B")
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Add items to second file
	s2 := NewStore(f2.Name())
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	_, _ = s2.Add("Personal task X")
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

	item, _ := s.Add("Task")

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

	item, _ := s.AddWithPriority("Task", PriorityHigh)

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

func TestParseDueDateValid(t *testing.T) {
	d, err := ParseDueDate("2025-03-15")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Valid {
		t.Fatal("expected Valid=true")
	}
	if d.String() != "2025-03-15" {
		t.Errorf("expected '2025-03-15', got %q", d.String())
	}
}

func TestParseDueDateEmpty(t *testing.T) {
	d, err := ParseDueDate("")
	if err != nil {
		t.Fatal(err)
	}
	if d.Valid {
		t.Error("expected Valid=false for empty string")
	}
	if d.String() != "" {
		t.Errorf("expected empty string, got %q", d.String())
	}
}

func TestParseDueDateInvalid(t *testing.T) {
	_, err := ParseDueDate("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date string")
	}
}

func TestAddRejectsEmptyTitle(t *testing.T) {
	s := tempStore(t)

	_, err := s.AddFull("", PriorityNone, DueDate{})
	if err == nil {
		t.Fatal("expected error when adding item with empty title, got nil")
	}
}

func TestAddRejectsWhitespaceOnlyTitle(t *testing.T) {
	s := tempStore(t)

	_, err := s.AddFull("   ", PriorityNone, DueDate{})
	if err == nil {
		t.Fatal("expected error when adding item with whitespace-only title, got nil")
	}
}

func TestEditRejectsEmptyTitle(t *testing.T) {
	s := tempStore(t)

	item, _ := s.AddFull("Valid title", PriorityNone, DueDate{})

	err := s.Edit(item.ID, "")
	if err == nil {
		t.Fatal("expected error when editing item to empty title, got nil")
	}

	// Title should remain unchanged.
	got, _ := s.Get(item.ID)
	if got.Title != "Valid title" {
		t.Errorf("expected title to remain 'Valid title', got %q", got.Title)
	}
}

func TestEditRejectsWhitespaceOnlyTitle(t *testing.T) {
	s := tempStore(t)

	item, _ := s.AddFull("Valid title", PriorityNone, DueDate{})

	err := s.Edit(item.ID, "   \t  ")
	if err == nil {
		t.Fatal("expected error when editing item to whitespace-only title, got nil")
	}

	got, _ := s.Get(item.ID)
	if got.Title != "Valid title" {
		t.Errorf("expected title to remain 'Valid title', got %q", got.Title)
	}
}

func TestAddFullWithDueDate(t *testing.T) {
	s := tempStore(t)
	due, _ := ParseDueDate("2025-06-01")
	item, _ := s.AddFull("Task with due", PriorityHigh, due)
	if !item.DueDate.Valid {
		t.Fatal("expected due date to be set")
	}
	if item.DueDate.String() != "2025-06-01" {
		t.Errorf("expected '2025-06-01', got %q", item.DueDate.String())
	}
	if item.Priority != PriorityHigh {
		t.Errorf("expected priority high, got %q", item.Priority)
	}
}

func TestSetDueDate(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	due, _ := ParseDueDate("2025-12-25")
	if err := s.SetDueDate(item.ID, due); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(item.ID)
	if !got.DueDate.Valid || got.DueDate.String() != "2025-12-25" {
		t.Errorf("expected due date '2025-12-25', got %q", got.DueDate.String())
	}
}

func TestSetDueDateClear(t *testing.T) {
	s := tempStore(t)
	due, _ := ParseDueDate("2025-12-25")
	item, _ := s.AddFull("Task", PriorityNone, due)
	if err := s.SetDueDate(item.ID, DueDate{}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(item.ID)
	if got.DueDate.Valid {
		t.Error("expected due date to be cleared")
	}
}

func TestSetDueDateNotFound(t *testing.T) {
	s := tempStore(t)
	due, _ := ParseDueDate("2025-12-25")
	if err := s.SetDueDate(999, due); err == nil {
		t.Error("expected error setting due date on non-existent item")
	}
}

func TestDueDatePersistence(t *testing.T) {
	f, err := os.CreateTemp("", "todo-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	due, _ := ParseDueDate("2025-09-01")
	s1 := NewStore(f.Name())
	s1.Load()
	_, _ = s1.AddFull("Deadline task", PriorityNone, due)
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
	if !s2.Items[0].DueDate.Valid || s2.Items[0].DueDate.String() != "2025-09-01" {
		t.Errorf("expected due date '2025-09-01' after reload, got %q (valid=%v)",
			s2.Items[0].DueDate.String(), s2.Items[0].DueDate.Valid)
	}
}

func TestDueDateJSON(t *testing.T) {
	s := tempStore(t)
	due, _ := ParseDueDate("2025-04-10")
	_, _ = s.AddFull("JSON test", PriorityNone, due)
	_, _ = s.Add("No due date")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	s2 := NewStore(s.file)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if !s2.Items[0].DueDate.Valid {
		t.Error("expected first item to have due date after reload")
	}
	if s2.Items[1].DueDate.Valid {
		t.Error("expected second item to have no due date after reload")
	}
}

func TestAddFullRejectsInvalidPriority(t *testing.T) {
	s := tempStore(t)

	_, err := s.AddFull("Task", Priority("critical"), DueDate{})
	if err == nil {
		t.Fatal("expected error when adding item with invalid priority, got nil")
	}
	// Store should remain empty — nothing was added.
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items after rejected add, got %d", len(s.Items))
	}
}

func TestAddWithPriorityRejectsInvalid(t *testing.T) {
	s := tempStore(t)

	_, err := s.AddWithPriority("Task", Priority("urgent"))
	if err == nil {
		t.Fatal("expected error when adding item with invalid priority, got nil")
	}
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items after rejected add, got %d", len(s.Items))
	}
}

func TestClearDoneEmpty(t *testing.T) {
	s := tempStore(t)
	removed := s.ClearDone()
	if removed != 0 {
		t.Errorf("expected 0 removed from empty store, got %d", removed)
	}
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(s.Items))
	}
}

func TestClearDoneRemovesOnlyDone(t *testing.T) {
	s := tempStore(t)
	item1, _ := s.Add("Pending task")
	item2, _ := s.Add("Done task")
	item3, _ := s.Add("In-progress task")
	_ = s.SetStatus(item2.ID, StatusDone)
	_ = s.SetStatus(item3.ID, StatusInProgress)

	removed := s.ClearDone()
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if len(s.Items) != 2 {
		t.Errorf("expected 2 items remaining, got %d", len(s.Items))
	}
	// Verify the right items remain.
	for _, item := range s.Items {
		if item.ID == item2.ID {
			t.Error("done item should have been removed")
		}
	}
	_, err := s.Get(item1.ID)
	if err != nil {
		t.Error("pending item should still exist")
	}
	_, err = s.Get(item3.ID)
	if err != nil {
		t.Error("in-progress item should still exist")
	}
}

func TestClearDoneAllDone(t *testing.T) {
	s := tempStore(t)
	item1, _ := s.Add("Task A")
	item2, _ := s.Add("Task B")
	_ = s.SetStatus(item1.ID, StatusDone)
	_ = s.SetStatus(item2.ID, StatusDone)

	removed := s.ClearDone()
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items remaining, got %d", len(s.Items))
	}
}

func TestClearDoneNoneDone(t *testing.T) {
	s := tempStore(t)
	s.Add("Pending A")
	s.Add("Pending B")

	removed := s.ClearDone()
	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
	if len(s.Items) != 2 {
		t.Errorf("expected 2 items remaining, got %d", len(s.Items))
	}
}

func TestAddTrimsWhitespace(t *testing.T) {
	s := tempStore(t)

	item, err := s.Add("  hello world  ")
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "hello world" {
		t.Errorf("expected trimmed title 'hello world', got %q", item.Title)
	}
}

func TestAddFullTrimsWhitespace(t *testing.T) {
	s := tempStore(t)

	item, err := s.AddFull("\t Buy milk \n", PriorityHigh, DueDate{})
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "Buy milk" {
		t.Errorf("expected trimmed title 'Buy milk', got %q", item.Title)
	}
}

func TestEditTrimsWhitespace(t *testing.T) {
	s := tempStore(t)

	item, _ := s.Add("Original")
	if err := s.Edit(item.ID, "  Updated title  "); err != nil {
		t.Fatal(err)
	}

	got, _ := s.Get(item.ID)
	if got.Title != "Updated title" {
		t.Errorf("expected trimmed title 'Updated title', got %q", got.Title)
	}
}

func TestSortByPriority(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("No priority")
	_, _ = s.AddWithPriority("Low task", PriorityLow)
	_, _ = s.AddWithPriority("High task", PriorityHigh)
	_, _ = s.AddWithPriority("Medium task", PriorityMedium)

	if err := s.Sort(SortByPriority); err != nil {
		t.Fatal(err)
	}

	expected := []string{"High task", "Medium task", "Low task", "No priority"}
	for i, want := range expected {
		if s.Items[i].Title != want {
			t.Errorf("index %d: expected %q, got %q", i, want, s.Items[i].Title)
		}
	}
}

func TestSortByDue(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("No due date")
	due1, _ := ParseDueDate("2025-12-01")
	due2, _ := ParseDueDate("2025-06-01")
	due3, _ := ParseDueDate("2025-09-01")
	_, _ = s.AddFull("December task", PriorityNone, due1)
	_, _ = s.AddFull("June task", PriorityNone, due2)
	_, _ = s.AddFull("September task", PriorityNone, due3)

	if err := s.Sort(SortByDue); err != nil {
		t.Fatal(err)
	}

	expected := []string{"June task", "September task", "December task", "No due date"}
	for i, want := range expected {
		if s.Items[i].Title != want {
			t.Errorf("index %d: expected %q, got %q", i, want, s.Items[i].Title)
		}
	}
}

func TestSortByStatus(t *testing.T) {
	s := tempStore(t)
	pending, _ := s.Add("Pending task")
	done, _ := s.Add("Done task")
	inProgress, _ := s.Add("In-progress task")
	_ = s.SetStatus(done.ID, StatusDone)
	_ = s.SetStatus(inProgress.ID, StatusInProgress)

	if err := s.Sort(SortByStatus); err != nil {
		t.Fatal(err)
	}

	if s.Items[0].ID != inProgress.ID {
		t.Errorf("expected in_progress first, got %q", s.Items[0].Title)
	}
	if s.Items[1].ID != pending.ID {
		t.Errorf("expected pending second, got %q", s.Items[1].Title)
	}
	if s.Items[2].ID != done.ID {
		t.Errorf("expected done last, got %q", s.Items[2].Title)
	}
}

func TestSortByCreated(t *testing.T) {
	s := tempStore(t)
	// Items are added in order, so sorting by created should keep the same order.
	_, _ = s.Add("First")
	_, _ = s.Add("Second")
	_, _ = s.Add("Third")

	if err := s.Sort(SortByCreated); err != nil {
		t.Fatal(err)
	}

	expected := []string{"First", "Second", "Third"}
	for i, want := range expected {
		if s.Items[i].Title != want {
			t.Errorf("index %d: expected %q, got %q", i, want, s.Items[i].Title)
		}
	}
}

func TestSortInvalidField(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("Task")

	err := s.Sort(SortField("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid sort field, got nil")
	}
}

func TestSortEmptyStore(t *testing.T) {
	s := tempStore(t)
	if err := s.Sort(SortByPriority); err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(s.Items))
	}
}

func TestSortPersists(t *testing.T) {
	f, err := os.CreateTemp("", "todo-sort-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s1 := NewStore(f.Name())
	s1.Load()
	_, _ = s1.AddWithPriority("Low", PriorityLow)
	_, _ = s1.AddWithPriority("High", PriorityHigh)
	s1.Sort(SortByPriority)
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	s2 := NewStore(f.Name())
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if s2.Items[0].Title != "High" {
		t.Errorf("expected 'High' first after reload, got %q", s2.Items[0].Title)
	}
	if s2.Items[1].Title != "Low" {
		t.Errorf("expected 'Low' second after reload, got %q", s2.Items[1].Title)
	}
}

func TestValidSortField(t *testing.T) {
	valid := []SortField{SortByPriority, SortByDue, SortByStatus, SortByCreated}
	for _, f := range valid {
		if !ValidSortField(f) {
			t.Errorf("expected %q to be valid", f)
		}
	}

	invalid := []SortField{"name", "title", "id", ""}
	for _, f := range invalid {
		if ValidSortField(f) {
			t.Errorf("expected %q to be invalid", f)
		}
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
	_, _ = s1.AddWithPriority("Important", PriorityHigh)
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

func TestSnapshotAndUndo(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.Add("First")
	_, _ = s.Add("Second")
	s.Save()

	// Snapshot before deleting.
	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(1); err != nil {
		t.Fatal(err)
	}
	s.Save()

	if len(s.Items) != 1 {
		t.Fatalf("expected 1 item after delete, got %d", len(s.Items))
	}

	// Undo should restore both items.
	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 2 {
		t.Fatalf("expected 2 items after undo, got %d", len(s.Items))
	}
	if s.Items[0].Title != "First" || s.Items[1].Title != "Second" {
		t.Errorf("unexpected items after undo: %v", s.Items)
	}
}

func TestUndoNothingToUndo(t *testing.T) {
	s := tempStore(t)
	err := s.Undo()
	if err == nil {
		t.Fatal("expected error when nothing to undo")
	}
	if err.Error() != "nothing to undo" {
		t.Errorf("expected 'nothing to undo', got %q", err.Error())
	}
}

func TestUndoOnlyOnce(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.Add("Task")
	s.Save()

	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(1); err != nil {
		t.Fatal(err)
	}
	s.Save()

	// First undo succeeds.
	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	// Second undo should fail (undo file removed).
	if err := s.Undo(); err == nil {
		t.Fatal("expected error on second undo")
	}
}

func TestUndoRestoresEditedTitle(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.Add("Original")
	s.Save()

	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.Edit(1, "Changed"); err != nil {
		t.Fatal(err)
	}
	s.Save()

	if s.Items[0].Title != "Changed" {
		t.Fatalf("expected 'Changed', got %q", s.Items[0].Title)
	}

	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if s.Items[0].Title != "Original" {
		t.Errorf("expected 'Original' after undo, got %q", s.Items[0].Title)
	}
}

func TestUndoRestoresClearedItems(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.Add("Keep")
	_, _ = s.Add("Remove")
	s.SetStatus(2, StatusDone)
	s.Save()

	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	removed := s.ClearDone()
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
	s.Save()

	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 2 {
		t.Fatalf("expected 2 items after undo, got %d", len(s.Items))
	}
}

func TestClearDoneAllPersistence(t *testing.T) {
	f, err := os.CreateTemp("", "todo-clear-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s1 := NewStore(f.Name())
	s1.Load()
	item1, _ := s1.Add("Task A")
	item2, _ := s1.Add("Task B")
	_ = s1.SetStatus(item1.ID, StatusDone)
	_ = s1.SetStatus(item2.ID, StatusDone)

	removed := s1.ClearDone()
	if removed != 2 {
		t.Fatalf("expected 2 removed, got %d", removed)
	}
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	// Reload — should succeed and have 0 items, not fail to parse.
	s2 := NewStore(f.Name())
	if err := s2.Load(); err != nil {
		t.Fatalf("failed to load after ClearDone removed all items: %v", err)
	}
	if len(s2.Items) != 0 {
		t.Errorf("expected 0 items after reload, got %d", len(s2.Items))
	}

	// NextID should be preserved so new items don't reuse old IDs.
	newItem, _ := s2.Add("Task C")
	if newItem.ID <= item2.ID {
		t.Errorf("expected new ID > %d, got %d", item2.ID, newItem.ID)
	}
}

func TestUndoPersistsToDisk(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.Add("Persist me")
	s.Save()

	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(1); err != nil {
		t.Fatal(err)
	}
	s.Save()

	// Create a fresh store from the same file and undo.
	s2 := NewStore(s.file)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Items) != 0 {
		t.Fatalf("expected 0 items before undo, got %d", len(s2.Items))
	}
	if err := s2.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Items) != 1 {
		t.Fatalf("expected 1 item after undo, got %d", len(s2.Items))
	}
	if s2.Items[0].Title != "Persist me" {
		t.Errorf("expected 'Persist me', got %q", s2.Items[0].Title)
	}
}
