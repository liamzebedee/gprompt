package todo

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultFile = "todos.json"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Priority string

const (
	PriorityNone   Priority = ""
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// ValidPriority reports whether p is a recognised priority value (including empty).
func ValidPriority(p Priority) bool {
	switch p {
	case PriorityNone, PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// DueDate is a date-only wrapper (no time component) that serialises as "YYYY-MM-DD".
type DueDate struct {
	time.Time
	Valid bool // false means no due date set
}

// DueDateLayout is the expected format for parsing/formatting due dates.
const DueDateLayout = "2006-01-02"

// ParseDueDate parses a "YYYY-MM-DD" string into a DueDate.
func ParseDueDate(s string) (DueDate, error) {
	if s == "" {
		return DueDate{}, nil
	}
	t, err := time.Parse(DueDateLayout, s)
	if err != nil {
		return DueDate{}, fmt.Errorf("invalid due date %q (expected YYYY-MM-DD): %w", s, err)
	}
	return DueDate{Time: t, Valid: true}, nil
}

func (d DueDate) String() string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(DueDateLayout)
}

func (d DueDate) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(d.Time.Format(DueDateLayout))
}

func (d *DueDate) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == nil || *s == "" {
		d.Valid = false
		return nil
	}
	t, err := time.Parse(DueDateLayout, *s)
	if err != nil {
		return err
	}
	d.Time = t
	d.Valid = true
	return nil
}

type Item struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	Priority  Priority  `json:"priority,omitempty"`
	DueDate   DueDate   `json:"due_date,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HasTag reports whether the item has the given tag (case-insensitive).
func (it *Item) HasTag(tag string) bool {
	lower := strings.ToLower(tag)
	for _, t := range it.Tags {
		if strings.ToLower(t) == lower {
			return true
		}
	}
	return false
}

type Store struct {
	file   string
	NextID int    `json:"next_id"`
	Items  []Item `json:"items"`
}

// undoFile returns the path to the undo backup file for this store.
func (s *Store) undoFile() string {
	return s.file + ".undo"
}

// Snapshot saves the current state to an undo backup file so it can be restored later.
func (s *Store) Snapshot() error {
	envelope := struct {
		NextID int    `json:"next_id"`
		Items  []Item `json:"items"`
	}{
		NextID: s.NextID,
		Items:  s.Items,
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.undoFile(), data, 0644)
}

// Undo restores the store to the state saved by the last Snapshot call.
// It removes the undo file after a successful restore.
func (s *Store) Undo() error {
	data, err := os.ReadFile(s.undoFile())
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("nothing to undo")
		}
		return err
	}
	var raw struct {
		NextID int    `json:"next_id"`
		Items  []Item `json:"items"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("corrupted undo file: %w", err)
	}
	s.NextID = raw.NextID
	s.Items = raw.Items
	if err := s.Save(); err != nil {
		return err
	}
	return os.Remove(s.undoFile())
}

func NewStore(file string) *Store {
	return &Store{file: file}
}

func (s *Store) Load() error {
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			s.Items = []Item{}
			return nil
		}
		return err
	}
	if len(data) == 0 {
		s.Items = []Item{}
		return nil
	}
	// Try new format (object with next_id + items) first.
	var raw struct {
		NextID int    `json:"next_id"`
		Items  []Item `json:"items"`
	}
	if err := json.Unmarshal(data, &raw); err == nil && raw.Items != nil {
		s.NextID = raw.NextID
		s.Items = raw.Items
		return nil
	}
	// Fall back to legacy format (bare array of items).
	if err := json.Unmarshal(data, &s.Items); err != nil {
		return err
	}
	// Recover NextID from the max existing ID.
	for _, item := range s.Items {
		if item.ID >= s.NextID {
			s.NextID = item.ID + 1
		}
	}
	return nil
}

func (s *Store) Save() error {
	envelope := struct {
		NextID int    `json:"next_id"`
		Items  []Item `json:"items"`
	}{
		NextID: s.NextID,
		Items:  s.Items,
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.file, data, 0644)
}

func (s *Store) nextID() int {
	id := s.NextID
	if id == 0 {
		id = 1
	}
	s.NextID = id + 1
	return id
}

func (s *Store) Add(title string) (Item, error) {
	return s.AddFull(title, PriorityNone, DueDate{})
}

// AddWithPriority creates a new item with the given title and priority.
func (s *Store) AddWithPriority(title string, priority Priority) (Item, error) {
	return s.AddFull(title, priority, DueDate{})
}

