package cluster

import (
	"testing"
	"time"
)

// TestSteerClientReceivesState verifies the steer client connects to
// the master, subscribes, and receives state updates both immediately
// and when changes occur.
func TestSteerClientReceivesState(t *testing.T) {
	srv, store, cleanup := startTestServer(t)
	defer cleanup()

	// Pre-apply an agent so initial state is non-empty.
	store.ApplyDefinitions([]AgentDef{
		{Name: "builder", ID: "abc", Definition: "(defagent \"builder\" (pipeline (step \"build\" (loop build))))"},
	})

	// Connect steer client
	client, err := NewSteerClient(srv.Addr())
	if err != nil {
		t.Fatalf("NewSteerClient: %v", err)
	}
	defer client.Close()

	// Should receive initial state within 2 seconds.
	select {
	case state := <-client.StateCh:
		if len(state.Objects) != 1 {
			t.Fatalf("expected 1 object in initial state, got %d", len(state.Objects))
		}
		if state.Objects[0].Name != "builder" {
			t.Fatalf("expected agent 'builder', got %q", state.Objects[0].Name)
		}
	case err := <-client.ErrCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initial state")
	}

	// Apply another agent — should trigger a push.
	store.ApplyDefinitions([]AgentDef{
		{Name: "tester", ID: "def", Definition: "(defagent \"tester\" (pipeline (step \"test\" (loop test))))"},
	})

	select {
	case state := <-client.StateCh:
		if len(state.Objects) != 2 {
			t.Fatalf("expected 2 objects after second apply, got %d", len(state.Objects))
		}
	case err := <-client.ErrCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for state push")
	}
}

// TestSteerClientShutdownNotice verifies the steer client receives an
// error when the master shuts down.
func TestSteerClientShutdownNotice(t *testing.T) {
	srv, _, cleanup := startTestServer(t)

	client, err := NewSteerClient(srv.Addr())
	if err != nil {
		t.Fatalf("NewSteerClient: %v", err)
	}
	defer client.Close()

	// Consume initial state
	select {
	case <-client.StateCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initial state")
	}

	// Shut down server
	cleanup()

	// Should receive error
	select {
	case err := <-client.ErrCh:
		if err == nil {
			t.Fatal("expected non-nil error on shutdown")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for shutdown error")
	}
}

// TestSteerClientInject verifies the inject method sends a message
// without error (the server logs it).
func TestSteerClientInject(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	client, err := NewSteerClient(srv.Addr())
	if err != nil {
		t.Fatalf("NewSteerClient: %v", err)
	}
	defer client.Close()

	// Consume initial state
	select {
	case <-client.StateCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for initial state")
	}

	// Inject should succeed (server just logs it)
	if err := client.Inject("builder", "build", 1, "focus on tests"); err != nil {
		t.Fatalf("Inject: %v", err)
	}
}

// TestSteerClientConnectionRefused verifies clear error when master
// is not running.
func TestSteerClientConnectionRefused(t *testing.T) {
	_, err := NewSteerClient("127.0.0.1:19999")
	if err == nil {
		t.Fatal("expected connection error")
	}
}
