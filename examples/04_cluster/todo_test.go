package todo

import (
	"bytes"
	"encoding/csv"
	"os"
	"strings"
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

	// Check header (includes tags column)
	expectedHeader := []string{"id", "title", "status", "priority", "due_date", "tags", "created_at", "updated_at"}
	if len(records[0]) != len(expectedHeader) {
		t.Fatalf("expected %d columns, got %d: %v", len(expectedHeader), len(records[0]), records[0])
	}
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

func TestExportIncludesTags(t *testing.T) {
	s := tempStore(t)

	item, _ := s.Add("Tagged task")
	_ = s.AddTag(item.ID, "work")
	_ = s.AddTag(item.ID, "urgent")
	_, _ = s.Add("Untagged task")

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

	// tags is column index 5
	if records[0][5] != "tags" {
		t.Errorf("expected header column 5 to be 'tags', got %q", records[0][5])
	}
	if records[1][5] != "work;urgent" {
		t.Errorf("expected tags 'work;urgent', got %q", records[1][5])
	}
	if records[2][5] != "" {
		t.Errorf("expected empty tags, got %q", records[2][5])
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

	// Check header includes due_date and tags columns
	expectedHeader := []string{"id", "title", "status", "priority", "due_date", "tags", "created_at", "updated_at"}
	if len(records[0]) != len(expectedHeader) {
		t.Fatalf("expected %d columns, got %d: %v", len(expectedHeader), len(records[0]), records[0])
	}
	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("header[%d]: expected %q, got %q", i, col, records[0][i])
		}
	}

	// Check first data row has due date (column index 4)
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

func TestListUnfilteredReturnsCopy(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("Task A")
	_, _ = s.Add("Task B")

	items, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Mutating the returned slice should NOT affect the store's internal state.
	items[0].Title = "MUTATED"

	got, _ := s.Get(1)
	if got.Title == "MUTATED" {
		t.Error("List(\"\") returned a reference to internal Items slice; callers can corrupt store state")
	}
	if got.Title != "Task A" {
		t.Errorf("expected title 'Task A', got %q", got.Title)
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

// --- Tag tests ---

func TestAddFullWithTags(t *testing.T) {
	s := tempStore(t)
	item, err := s.AddFullWithTags("Tagged task", PriorityNone, DueDate{}, []string{"work", "urgent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(item.Tags))
	}
	if item.Tags[0] != "work" || item.Tags[1] != "urgent" {
		t.Errorf("expected [work, urgent], got %v", item.Tags)
	}
}

func TestAddFullWithTagsNormalisesToLowercase(t *testing.T) {
	s := tempStore(t)
	item, err := s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"Work", "URGENT"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Tags[0] != "work" || item.Tags[1] != "urgent" {
		t.Errorf("expected lowercase tags, got %v", item.Tags)
	}
}

func TestAddFullWithTagsDeduplicates(t *testing.T) {
	s := tempStore(t)
	item, err := s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"work", "Work", "WORK"})
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Tags) != 1 {
		t.Errorf("expected 1 tag after dedup, got %d: %v", len(item.Tags), item.Tags)
	}
}

func TestAddFullWithTagsTrimsWhitespace(t *testing.T) {
	s := tempStore(t)
	item, err := s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"  work  ", " ", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Tags) != 1 {
		t.Errorf("expected 1 tag after trimming empties, got %d: %v", len(item.Tags), item.Tags)
	}
	if item.Tags[0] != "work" {
		t.Errorf("expected 'work', got %q", item.Tags[0])
	}
}

func TestAddFullWithNilTags(t *testing.T) {
	s := tempStore(t)
	item, err := s.AddFullWithTags("Task", PriorityNone, DueDate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(item.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(item.Tags))
	}
}