// AddFull creates a new item with the given title, priority, and optional due date.
func (s *Store) AddFull(title string, priority Priority, due DueDate) (Item, error) {
	return s.AddFullWithTags(title, priority, due, nil)
}

// AddFullWithTags creates a new item with the given title, priority, due date, and tags.
func (s *Store) AddFullWithTags(title string, priority Priority, due DueDate, tags []string) (Item, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Item{}, fmt.Errorf("title must not be empty")
	}
	if !ValidPriority(priority) {
		return Item{}, fmt.Errorf("invalid priority: %q (valid values: low, medium, high, or empty to clear)", priority)
	}
	// Validate and normalise tags.
	for _, t := range tags {
		if strings.Contains(t, ";") {
			return Item{}, fmt.Errorf("tag must not contain semicolon: %q", t)
		}
	}
	cleaned := normaliseTags(tags)
	now := time.Now()
	item := Item{
		ID:        s.nextID(),
		Title:     title,
		Status:    StatusPending,
		Priority:  priority,
		DueDate:   due,
		Tags:      cleaned,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Items = append(s.Items, item)
	return item, nil
}

// SetDueDate updates the due date of an existing item. Pass an empty DueDate to clear it.
func (s *Store) SetDueDate(id int, due DueDate) error {
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.DueDate = due
	item.UpdatedAt = time.Now()
	return nil
}

func (s *Store) Get(id int) (*Item, error) {
	for i := range s.Items {
		if s.Items[i].ID == id {
			return &s.Items[i], nil
		}
	}
	return nil, fmt.Errorf("todo #%d not found", id)
}

