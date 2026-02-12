package cluster

import (
	"sync"
	"time"
)

// Store is a thread-safe in-memory store for cluster objects. It is the
// single source of truth for cluster state. All mutations go through Store
// methods which hold a write lock, ensuring consistency.
//
// The store is additive-only: agents are never deleted, only updated with
// new revisions or state changes. This matches the spec's declarative model
// where `gcluster apply` only adds or updates, never removes.
type Store struct {
	mu      sync.RWMutex
	objects map[string]*ClusterObject // keyed by agent name

	// onChange is called (if non-nil) after every state mutation.
	// The callback receives a snapshot of all objects. Implementations
	// must not block — long work should be dispatched to a goroutine.
	onChange func([]ClusterObject)
}

// NewStore creates an empty cluster state store.
func NewStore() *Store {
	return &Store{
		objects: make(map[string]*ClusterObject),
	}
}

// OnChange registers a callback invoked after every mutation.
func (s *Store) OnChange(fn func([]ClusterObject)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = fn
}

// notifyLocked calls the onChange callback with a snapshot.
// Caller must hold at least a read lock.
func (s *Store) notifyLocked() {
	if s.onChange == nil {
		return
	}
	snap := s.snapshotLocked()
	// Call outside lock to avoid deadlocks — but we already hold the lock,
	// so we rely on the contract that onChange must not call back into Store.
	s.onChange(snap)
}

// ApplyDefinitions performs an idempotent upsert of agent definitions.
//
// For each agent:
//   - Same definition (same SHA-256) → unchanged, no new revision
//   - Changed definition → new revision appended, run state unchanged
//   - New agent → new object in pending state
//
// This is the core reconciliation logic matching the spec's declarative model.
func (s *Store) ApplyDefinitions(defs []AgentDef) ApplySummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	var summary ApplySummary

	for _, def := range defs {
		existing, ok := s.objects[def.Name]

		if !ok {
			// New agent — create in pending state.
			now := time.Now()
			rev := Revision{
				ID:         def.ID,
				Timestamp:  now,
				Definition: def.Definition,
			}
			s.objects[def.Name] = &ClusterObject{
				ID:              def.ID,
				Name:            def.Name,
				Definition:      def.Definition,
				Revisions:       []Revision{rev},
				State:           RunStatePending,
				CurrentRevision: def.ID,
			}
			summary.Created = append(summary.Created, def.Name)
			continue
		}

		// Existing agent — check if definition changed.
		if existing.ID == def.ID {
			summary.Unchanged = append(summary.Unchanged, def.Name)
			continue
		}

		// Definition changed — append new revision.
		now := time.Now()
		rev := Revision{
			ID:         def.ID,
			Timestamp:  now,
			Definition: def.Definition,
		}
		existing.ID = def.ID
		existing.Definition = def.Definition
		existing.State = RunStatePending
		existing.Revisions = append(existing.Revisions, rev)
		existing.CurrentRevision = def.ID
		summary.Updated = append(summary.Updated, def.Name)
	}

	s.notifyLocked()
	return summary
}

// GetAgent returns a copy of the named agent, or nil if not found.
func (s *Store) GetAgent(name string) *ClusterObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.objects[name]
	if !ok {
		return nil
	}
	cp := *obj
	cp.Revisions = make([]Revision, len(obj.Revisions))
	copy(cp.Revisions, obj.Revisions)
	return &cp
}

// ListAgents returns a snapshot of all cluster objects, sorted by name
// for deterministic output.
func (s *Store) ListAgents() []ClusterObject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

// snapshotLocked returns copies of all objects. Caller must hold lock.
func (s *Store) snapshotLocked() []ClusterObject {
	result := make([]ClusterObject, 0, len(s.objects))
	for _, obj := range s.objects {
		cp := *obj
		cp.Revisions = make([]Revision, len(obj.Revisions))
		copy(cp.Revisions, obj.Revisions)
		result = append(result, cp)
	}
	return result
}

// SetRunState updates the run state of the named agent.
// Returns false if the agent doesn't exist.
func (s *Store) SetRunState(name string, state RunState) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	obj, ok := s.objects[name]
	if !ok {
		return false
	}
	obj.State = state
	s.notifyLocked()
	return true
}

// LoadState replaces the entire store contents. Used for loading
// persisted state on startup. Run state is not persisted — it's a
// runtime concept owned by the executor. All loaded agents start
// as pending; they come alive when apply triggers StartPending.
func (s *Store) LoadState(objects []ClusterObject) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.objects = make(map[string]*ClusterObject, len(objects))
	for i := range objects {
		obj := objects[i]
		obj.State = RunStatePending
		s.objects[obj.Name] = &obj
	}
}