func TestAddTag(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	if err := s.AddTag(item.ID, "work"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(item.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "work" {
		t.Errorf("expected [work], got %v", got.Tags)
	}
}

func TestAddTagDuplicateIgnored(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	s.AddTag(item.ID, "work")
	s.AddTag(item.ID, "Work")   // case-insensitive duplicate
	s.AddTag(item.ID, "  work") // whitespace duplicate
	got, _ := s.Get(item.ID)
	if len(got.Tags) != 1 {
		t.Errorf("expected 1 tag (no duplicates), got %d: %v", len(got.Tags), got.Tags)
	}
}

func TestAddTagRejectsEmpty(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	if err := s.AddTag(item.ID, ""); err == nil {
		t.Fatal("expected error for empty tag")
	}
	if err := s.AddTag(item.ID, "   "); err == nil {
		t.Fatal("expected error for whitespace-only tag")
	}
}

func TestAddTagNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.AddTag(999, "work"); err == nil {
		t.Fatal("expected error for non-existent item")
	}
}

func TestRemoveTag(t *testing.T) {
	s := tempStore(t)
	item, _ := s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"work", "urgent"})
	if err := s.RemoveTag(item.ID, "work"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(item.ID)
	if len(got.Tags) != 1 || got.Tags[0] != "urgent" {
		t.Errorf("expected [urgent], got %v", got.Tags)
	}
}

func TestRemoveTagCaseInsensitive(t *testing.T) {
	s := tempStore(t)
	item, _ := s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"work"})
	if err := s.RemoveTag(item.ID, "WORK"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(item.ID)
	if len(got.Tags) != 0 {
		t.Errorf("expected 0 tags after removal, got %v", got.Tags)
	}
}

func TestRemoveTagNotPresent(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	if err := s.RemoveTag(item.ID, "nonexistent"); err == nil {
		t.Fatal("expected error removing tag that doesn't exist")
	}
}

func TestRemoveTagRejectsEmpty(t *testing.T) {
	s := tempStore(t)
	item, _ := s.Add("Task")
	if err := s.RemoveTag(item.ID, ""); err == nil {
		t.Fatal("expected error for empty tag")
	}
}

func TestRemoveTagNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.RemoveTag(999, "work"); err == nil {
		t.Fatal("expected error for non-existent item")
	}
}

func TestHasTag(t *testing.T) {
	item := &Item{Tags: []string{"work", "urgent"}}
	if !item.HasTag("work") {
		t.Error("expected HasTag('work') to be true")
	}
	if !item.HasTag("WORK") {
		t.Error("expected HasTag('WORK') to be true (case-insensitive)")
	}
	if item.HasTag("personal") {
		t.Error("expected HasTag('personal') to be false")
	}
}

func TestHasTagEmptyTags(t *testing.T) {
	item := &Item{}
	if item.HasTag("work") {
		t.Error("expected HasTag to be false for item with no tags")
	}
}

func TestListByTag(t *testing.T) {
	s := tempStore(t)
	s.AddFullWithTags("Work task 1", PriorityNone, DueDate{}, []string{"work"})
	s.AddFullWithTags("Work task 2", PriorityNone, DueDate{}, []string{"work", "urgent"})
	s.Add("Personal task")

	items, err := s.ListByTag("work")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items with tag 'work', got %d", len(items))
	}
}