func (s *Store) SetStatus(id int, status Status) error {
	if !ValidStatus(status) {
		return fmt.Errorf("invalid status: %q (valid values: pending, in_progress, done)", status)
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Status = status
	item.UpdatedAt = time.Now()
	return nil
}

// Edit updates the title of an existing item.
func (s *Store) Edit(id int, newTitle string) error {
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" {
		return fmt.Errorf("title must not be empty")
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Title = newTitle
	item.UpdatedAt = time.Now()
	return nil
}

// SetNote sets or clears the note on an existing item.
// Pass an empty string to clear the note.
func (s *Store) SetNote(id int, note string) error {
	note = strings.TrimSpace(note)
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Note = note
	item.UpdatedAt = time.Now()
	return nil
}

// SetPriority updates the priority of an existing item.
func (s *Store) SetPriority(id int, priority Priority) error {
	if !ValidPriority(priority) {
		return fmt.Errorf("invalid priority: %q (valid values: low, medium, high, or empty to clear)", priority)
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Priority = priority
	item.UpdatedAt = time.Now()
	return nil
}

func (s *Store) Delete(id int) error {
	for i, item := range s.Items {
		if item.ID == id {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo #%d not found", id)
}

// Stats returns a map of status to count for all items.
func (s *Store) Stats() map[Status]int {
	counts := map[Status]int{
		StatusPending:    0,
		StatusInProgress: 0,
		StatusDone:       0,
	}
	for _, item := range s.Items {
		counts[item.Status]++
	}
	return counts
}

// Search returns items whose title contains the query (case-insensitive).
// It returns an error if the query is empty or whitespace-only.
func (s *Store) Search(query string) ([]Item, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query must not be empty")
	}
	lower := strings.ToLower(query)
	var result []Item
	for _, item := range s.Items {
		if strings.Contains(strings.ToLower(item.Title), lower) {
			result = append(result, item)
		}
	}
	return result, nil
}

// ParseAddTitle joins all arguments after the command into a single title string.
// This allows users to type multi-word titles without quoting, e.g. "todo add Buy some milk".
func ParseAddTitle(args []string) string {
	return strings.Join(args, " ")
}

// ValidStatus reports whether s is one of the recognised status values.
func ValidStatus(s Status) bool {
	switch s {
	case StatusPending, StatusInProgress, StatusDone:
		return true
	}
	return false
}

// Export writes all items as CSV to the given writer.
// Columns: id, title, status, priority, due_date, tags, note, created_at, updated_at.
func (s *Store) Export(w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row
	if err := cw.Write([]string{"id", "title", "status", "priority", "due_date", "tags", "note", "created_at", "updated_at"}); err != nil {
		return err
	}

	for _, item := range s.Items {
		tagsStr := strings.Join(item.Tags, ";")
		record := []string{
			strconv.Itoa(item.ID),
			item.Title,
			string(item.Status),
			string(item.Priority),
			item.DueDate.String(),
			tagsStr,
			item.Note,
			item.CreatedAt.Format(time.RFC3339),
			item.UpdatedAt.Format(time.RFC3339),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	return cw.Error()
}

// ClearDone removes all items with StatusDone and returns the number removed.
func (s *Store) ClearDone() int {
	kept := make([]Item, 0, len(s.Items))
	removed := 0
	for _, item := range s.Items {
		if item.Status == StatusDone {
			removed++
		} else {
			kept = append(kept, item)
		}
	}
	s.Items = kept
	return removed
}

// SortField represents a field by which items can be sorted.
type SortField string

const (
	SortByPriority SortField = "priority"
	SortByDue      SortField = "due"
	SortByStatus   SortField = "status"
	SortByCreated  SortField = "created"
)

// ValidSortField reports whether f is a recognised sort field.
func ValidSortField(f SortField) bool {
	switch f {
	case SortByPriority, SortByDue, SortByStatus, SortByCreated:
		return true
	}
	return false
}

// priorityRank returns a numeric rank for sorting (higher priority = lower rank = sorts first).
func priorityRank(p Priority) int {
	switch p {
	case PriorityHigh:
		return 0
	case PriorityMedium:
		return 1
	case PriorityLow:
		return 2
	default:
		return 3 // no priority sorts last
	}
}

// statusRank returns a numeric rank for sorting (in_progress first, then pending, then done).
func statusRank(s Status) int {
	switch s {
	case StatusInProgress:
		return 0
	case StatusPending:
		return 1
	case StatusDone:
		return 2
	default:
		return 3
	}
}

// Sort reorders items in-place by the given field. Returns an error if the field is invalid.
func (s *Store) Sort(field SortField) error {
	if !ValidSortField(field) {
		return fmt.Errorf("invalid sort field: %q (valid values: priority, due, status, created)", field)
	}
	sort.SliceStable(s.Items, func(i, j int) bool {
		a, b := s.Items[i], s.Items[j]
		switch field {
		case SortByPriority:
			return priorityRank(a.Priority) < priorityRank(b.Priority)
		case SortByDue:
			// Items with due dates sort before items without.
			if a.DueDate.Valid != b.DueDate.Valid {
				return a.DueDate.Valid
			}
			if a.DueDate.Valid && b.DueDate.Valid {
				return a.DueDate.Time.Before(b.DueDate.Time)
			}
			return false
		case SortByStatus:
			return statusRank(a.Status) < statusRank(b.Status)
		case SortByCreated:
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return false
	})
	return nil
}

func (s *Store) List(filter Status) ([]Item, error) {
	if filter == "" {
		result := make([]Item, len(s.Items))
		copy(result, s.Items)
		return result, nil
	}
	if !ValidStatus(filter) {
		return nil, fmt.Errorf("invalid status filter: %q (valid values: pending, in_progress, done)", filter)
	}
	var result []Item
	for _, item := range s.Items {
		if item.Status == filter {
			result = append(result, item)
		}
	}
	return result, nil
}

// ListWithTag returns items matching both the status filter and tag filter.
// An empty status filter matches all statuses. The tag is validated and must
// not be empty or whitespace-only. This combines the functionality of List and
// ListByTag into a single call with proper validation of both filters.
func (s *Store) ListWithTag(statusFilter Status, tag string) ([]Item, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, fmt.Errorf("tag filter must not be empty")
	}
	if statusFilter != "" && !ValidStatus(statusFilter) {
		return nil, fmt.Errorf("invalid status filter: %q (valid values: pending, in_progress, done)", statusFilter)
	}
	var result []Item
	for _, item := range s.Items {
		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		if item.HasTag(tag) {
			result = append(result, item)
		}
	}
	return result, nil
}

// normaliseTags trims, lowercases, and deduplicates tags, dropping empty values.
func normaliseTags(tags []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// ValidTag reports whether tag is a non-empty, trimmed string without semicolons.
// Semicolons are reserved as the tag separator in CSV export/import.
func ValidTag(tag string) bool {
	t := strings.TrimSpace(tag)
	return t != "" && !strings.Contains(t, ";")
}

// AddTag adds a tag to an existing item. Duplicate tags are ignored (case-insensitive).
func (s *Store) AddTag(id int, tag string) error {
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag == "" {
		return fmt.Errorf("tag must not be empty")
	}
	if strings.Contains(tag, ";") {
		return fmt.Errorf("tag must not contain semicolon: %q", tag)
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	if item.HasTag(tag) {
		return nil // already tagged
	}
	item.Tags = append(item.Tags, tag)
	item.UpdatedAt = time.Now()
	return nil
}

// RemoveTag removes a tag from an existing item. Returns an error if the tag is not present.
func (s *Store) RemoveTag(id int, tag string) error {
	tag = strings.TrimSpace(strings.ToLower(tag))
	if tag == "" {
		return fmt.Errorf("tag must not be empty")
	}
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	idx := -1
	for i, t := range item.Tags {
		if strings.ToLower(t) == tag {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("tag %q not found on #%d", tag, id)
	}
	item.Tags = append(item.Tags[:idx], item.Tags[idx+1:]...)
	item.UpdatedAt = time.Now()
	return nil
}

// RenameTag renames a tag across all items in the store. Both old and new tag
// names are normalised (trimmed, lowercased). Returns the number of items updated.
func (s *Store) RenameTag(oldTag, newTag string) (int, error) {
	oldTag = strings.TrimSpace(strings.ToLower(oldTag))
	newTag = strings.TrimSpace(strings.ToLower(newTag))
	if oldTag == "" {
		return 0, fmt.Errorf("old tag must not be empty")
	}
	if newTag == "" {
		return 0, fmt.Errorf("new tag must not be empty")
	}
	if strings.Contains(newTag, ";") {
		return 0, fmt.Errorf("tag must not contain semicolon: %q", newTag)
	}
	if oldTag == newTag {
		return 0, fmt.Errorf("old and new tag are the same: %q", oldTag)
	}
	count := 0
	now := time.Now()
	for i := range s.Items {
		item := &s.Items[i]
		idx := -1
		for j, t := range item.Tags {
			if strings.ToLower(t) == oldTag {
				idx = j
				break
			}
		}
		if idx == -1 {
			continue
		}
		// If the item already has the new tag, just remove the old one to avoid duplicates.
		if item.HasTag(newTag) {
			item.Tags = append(item.Tags[:idx], item.Tags[idx+1:]...)
		} else {
			item.Tags[idx] = newTag
		}
		item.UpdatedAt = now
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("no items have tag %q", oldTag)
	}
	return count, nil
}

// ListByTag returns items that have the given tag (case-insensitive).
func (s *Store) ListByTag(tag string) ([]Item, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, fmt.Errorf("tag filter must not be empty")
	}
	var result []Item
	for _, item := range s.Items {
		if item.HasTag(tag) {
			result = append(result, item)
		}
	}
	return result, nil
}

// Import reads items from a CSV reader and adds them to the store.
// The CSV must have a header row matching the export format:
// id, title, status, priority, due_date, tags, note, created_at, updated_at.
// Imported items receive new IDs; the original IDs are ignored.
// Returns the number of items imported and any error encountered.
func (s *Store) Import(r io.Reader) (int, error) {
	cr := csv.NewReader(r)

	// Read and validate header.
	header, err := cr.Read()
	if err != nil {
		if err == io.EOF {
			return 0, fmt.Errorf("empty CSV file")
		}
		return 0, fmt.Errorf("reading CSV header: %w", err)
	}
	expected := []string{"id", "title", "status", "priority", "due_date", "tags", "note", "created_at", "updated_at"}
	if len(header) != len(expected) {
		return 0, fmt.Errorf("CSV header has %d columns, expected %d", len(header), len(expected))
	}
	for i, col := range expected {
		if strings.TrimSpace(strings.ToLower(header[i])) != col {
			return 0, fmt.Errorf("CSV column %d: expected %q, got %q", i+1, col, header[i])
		}
	}

	count := 0
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("reading CSV row %d: %w", count+2, err)
		}
		if len(record) != len(expected) {
			return count, fmt.Errorf("CSV row %d has %d columns, expected %d", count+2, len(record), len(expected))
		}

		title := strings.TrimSpace(record[1])
		if title == "" {
			return count, fmt.Errorf("CSV row %d: title must not be empty", count+2)
		}

		status := Status(strings.TrimSpace(record[2]))
		if !ValidStatus(status) {
			return count, fmt.Errorf("CSV row %d: invalid status %q", count+2, record[2])
		}

		priority := Priority(strings.TrimSpace(record[3]))
		if !ValidPriority(priority) {
			return count, fmt.Errorf("CSV row %d: invalid priority %q", count+2, record[3])
		}

		due, err := ParseDueDate(strings.TrimSpace(record[4]))
		if err != nil {
			return count, fmt.Errorf("CSV row %d: %w", count+2, err)
		}

		var tags []string
		tagsStr := strings.TrimSpace(record[5])
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ";")
		}

		note := record[6]

		// Parse timestamps from CSV; fall back to now if missing or invalid.
		now := time.Now()
		createdAt := now
		if ts := strings.TrimSpace(record[7]); ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				createdAt = parsed
			}
		}
		updatedAt := now
		if ts := strings.TrimSpace(record[8]); ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				updatedAt = parsed
			}
		}

		item := Item{
			ID:        s.nextID(),
			Title:     title,
			Status:    status,
			Priority:  priority,
			DueDate:   due,
			Tags:      normaliseTags(tags),
			Note:      note,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		s.Items = append(s.Items, item)
		count++
	}

	return count, nil
}

