package cluster

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Create and populate a store.
	s1 := NewStore()
	s1.ApplyDefinitions([]AgentDef{
		{Name: "alpha", Definition: "(defagent \"alpha\")", ID: "id-a"},
		{Name: "beta", Definition: "(defagent \"beta\")", ID: "id-b"},
	})
	s1.SetRunState("alpha", RunStateRunning)

	// Save.
	if err := SaveState(s1, path); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}

	// Load into a fresh store.
	s2 := NewStore()
	LoadState(s2, path)

	agents := s2.ListAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents after load, got %d", len(agents))
	}

	alpha := s2.GetAgent("alpha")
	if alpha == nil {
		t.Fatal("expected alpha to exist after load")
	}
	if alpha.State != RunStateRunning {
		t.Fatalf("expected alpha running, got %s", alpha.State)
	}
	if alpha.ID != "id-a" {
		t.Fatalf("expected alpha ID id-a, got %s", alpha.ID)
	}

	beta := s2.GetAgent("beta")
	if beta == nil {
		t.Fatal("expected beta to exist after load")
	}
	if beta.State != RunStatePending {
		t.Fatalf("expected beta pending, got %s", beta.State)
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	s := NewStore()
	LoadState(s, path) // Should not panic.

	if len(s.ListAgents()) != 0 {
		t.Fatal("expected empty store when file doesn't exist")
	}
}

func TestLoadStateCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write corrupt data.
	if err := os.WriteFile(path, []byte("not json at all {{{"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewStore()
	LoadState(s, path) // Should not panic, should start fresh.

	if len(s.ListAgents()) != 0 {
		t.Fatal("expected empty store after corrupt file")
	}

	// Corrupt file should be preserved.
	corrupt := path + ".corrupt"
	if _, err := os.Stat(corrupt); err != nil {
		t.Fatalf("corrupt file should be preserved at %s: %v", corrupt, err)
	}
}

func TestSaveStateCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "state.json")

	s := NewStore()
	if err := SaveState(s, path); err != nil {
		t.Fatalf("SaveState should create parent dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}
}

func TestSaveStateAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := NewStore()
	s.ApplyDefinitions([]AgentDef{
		{Name: "test", Definition: "(defagent \"test\")", ID: "id-t"},
	})

	if err := SaveState(s, path); err != nil {
		t.Fatal(err)
	}

	// Temp file should not remain.
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp file should not remain: %v", err)
	}
}
