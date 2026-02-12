package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// DefaultStateDir returns the default directory for cluster state files.
func DefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".gcluster")
}

// DefaultStatePath returns the default path for the cluster state file.
func DefaultStatePath() string {
	return filepath.Join(DefaultStateDir(), "state.json")
}

// persistedState is the on-disk JSON format for cluster state.
type persistedState struct {
	Objects []ClusterObject `json:"objects"`
}

// SaveState writes the current store contents to disk as JSON.
// It creates the parent directory if needed. Writes are atomic:
// data goes to a temp file first, then renamed into place.
func SaveState(store *Store, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	objects := store.ListAgents()
	state := persistedState{Objects: objects}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Atomic write: temp file + rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// LoadState reads persisted cluster state from disk into the store.
// Per spec, if the file is unreadable or corrupt, start fresh and log
// a warning. The old file is preserved (renamed to .corrupt) for debugging.
func LoadState(store *Store, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file — fresh start, not an error.
			return
		}
		log.Printf("warning: cannot read state file %s: %v (starting fresh)", path, err)
		preserveCorrupt(path)
		return
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("warning: corrupt state file %s: %v (starting fresh)", path, err)
		preserveCorrupt(path)
		return
	}

	store.LoadState(state.Objects)
	log.Printf("loaded %d agents from %s", len(state.Objects), path)
}

// preserveCorrupt renames a corrupt state file so it can be inspected later.
func preserveCorrupt(path string) {
	corrupt := path + ".corrupt"
	if err := os.Rename(path, corrupt); err != nil {
		log.Printf("warning: could not preserve corrupt state file: %v", err)
	} else {
		log.Printf("preserved corrupt state file as %s", corrupt)
	}
}
