package cluster

import (
	"sort"
	"testing"
)

func TestNewStoreIsEmpty(t *testing.T) {
	s := NewStore()
	agents := s.ListAgents()
	if len(agents) != 0 {
		t.Fatalf("new store should be empty, got %d agents", len(agents))
	}
}

func TestApplyCreatesNewAgent(t *testing.T) {
	s := NewStore()
	defs := []AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\" ...)", ID: "abc123"},
	}

	summary := s.ApplyDefinitions(defs)

	if len(summary.Created) != 1 || summary.Created[0] != "watcher" {
		t.Fatalf("expected 1 created (watcher), got %v", summary.Created)
	}
	if len(summary.Updated) != 0 {
		t.Fatalf("expected 0 updated, got %v", summary.Updated)
	}
	if len(summary.Unchanged) != 0 {
		t.Fatalf("expected 0 unchanged, got %v", summary.Unchanged)
	}

	agent := s.GetAgent("watcher")
	if agent == nil {
		t.Fatal("expected agent to exist")
	}
	if agent.State != RunStatePending {
		t.Fatalf("expected pending state, got %s", agent.State)
	}
	if agent.ID != "abc123" {
		t.Fatalf("expected ID abc123, got %s", agent.ID)
	}
	if len(agent.Revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(agent.Revisions))
	}
	if agent.CurrentRevision != "abc123" {
		t.Fatalf("expected current revision abc123, got %s", agent.CurrentRevision)
	}
}

func TestApplyIdempotent(t *testing.T) {
	s := NewStore()
	defs := []AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\" ...)", ID: "abc123"},
	}

	// First apply creates.
	s.ApplyDefinitions(defs)

	// Second apply is unchanged.
	summary := s.ApplyDefinitions(defs)
	if len(summary.Created) != 0 {
		t.Fatalf("expected 0 created on reapply, got %v", summary.Created)
	}
	if len(summary.Unchanged) != 1 || summary.Unchanged[0] != "watcher" {
		t.Fatalf("expected 1 unchanged (watcher), got %v", summary.Unchanged)
	}

	// Still only 1 revision.
	agent := s.GetAgent("watcher")
	if len(agent.Revisions) != 1 {
		t.Fatalf("expected 1 revision after idempotent apply, got %d", len(agent.Revisions))
	}
}

func TestApplyUpdatesExistingAgent(t *testing.T) {
	s := NewStore()
	s.ApplyDefinitions([]AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\" v1)", ID: "id-v1"},
	})

	// Change definition.
	summary := s.ApplyDefinitions([]AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\" v2)", ID: "id-v2"},
	})

	if len(summary.Updated) != 1 || summary.Updated[0] != "watcher" {
		t.Fatalf("expected 1 updated (watcher), got %v", summary.Updated)
	}

	agent := s.GetAgent("watcher")
	if len(agent.Revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(agent.Revisions))
	}
	if agent.ID != "id-v2" {
		t.Fatalf("expected current ID id-v2, got %s", agent.ID)
	}
	if agent.CurrentRevision != "id-v2" {
		t.Fatalf("expected current revision id-v2, got %s", agent.CurrentRevision)
	}
}

func TestApplyMultipleAgents(t *testing.T) {
	s := NewStore()
	defs := []AgentDef{
		{Name: "alpha", Definition: "(defagent \"alpha\")", ID: "id-a"},
		{Name: "beta", Definition: "(defagent \"beta\")", ID: "id-b"},
		{Name: "gamma", Definition: "(defagent \"gamma\")", ID: "id-c"},
	}

	summary := s.ApplyDefinitions(defs)
	if len(summary.Created) != 3 {
		t.Fatalf("expected 3 created, got %d", len(summary.Created))
	}

	agents := s.ListAgents()
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}
}

