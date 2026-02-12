package todo

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

type Item struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	Priority  Priority  `json:"priority,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	file   string
	NextID int    `json:"next_id"`
	Items  []Item `json:"items"`
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

func (s *Store) Add(title string) Item {
	return s.AddWithPriority(title, PriorityNone)
}

// AddWithPriority creates a new item with the given title and priority.
func (s *Store) AddWithPriority(title string, priority Priority) Item {
	now := time.Now()
	item := Item{
		ID:        s.nextID(),
		Title:     title,
		Status:    StatusPending,
		Priority:  priority,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.Items = append(s.Items, item)
	return item
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
	item, err := s.Get(id)
	if err != nil {
		return err
	}
	item.Title = newTitle
	item.UpdatedAt = time.Now()
	return nil
}

// SetPriority updates the priority of an existing item.
func (s *Store) SetPriority(id int, priority Priority) error {
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
func (s *Store) Search(query string) []Item {
	lower := strings.ToLower(query)
	var result []Item
	for _, item := range s.Items {
		if strings.Contains(strings.ToLower(item.Title), lower) {
			result = append(result, item)
		}
	}
	return result
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
// Columns: id, title, status, created_at, updated_at.
func (s *Store) Export(w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header row
	if err := cw.Write([]string{"id", "title", "status", "priority", "created_at", "updated_at"}); err != nil {
		return err
	}

	for _, item := range s.Items {
		record := []string{
			strconv.Itoa(item.ID),
			item.Title,
			string(item.Status),
			string(item.Priority),
			item.CreatedAt.Format(time.RFC3339),
			item.UpdatedAt.Format(time.RFC3339),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	return cw.Error()
}

func (s *Store) List(filter Status) ([]Item, error) {
	if filter == "" {
		return s.Items, nil
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
