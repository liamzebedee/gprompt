package cluster

import (
	"bufio"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

// dial connects to the server and returns the connection plus a scanner
// for reading newline-delimited JSON responses.
func dial(t *testing.T, addr string) (net.Conn, *bufio.Scanner) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	return conn, scanner
}

// sendEnvelope marshals and writes an envelope followed by newline.
func sendEnvelope(t *testing.T, conn net.Conn, msgType MessageType, payload interface{}) {
	t.Helper()
	env, err := NewEnvelope(msgType, payload)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// readEnvelope reads one line from the scanner and decodes it as an Envelope.
func readEnvelope(t *testing.T, scanner *bufio.Scanner) *Envelope {
	t.Helper()
	if !scanner.Scan() {
		t.Fatalf("expected response, got EOF or error: %v", scanner.Err())
	}
	var env Envelope
	if err := json.Unmarshal(scanner.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal response: %v (raw: %s)", err, scanner.Text())
	}
	return &env
}

// startTestServer creates a store and server on a random port, starts it
// in a goroutine, and returns cleanup function.
func startTestServer(t *testing.T) (*Server, *Store, func()) {
	t.Helper()
	store := NewStore()
	srv := NewServer(store, "127.0.0.1:0")

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Wait for listener to be ready
	deadline := time.Now().Add(2 * time.Second)
	for srv.listener == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if srv.listener == nil {
		t.Fatal("server did not start in time")
	}

	cleanup := func() {
		srv.Stop()
	}

	return srv, store, cleanup
}

// TestServerApplyRequest verifies the apply request/response flow:
// sending agent definitions and receiving summary.
func TestServerApplyRequest(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	conn, scanner := dial(t, srv.Addr())
	defer conn.Close()

	// Send apply request with two agents
	req := ApplyRequest{
		Agents: []AgentDef{
			{Name: "builder", ID: "abc123", Definition: "(defagent \"builder\" (loop build))"},
			{Name: "tester", ID: "def456", Definition: "(defagent \"tester\" (loop test))"},
		},
	}
	sendEnvelope(t, conn, MsgApplyRequest, req)

	// Read response
	env := readEnvelope(t, scanner)
	if env.Type != MsgApplyResponse {
		t.Fatalf("expected apply_response, got %s", env.Type)
	}

	var resp ApplyResponse
	if err := env.DecodePayload(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	if len(resp.Summary.Created) != 2 {
		t.Fatalf("expected 2 created, got %d: %v", len(resp.Summary.Created), resp.Summary.Created)
	}
}

// TestServerApplyIdempotent verifies that applying the same definitions
// twice results in "unchanged" on the second apply.
func TestServerApplyIdempotent(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	agents := []AgentDef{
		{Name: "builder", ID: "abc123", Definition: "(defagent \"builder\" (loop build))"},
	}

	// First apply
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{Agents: agents})
	env1 := readEnvelope(t, scanner1)
	var resp1 ApplyResponse
	env1.DecodePayload(&resp1)
	conn1.Close()

	if len(resp1.Summary.Created) != 1 {
		t.Fatalf("first apply: expected 1 created, got %v", resp1.Summary)
	}

	// Second apply with same ID — should be unchanged
	conn2, scanner2 := dial(t, srv.Addr())
	sendEnvelope(t, conn2, MsgApplyRequest, ApplyRequest{Agents: agents})
	env2 := readEnvelope(t, scanner2)
	var resp2 ApplyResponse
	env2.DecodePayload(&resp2)
	conn2.Close()

	if len(resp2.Summary.Unchanged) != 1 {
		t.Fatalf("second apply: expected 1 unchanged, got %v", resp2.Summary)
	}
	if len(resp2.Summary.Created) != 0 {
		t.Fatalf("second apply: expected 0 created, got %v", resp2.Summary.Created)
	}
}

// TestServerApplyUpdate verifies that changing an agent's definition
// creates a new revision.
func TestServerApplyUpdate(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	// First apply
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{Name: "builder", ID: "v1", Definition: "(defagent \"builder\" v1)"},
		},
	})
	env1 := readEnvelope(t, scanner1)
	var resp1 ApplyResponse
	env1.DecodePayload(&resp1)
	conn1.Close()

	if len(resp1.Summary.Created) != 1 {
		t.Fatalf("expected 1 created")
	}

	// Second apply with different ID — should be updated
	conn2, scanner2 := dial(t, srv.Addr())
	sendEnvelope(t, conn2, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{Name: "builder", ID: "v2", Definition: "(defagent \"builder\" v2)"},
		},
	})
	env2 := readEnvelope(t, scanner2)
	var resp2 ApplyResponse
	env2.DecodePayload(&resp2)
	conn2.Close()

	if len(resp2.Summary.Updated) != 1 {
		t.Fatalf("expected 1 updated, got %v", resp2.Summary)
	}
}

