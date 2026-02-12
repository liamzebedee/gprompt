package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const DefaultFile = "todos.json"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Item struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	file  string
	Items []Item `json:"items"`
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
	return json.Unmarshal(data, &s.Items)
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.Items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.file, data, 0644)
}

func (s *Store) nextID() int {
	max := 0
	for _, item := range s.Items {
		if item.ID > max {
			max = item.ID
		}
	}
	return max + 1
}

func (s *Store) Add(title string) Item {
	now := time.Now()
	item := Item{
		ID:        s.nextID(),
		Title:     title,
		Status:    StatusPending,
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

func (s *Store) Delete(id int) error {
	for i, item := range s.Items {
		if item.ID == id {
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo #%d not found", id)
}

func (s *Store) List(filter Status) []Item {
	if filter == "" {
		return s.Items
	}
	var result []Item
	for _, item := range s.Items {
		if item.Status == filter {
			result = append(result, item)
		}
	}
	return result
}