func TestApplyMixedCreateUpdateUnchanged(t *testing.T) {
	s := NewStore()

	// Create alpha and beta.
	s.ApplyDefinitions([]AgentDef{
		{Name: "alpha", Definition: "(defagent \"alpha\" v1)", ID: "a-v1"},
		{Name: "beta", Definition: "(defagent \"beta\" v1)", ID: "b-v1"},
	})

	// Apply: alpha unchanged, beta updated, gamma new.
	summary := s.ApplyDefinitions([]AgentDef{
		{Name: "alpha", Definition: "(defagent \"alpha\" v1)", ID: "a-v1"},
		{Name: "beta", Definition: "(defagent \"beta\" v2)", ID: "b-v2"},
		{Name: "gamma", Definition: "(defagent \"gamma\" v1)", ID: "c-v1"},
	})

	if len(summary.Unchanged) != 1 || summary.Unchanged[0] != "alpha" {
		t.Fatalf("expected alpha unchanged, got %v", summary.Unchanged)
	}
	if len(summary.Updated) != 1 || summary.Updated[0] != "beta" {
		t.Fatalf("expected beta updated, got %v", summary.Updated)
	}
	if len(summary.Created) != 1 || summary.Created[0] != "gamma" {
		t.Fatalf("expected gamma created, got %v", summary.Created)
	}
}

func TestSetRunState(t *testing.T) {
	s := NewStore()
	s.ApplyDefinitions([]AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\")", ID: "abc"},
	})

	ok := s.SetRunState("watcher", RunStateRunning)
	if !ok {
		t.Fatal("expected SetRunState to succeed")
	}

	agent := s.GetAgent("watcher")
	if agent.State != RunStateRunning {
		t.Fatalf("expected running state, got %s", agent.State)
	}

	// Non-existent agent.
	ok = s.SetRunState("nonexistent", RunStateRunning)
	if ok {
		t.Fatal("expected SetRunState to fail for nonexistent agent")
	}
}

func TestGetAgentReturnsCopy(t *testing.T) {
	s := NewStore()
	s.ApplyDefinitions([]AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\")", ID: "abc"},
	})

	a1 := s.GetAgent("watcher")
	a1.State = RunStateStopped

	a2 := s.GetAgent("watcher")
	if a2.State != RunStatePending {
		t.Fatal("modifying returned copy should not affect store")
	}
}

func TestGetAgentNotFound(t *testing.T) {
	s := NewStore()
	if s.GetAgent("nonexistent") != nil {
		t.Fatal("expected nil for nonexistent agent")
	}
}

func TestListAgentsReturnsCopies(t *testing.T) {
	s := NewStore()
	s.ApplyDefinitions([]AgentDef{
		{Name: "alpha", Definition: "(defagent \"alpha\")", ID: "a"},
		{Name: "beta", Definition: "(defagent \"beta\")", ID: "b"},
	})

	agents := s.ListAgents()
	sort.Slice(agents, func(i, j int) bool { return agents[i].Name < agents[j].Name })

	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].Name != "alpha" || agents[1].Name != "beta" {
		t.Fatalf("unexpected agent names: %s, %s", agents[0].Name, agents[1].Name)
	}
}

func TestOnChangeCallback(t *testing.T) {
	s := NewStore()
	var callCount int
	var lastObjects []ClusterObject

	s.OnChange(func(objects []ClusterObject) {
		callCount++
		lastObjects = objects
	})

	s.ApplyDefinitions([]AgentDef{
		{Name: "watcher", Definition: "(defagent \"watcher\")", ID: "abc"},
	})

	if callCount != 1 {
		t.Fatalf("expected onChange called once, got %d", callCount)
	}
	if len(lastObjects) != 1 {
		t.Fatalf("expected 1 object in callback, got %d", len(lastObjects))
	}

	s.SetRunState("watcher", RunStateRunning)
	if callCount != 2 {
		t.Fatalf("expected onChange called twice, got %d", callCount)
	}
}

func TestLoadState(t *testing.T) {
	s := NewStore()
	s.LoadState([]ClusterObject{
		{
			ID:              "abc",
			Name:            "restored",
			Definition:      "(defagent \"restored\")",
			Revisions:       []Revision{{ID: "abc", Definition: "(defagent \"restored\")"}},
			State:           RunStateStopped,
			CurrentRevision: "abc",
		},
	})

	agent := s.GetAgent("restored")
	if agent == nil {
		t.Fatal("expected restored agent")
	}
	if agent.State != RunStateStopped {
		t.Fatalf("expected stopped state, got %s", agent.State)
	}
}