// Overdue returns all non-done items whose due date is before today.
func (s *Store) Overdue() []Item {
	today := time.Now().Truncate(24 * time.Hour)
	var result []Item
	for _, item := range s.Items {
		if item.Status != StatusDone && item.DueDate.Valid && item.DueDate.Time.Truncate(24*time.Hour).Before(today) {
			result = append(result, item)
		}
	}
	return result
}

// Upcoming returns all non-done items whose due date is today or within the next `days` days.
// Items with no due date are excluded. A days value of 0 means today only.
// Returns an error if days is negative.
func (s *Store) Upcoming(days int) ([]Item, error) {
	if days < 0 {
		return nil, fmt.Errorf("days must not be negative: %d", days)
	}
	today := time.Now().Truncate(24 * time.Hour)
	horizon := today.AddDate(0, 0, days)
	var result []Item
	for _, item := range s.Items {
		if item.Status == StatusDone || !item.DueDate.Valid {
			continue
		}
		due := item.DueDate.Time.Truncate(24 * time.Hour)
		if (due.Equal(today) || due.After(today)) && (due.Equal(horizon) || due.Before(horizon)) {
			result = append(result, item)
		}
	}
	return result, nil
}

// archiveFile returns the path to the archive file for this store.
func (s *Store) archiveFile() string {
	return s.file + ".archive"
}

