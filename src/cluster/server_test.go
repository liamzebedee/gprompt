package cluster

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
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

// startTestServerWithExecutor creates a server with a real executor
// using a fake claude function, for testing inject forwarding.
func startTestServerWithExecutor(t *testing.T, claudeFn ClaudeFunc) (*Server, *Store, func()) {
	t.Helper()
	store := NewStore()
	srv := NewServer(store, "127.0.0.1:0", claudeFn)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

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

// TestServerInjectForwarding verifies that steer_inject messages are
// forwarded through the server to the executor, and the agent receives
// the injected message in its next iteration prompt.
func TestServerInjectForwarding(t *testing.T) {
	// Track prompts to verify injection delivery
	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string, onMessage func(ConvoMessage)) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(30 * time.Millisecond):
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	srv, _, cleanup := startTestServerWithExecutor(t, claudeFn)
	defer cleanup()

	// Apply an agent via the TCP protocol
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{
				Name:       "builder",
				ID:         "abc",
				Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`,
				Methods:    map[string]string{"build": "do some work"},
			},
		},
	})
	readEnvelope(t, scanner1) // consume response
	conn1.Close()

	// Wait for agent to start executing
	time.Sleep(50 * time.Millisecond)

	// Subscribe as steer client and send an inject
	steerConn, steerScanner := dial(t, srv.Addr())
	defer steerConn.Close()
	sendEnvelope(t, steerConn, MsgSteerSubscribe, SteerSubscribeRequest{})
	readEnvelope(t, steerScanner) // consume initial state

	// Send inject
	sendEnvelope(t, steerConn, MsgSteerInject, SteerInjectRequest{
		AgentName: "builder",
		StepLabel: "build",
		Iteration: 1,
		Message:   "prioritize security fixes",
	})

	// Wait for the agent to complete another iteration with the injected message
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	found := false
	for _, p := range prompts {
		if strings.Contains(p, "prioritize security fixes") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected injected message to appear in agent prompt; got %d prompts", len(prompts))
	}
}

// TestServerSteerStateIncludesMethodsAndPipelines verifies that steer state
// pushes include cached method bodies and pipeline definitions from apply
// requests, so the TUI can display human-readable method text and
// pipeline-aware tree structure.
func TestServerSteerStateIncludesMethodsAndPipelines(t *testing.T) {
	srv, _, cleanup := startTestServer(t)
	defer cleanup()

	// Apply an agent with methods and pipeline
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{
				Name:       "planner",
				ID:         "abc123",
				Definition: `(defagent "planner" ...)`,
				Methods:    map[string]string{"build": "Read BACKLOG.md, pick item, build it"},
				Pipeline: &PipelineDef{
					Steps: []PipelineStep{
						{Label: "build", Kind: StepKindLoop, LoopMethod: "build"},
					},
				},
			},
		},
	})
	readEnvelope(t, scanner1) // consume apply response
	conn1.Close()

	// Subscribe as steer client
	steerConn, steerScanner := dial(t, srv.Addr())
	defer steerConn.Close()
	sendEnvelope(t, steerConn, MsgSteerSubscribe, SteerSubscribeRequest{})

	env := readEnvelope(t, steerScanner)
	if env.Type != MsgSteerState {
		t.Fatalf("expected steer_state, got %s", env.Type)
	}

	var state SteerStatePayload
	if err := env.DecodePayload(&state); err != nil {
		t.Fatalf("decode state: %v", err)
	}

	// Verify methods are included
	if state.Methods == nil {
		t.Fatal("expected Methods in steer state, got nil")
	}
	if body, ok := state.Methods["planner"]["build"]; !ok || !strings.Contains(body, "BACKLOG") {
		t.Fatalf("expected method body with BACKLOG, got %q", body)
	}

	// Verify pipelines are included
	if state.Pipelines == nil {
		t.Fatal("expected Pipelines in steer state, got nil")
	}
	pdef, ok := state.Pipelines["planner"]
	if !ok {
		t.Fatal("expected pipeline for 'planner'")
	}
	if len(pdef.Steps) != 1 || pdef.Steps[0].LoopMethod != "build" {
		t.Fatalf("expected 1 step with loop method 'build', got %+v", pdef)
	}
}

// TestServerEditPromptUpdatesMethodCache verifies that steer_edit_prompt
// messages update the server's cached method bodies, and the change is
// reflected in subsequent steer_state pushes to all connected clients.
func TestServerEditPromptUpdatesMethodCache(t *testing.T) {
	// Use a slow claude function so the agent stays alive during the test.
	claudeFn := func(ctx context.Context, prompt string, onMessage func(ConvoMessage)) (string, error) {
		select {
		case <-time.After(5 * time.Second):
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	srv, _, cleanup := startTestServerWithExecutor(t, claudeFn)
	defer cleanup()

	// Apply an agent with initial method body
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{
				Name:       "builder",
				ID:         "abc",
				Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`,
				Methods:    map[string]string{"build": "original prompt"},
				Pipeline: &PipelineDef{
					Steps: []PipelineStep{
						{Label: "build", Kind: StepKindLoop, LoopMethod: "build"},
					},
				},
			},
		},
	})
	readEnvelope(t, scanner1)
	conn1.Close()

	// Wait for agent to start
	time.Sleep(50 * time.Millisecond)

	// Subscribe as steer client
	steerConn, steerScanner := dial(t, srv.Addr())
	defer steerConn.Close()
	sendEnvelope(t, steerConn, MsgSteerSubscribe, SteerSubscribeRequest{})
	readEnvelope(t, steerScanner) // initial state

	// Send edit prompt
	sendEnvelope(t, steerConn, MsgSteerEditPrompt, SteerEditPromptRequest{
		AgentName:  "builder",
		MethodName: "build",
		NewBody:    "updated prompt: focus on security",
	})

	// Should receive updated state push with new method body
	env := readEnvelope(t, steerScanner)
	if env.Type != MsgSteerState {
		t.Fatalf("expected steer_state, got %s", env.Type)
	}

	var updated SteerStatePayload
	env.DecodePayload(&updated)
	if body := updated.Methods["builder"]["build"]; body != "updated prompt: focus on security" {
		t.Fatalf("expected updated method body, got %q", body)
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

// TestConcurrentSteerSessionConsistency verifies that two steer clients
// connected simultaneously see consistent state, including when both
// inject messages into the same agent concurrently. Per spec: "Two steer
// terminals connected to the same master show consistent state."
func TestConcurrentSteerSessionConsistency(t *testing.T) {
	// Track all prompts received by the agent.
	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string, onMessage func(ConvoMessage)) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(30 * time.Millisecond):
			return "output", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	srv, _, cleanup := startTestServerWithExecutor(t, claudeFn)
	defer cleanup()

	// Apply an agent
	conn1, scanner1 := dial(t, srv.Addr())
	sendEnvelope(t, conn1, MsgApplyRequest, ApplyRequest{
		Agents: []AgentDef{
			{
				Name:       "shared",
				ID:         "abc",
				Definition: `(defagent "shared" (loop work))`,
				Methods:    map[string]string{"work": "base prompt"},
			},
		},
	})
	readEnvelope(t, scanner1)
	conn1.Close()

	// Wait for agent to start
	time.Sleep(50 * time.Millisecond)

	// Subscribe two steer clients
	steer1, scan1 := dial(t, srv.Addr())
	defer steer1.Close()
	sendEnvelope(t, steer1, MsgSteerSubscribe, SteerSubscribeRequest{})
	state1 := readEnvelope(t, scan1) // initial state

	steer2, scan2 := dial(t, srv.Addr())
	defer steer2.Close()
	sendEnvelope(t, steer2, MsgSteerSubscribe, SteerSubscribeRequest{})
	state2 := readEnvelope(t, scan2) // initial state

	// Both should see the same initial state (1 agent)
	var payload1, payload2 SteerStatePayload
	state1.DecodePayload(&payload1)
	state2.DecodePayload(&payload2)

	if len(payload1.Objects) != len(payload2.Objects) {
		t.Fatalf("inconsistent initial state: client1 sees %d objects, client2 sees %d",
			len(payload1.Objects), len(payload2.Objects))
	}

	// Both clients inject messages concurrently
	sendEnvelope(t, steer1, MsgSteerInject, SteerInjectRequest{
		AgentName: "shared", StepLabel: "work", Iteration: 1, Message: "msg-from-client-1",
	})
	sendEnvelope(t, steer2, MsgSteerInject, SteerInjectRequest{
		AgentName: "shared", StepLabel: "work", Iteration: 1, Message: "msg-from-client-2",
	})

	// Wait for the agent to process the messages and complete iterations
	time.Sleep(150 * time.Millisecond)

	// Both clients should receive state push updates — read at least one
	// push from each to verify they're getting updates.
	env1 := readEnvelope(t, scan1)
	env2 := readEnvelope(t, scan2)

	if env1.Type != MsgSteerState {
		t.Fatalf("client1: expected steer_state push, got %s", env1.Type)
	}
	if env2.Type != MsgSteerState {
		t.Fatalf("client2: expected steer_state push, got %s", env2.Type)
	}

	// Verify both injected messages were delivered to the agent
	mu.Lock()
	defer mu.Unlock()

	foundClient1, foundClient2 := false, false
	for _, p := range prompts {
		if strings.Contains(p, "msg-from-client-1") {
			foundClient1 = true
		}
		if strings.Contains(p, "msg-from-client-2") {
			foundClient2 = true
		}
	}
	if !foundClient1 {
		t.Error("expected client 1's inject message to be delivered to agent")
	}
	if !foundClient2 {
		t.Error("expected client 2's inject message to be delivered to agent")
	}
}