func TestListByTagCaseInsensitive(t *testing.T) {
	s := tempStore(t)
	s.AddFullWithTags("Task", PriorityNone, DueDate{}, []string{"work"})

	items, err := s.ListByTag("WORK")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestListByTagEmpty(t *testing.T) {
	s := tempStore(t)
	_, err := s.ListByTag("")
	if err == nil {
		t.Fatal("expected error for empty tag filter")
	}
}

func TestListByTagNoMatches(t *testing.T) {
	s := tempStore(t)
	s.Add("Task without tags")
	items, err := s.ListByTag("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestTagsPersistence(t *testing.T) {
	f, err := os.CreateTemp("", "todo-tags-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s1 := NewStore(f.Name())
	s1.Load()
	_, _ = s1.AddFullWithTags("Tagged task", PriorityNone, DueDate{}, []string{"work", "urgent"})
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
	if len(s2.Items[0].Tags) != 2 {
		t.Fatalf("expected 2 tags after reload, got %d", len(s2.Items[0].Tags))
	}
	if s2.Items[0].Tags[0] != "work" || s2.Items[0].Tags[1] != "urgent" {
		t.Errorf("expected [work, urgent] after reload, got %v", s2.Items[0].Tags)
	}
}

func TestTagsOmittedWhenEmpty(t *testing.T) {
	s := tempStore(t)
	_, _ = s.Add("No tags")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	s2 := NewStore(s.file)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if len(s2.Items[0].Tags) != 0 {
		t.Errorf("expected 0 tags, got %v", s2.Items[0].Tags)
	}
}

func TestExportIncludesTagsFromAddFullWithTags(t *testing.T) {
	s := tempStore(t)
	_, _ = s.AddFullWithTags("Tagged", PriorityNone, DueDate{}, []string{"work", "urgent"})
	_, _ = s.Add("No tags")

	var buf bytes.Buffer
	if err := s.Export(&buf); err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Check header includes tags
	expectedHeader := []string{"id", "title", "status", "priority", "due_date", "tags", "created_at", "updated_at"}
	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("header[%d]: expected %q, got %q", i, col, records[0][i])
		}
	}

	// Check tags column (index 5)
	if records[1][5] != "work;urgent" {
		t.Errorf("expected tags 'work;urgent', got %q", records[1][5])
	}
	if records[2][5] != "" {
		t.Errorf("expected empty tags for untagged item, got %q", records[2][5])
	}
}

func TestFormatTags(t *testing.T) {
	if got := FormatTags(nil); got != "-" {
		t.Errorf("expected '-' for nil tags, got %q", got)
	}
	if got := FormatTags([]string{}); got != "-" {
		t.Errorf("expected '-' for empty tags, got %q", got)
	}
	if got := FormatTags([]string{"work"}); got != "work" {
		t.Errorf("expected 'work', got %q", got)
	}
	if got := FormatTags([]string{"work", "urgent"}); got != "work, urgent" {
		t.Errorf("expected 'work, urgent', got %q", got)
	}
}

func TestValidTag(t *testing.T) {
	if ValidTag("") {
		t.Error("expected empty string to be invalid")
	}
	if ValidTag("   ") {
		t.Error("expected whitespace-only to be invalid")
	}
	if !ValidTag("work") {
		t.Error("expected 'work' to be valid")
	}
	if !ValidTag("  work  ") {
		t.Error("expected '  work  ' to be valid (trimmable)")
	}
}

// --- Import tests ---

func TestImportBasic(t *testing.T) {
	s := tempStore(t)

	csvData := `id,title,status,priority,due_date,tags,created_at,updated_at
1,Buy milk,pending,high,2025-03-01,grocery;home,2025-01-01T00:00:00Z,2025-01-01T00:00:00Z
2,Write report,in_progress,medium,,work,2025-01-02T00:00:00Z,2025-01-02T00:00:00Z
3,Relax,done,,,,,
`
	count, err := s.Import(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 imported, got %d", count)
	}
	if len(s.Items) != 3 {
		t.Fatalf("expected 3 items in store, got %d", len(s.Items))
	}

	// Items get new IDs starting from 1.
	if s.Items[0].ID != 1 || s.Items[0].Title != "Buy milk" {
		t.Errorf("item 0: got ID=%d Title=%q", s.Items[0].ID, s.Items[0].Title)
	}
	if s.Items[0].Status != StatusPending {
		t.Errorf("item 0: expected pending, got %s", s.Items[0].Status)
	}
	if s.Items[0].Priority != PriorityHigh {
		t.Errorf("item 0: expected high, got %s", s.Items[0].Priority)
	}
	if !s.Items[0].DueDate.Valid || s.Items[0].DueDate.String() != "2025-03-01" {
		t.Errorf("item 0: expected due 2025-03-01, got %s", s.Items[0].DueDate)
	}
	if len(s.Items[0].Tags) != 2 || s.Items[0].Tags[0] != "grocery" || s.Items[0].Tags[1] != "home" {
		t.Errorf("item 0: expected tags [grocery home], got %v", s.Items[0].Tags)
	}

	if s.Items[1].Status != StatusInProgress {
		t.Errorf("item 1: expected in_progress, got %s", s.Items[1].Status)
	}
	if s.Items[2].Status != StatusDone {
		t.Errorf("item 2: expected done, got %s", s.Items[2].Status)
	}
}

func TestImportAssignsNewIDs(t *testing.T) {
	s := tempStore(t)
	// Pre-populate the store with one item.
	s.Add("Existing item")

	csvData := `id,title,status,priority,due_date,tags,created_at,updated_at
99,Imported item,pending,,,,2025-01-01T00:00:00Z,2025-01-01T00:00:00Z
`
	count, err := s.Import(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 imported, got %d", count)
	}
	// The imported item should get ID 2, not 99.
	if s.Items[1].ID != 2 {
		t.Errorf("expected imported item ID=2, got %d", s.Items[1].ID)
	}
}

func TestImportEmptyCSV(t *testing.T) {
	s := tempStore(t)
	_, err := s.Import(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty CSV")
	}
}

func TestImportBadHeader(t *testing.T) {
	s := tempStore(t)
	_, err := s.Import(strings.NewReader("foo,bar\n"))
	if err == nil {
		t.Fatal("expected error for bad header")
	}
}

func TestImportInvalidStatus(t *testing.T) {
	s := tempStore(t)
	csvData := `id,title,status,priority,due_date,tags,created_at,updated_at
1,Bad status,invalid_status,,,,2025-01-01T00:00:00Z,2025-01-01T00:00:00Z
`
	_, err := s.Import(strings.NewReader(csvData))
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestImportInvalidPriority(t *testing.T) {
	s := tempStore(t)
	csvData := `id,title,status,priority,due_date,tags,created_at,updated_at
1,Bad priority,pending,ultra,,,,
`
	_, err := s.Import(strings.NewReader(csvData))
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestImportEmptyTitle(t *testing.T) {
	s := tempStore(t)
	csvData := `id,title,status,priority,due_date,tags,created_at,updated_at
1,,pending,,,,2025-01-01T00:00:00Z,2025-01-01T00:00:00Z
`
	_, err := s.Import(strings.NewReader(csvData))
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestImportRoundTrip(t *testing.T) {
	// Create a store, add items, export to CSV, import into a new store.
	s1 := tempStore(t)
	s1.AddFullWithTags("Task A", PriorityHigh, DueDate{}, []string{"work"})
	s1.AddFullWithTags("Task B", PriorityLow, DueDate{}, nil)
	s1.SetStatus(1, StatusDone)

	var buf bytes.Buffer
	if err := s1.Export(&buf); err != nil {
		t.Fatal(err)
	}

	s2 := tempStore(t)
	count, err := s2.Import(&buf)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 imported, got %d", count)
	}
	if s2.Items[0].Title != "Task A" || s2.Items[0].Status != StatusDone || s2.Items[0].Priority != PriorityHigh {
		t.Errorf("round-trip item 0 mismatch: %+v", s2.Items[0])
	}
	if len(s2.Items[0].Tags) != 1 || s2.Items[0].Tags[0] != "work" {
		t.Errorf("round-trip item 0 tags mismatch: %v", s2.Items[0].Tags)
	}
	if s2.Items[1].Title != "Task B" || s2.Items[1].Priority != PriorityLow {
		t.Errorf("round-trip item 1 mismatch: %+v", s2.Items[1])
	}
}

func TestUndoRestoresTags(t *testing.T) {
	s := tempStore(t)
	t.Cleanup(func() { os.Remove(s.undoFile()) })

	_, _ = s.AddFullWithTags("Tagged", PriorityNone, DueDate{}, []string{"work"})
	s.Save()

	if err := s.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.AddTag(1, "urgent"); err != nil {
		t.Fatal(err)
	}
	s.Save()

	got, _ := s.Get(1)
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags before undo, got %d", len(got.Tags))
	}

	if err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(1)
	if len(got.Tags) != 1 || got.Tags[0] != "work" {
		t.Errorf("expected [work] after undo, got %v", got.Tags)
	}
}