// Archive moves all completed (done) items to a separate archive file and
// removes them from the active store. The archive file uses the same JSON
// envelope format as the main store (next_id + items). Archived items keep
// their original IDs. Returns the number of items archived.
func (s *Store) Archive() (int, error) {
	// Collect done items.
	var archived []Item
	kept := make([]Item, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Status == StatusDone {
			archived = append(archived, item)
		} else {
			kept = append(kept, item)
		}
	}
	if len(archived) == 0 {
		return 0, nil
	}

	// Load existing archive (if any) and append.
	archStore := NewStore(s.archiveFile())
	if err := archStore.Load(); err != nil {
		return 0, fmt.Errorf("loading archive file: %w", err)
	}
	archStore.Items = append(archStore.Items, archived...)
	// Keep NextID in the archive in sync so IDs don't collide if the archive
	// is ever imported back.
	if s.NextID > archStore.NextID {
		archStore.NextID = s.NextID
	}
	if err := archStore.Save(); err != nil {
		return 0, fmt.Errorf("saving archive file: %w", err)
	}

	s.Items = kept
	return len(archived), nil
}

// Swap exchanges the positions of two items identified by their IDs.
// Both IDs must exist and be different.
func (s *Store) Swap(id1, id2 int) error {
	if id1 == id2 {
		return fmt.Errorf("cannot swap an item with itself")
	}
	idx1, idx2 := -1, -1
	for i := range s.Items {
		if s.Items[i].ID == id1 {
			idx1 = i
		}
		if s.Items[i].ID == id2 {
			idx2 = i
		}
	}
	if idx1 == -1 {
		return fmt.Errorf("todo #%d not found", id1)
	}
	if idx2 == -1 {
		return fmt.Errorf("todo #%d not found", id2)
	}
	s.Items[idx1], s.Items[idx2] = s.Items[idx2], s.Items[idx1]
	return nil
}