// TestServerSteerSubscribe verifies that steer clients receive state
// immediately on subscription and on subsequent changes.
func TestServerSteerSubscribe(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	// Pre-apply an agent
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{Name: "builder", ID: "abc", Definition: "(defagent \"builder\" body)"},
		},
	})
	readEnvelope(t, scanner1) // consume response
	conn1.Close()

	// Subscribe as steer client
	steerConn, steerScanner := dial(t, srv.Addr())
	defer steerConn.Close()

	sendEnvelope(t, steerConn, MsgSteerSubscribe, SteerSubscribeRequest{})

	// Should receive initial state with 1 agent
	env := readEnvelope(t, steerScanner)
	if env.Type != MsgSteerState {
		t.Fatalf("expected steer_state, got %s", env.Type)
	}

	var state SteerStatePayload
	if err := env.DecodePayload(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(state.Objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(state.Objects))
	}
	if state.Objects[0].Name != "builder" {
		t.Fatalf("expected agent name 'builder', got %q", state.Objects[0].Name)
	}

	// Now apply another agent — steer client should receive push
	conn2, scanner2 := dial(t, srv.Addr())
	sendEnvelope(t, conn2, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{Name: "tester", ID: "def", Definition: "(defagent \"tester\" body)"},
		},
	})
	readEnvelope(t, scanner2)
	conn2.Close()

	// Steer client should receive updated state with 2 agents
	env2 := readEnvelope(t, steerScanner)
	if env2.Type != MsgSteerState {
		t.Fatalf("expected steer_state push, got %s", env2.Type)
	}

	var state2 SteerStatePayload
	env2.DecodePayload(&state2)
	if len(state2.Objects) != 2 {
		t.Fatalf("expected 2 objects in push, got %d", len(state2.Objects))
	}
}

// TestServerStopSendsShutdown verifies that stopping the server sends
// shutdown notices to connected steer clients.
func TestServerStopSendsShutdown(t *testing.T) {
	srv, _, cleanup := startTestServer(t)

	// Subscribe as steer client
	steerConn, steerScanner := dial(t, srv.Addr())
	defer steerConn.Close()

	sendEnvelope(t, steerConn, MsgSteerSubscribe, SteerSubscribeRequest{})
	readEnvelope(t, steerScanner) // consume initial state

	// Stop the server
	cleanup()

	// Steer client should receive shutdown notice (or EOF)
	if steerScanner.Scan() {
		var env Envelope
		if err := json.Unmarshal(steerScanner.Bytes(), &env); err == nil {
			if env.Type != MsgShutdownNotice {
				t.Fatalf("expected shutdown_notice, got %s", env.Type)
			}
			var payload ShutdownNoticePayload
			env.DecodePayload(&payload)
			if !strings.Contains(payload.Reason, "shutting down") {
				t.Fatalf("expected shutdown reason, got %q", payload.Reason)
			}
		}
	}
}

// TestServerPortInUse verifies clear error when the port is already occupied.
func TestServerPortInUse(t *testing.T) {
	// Occupy a port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	occupiedAddr := ln.Addr().String()

	// Try to start server on same port
	store := NewStore()
	srv := NewServer(store, occupiedAddr)
	err = srv.ListenAndServe()
	if err == nil {
		srv.Stop()
		t.Fatal("expected error for port in use")
	}
	if !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("expected 'listen on' in error, got: %v", err)
	}
}

// TestServerMultipleSteerClients verifies that multiple steer clients
// all receive state push updates.
func TestServerMultipleSteerClients(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	// Subscribe two steer clients
	steer1, scan1 := dial(t, srv.Addr())
	defer steer1.Close()
	sendEnvelope(t, steer1, MsgSteerSubscribe, SteerSubscribeRequest{})
	readEnvelope(t, scan1) // initial state

	steer2, scan2 := dial(t, srv.Addr())
	defer steer2.Close()
	sendEnvelope(t, steer2, MsgSteerSubscribe, SteerSubscribeRequest{})
	readEnvelope(t, scan2) // initial state

	// Apply an agent — both should get push
	applyConn, applyScan := dial(t, srv.Addr())
	sendEnvelope(t, applyConn, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{Name: "agent1", ID: "id1", Definition: "def1"},
		},
	})
	readEnvelope(t, applyScan)
	applyConn.Close()

	// Both steer clients should receive state update
	env1 := readEnvelope(t, scan1)
	env2 := readEnvelope(t, scan2)

	if env1.Type != MsgSteerState || env2.Type != MsgSteerState {
		t.Fatalf("expected steer_state for both, got %s and %s", env1.Type, env2.Type)
	}
}