// BulkDone marks multiple items as done in a single operation. It requires at
// least one ID. All IDs are validated before any changes are made so the
// operation is atomic — either all items are marked done or none are.
// Returns the number of items whose status actually changed.
func (s *Store) BulkDone(ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("at least one ID is required")
	}
	// Check for duplicate IDs.
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return 0, fmt.Errorf("duplicate ID: %d", id)
		}
		seen[id] = true
	}
	// Validate that all IDs exist before making any changes.
	items := make([]*Item, 0, len(ids))
	for _, id := range ids {
		item, err := s.Get(id)
		if err != nil {
			return 0, err
		}
		items = append(items, item)
	}
	// Apply changes.
	now := time.Now()
	changed := 0
	for _, item := range items {
		if item.Status != StatusDone {
			item.Status = StatusDone
			item.UpdatedAt = now
			changed++
		}
	}
	return changed, nil
}

// Duplicate creates a new pending item that copies the title, priority, due date,
// tags, and note from the item with the given ID. The new item gets a fresh ID
// and timestamps. Returns the newly created item.
func (s *Store) Duplicate(id int) (Item, error) {
	src, err := s.Get(id)
	if err != nil {
		return Item{}, err
	}
	// Copy tags slice to avoid sharing the backing array.
	var tags []string
	if len(src.Tags) > 0 {
		tags = make([]string, len(src.Tags))
		copy(tags, src.Tags)
	}
	now := time.Now()
	item := Item{
		ID:        s.nextID(),
		Title:     src.Title,
		Status:    StatusPending,
		Priority:  src.Priority,
		DueDate:   src.DueDate,
		Tags:      tags,
		Note:      src.Note,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Items = append(s.Items, item)
	return item, nil
}

// TimelineBucket represents a group of items in the timeline view.
type TimelineBucket struct {
	Label string
	Items []Item
}

// Timeline groups non-done items by due-date urgency and returns them as
// ordered buckets: Overdue, Today, This Week (2–7 days), Later, and No Due Date.
// Done items are excluded from the timeline.
func (s *Store) Timeline() []TimelineBucket {
	today := time.Now().Truncate(24 * time.Hour)
	weekEnd := today.AddDate(0, 0, 7)

	var overdue, todayItems, thisWeek, later, noDue []Item
	for _, item := range s.Items {
		if item.Status == StatusDone {
			continue
		}
		if !item.DueDate.Valid {
			noDue = append(noDue, item)
			continue
		}
		due := item.DueDate.Time.Truncate(24 * time.Hour)
		switch {
		case due.Before(today):
			overdue = append(overdue, item)
		case due.Equal(today):
			todayItems = append(todayItems, item)
		case due.Before(weekEnd) || due.Equal(weekEnd):
			thisWeek = append(thisWeek, item)
		default:
			later = append(later, item)
		}
	}

	var buckets []TimelineBucket
	if len(overdue) > 0 {
		buckets = append(buckets, TimelineBucket{Label: "Overdue", Items: overdue})
	}
	if len(todayItems) > 0 {
		buckets = append(buckets, TimelineBucket{Label: "Today", Items: todayItems})
	}
	if len(thisWeek) > 0 {
		buckets = append(buckets, TimelineBucket{Label: "This Week", Items: thisWeek})
	}
	if len(later) > 0 {
		buckets = append(buckets, TimelineBucket{Label: "Later", Items: later})
	}
	if len(noDue) > 0 {
		buckets = append(buckets, TimelineBucket{Label: "No Due Date", Items: noDue})
	}
	return buckets
}

// TagGroup represents a group of items sharing a common tag.
type TagGroup struct {
	Tag   string
	Items []Item
}

// GroupByTag returns items grouped by tag, sorted alphabetically by tag name.
// Items with multiple tags appear in each relevant group. Items with no tags
// are collected under a special group with an empty Tag at the end (if any exist).
func (s *Store) GroupByTag() []TagGroup {
	tagMap := map[string][]Item{}
	var untagged []Item
	for _, item := range s.Items {
		if len(item.Tags) == 0 {
			untagged = append(untagged, item)
			continue
		}
		for _, tag := range item.Tags {
			lower := strings.ToLower(tag)
			tagMap[lower] = append(tagMap[lower], item)
		}
	}

	// Collect tag names and sort alphabetically.
	tagNames := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)

	groups := make([]TagGroup, 0, len(tagNames)+1)
	for _, tag := range tagNames {
		groups = append(groups, TagGroup{Tag: tag, Items: tagMap[tag]})
	}
	if len(untagged) > 0 {
		groups = append(groups, TagGroup{Tag: "", Items: untagged})
	}
	return groups
}

// FormatTags returns a comma-separated string of tags, or "-" if empty.
func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ", ")
}
