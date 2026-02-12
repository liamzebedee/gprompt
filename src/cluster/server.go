// Package cluster server implements the TCP control plane for gcluster master.
//
// The server listens on DefaultAddr (127.0.0.1:43252), accepting connections
// from both `gcluster apply` and `gcluster steer` clients. It dispatches
// messages by type: apply_request goes to the store, steer_subscribe adds
// the connection to the push set, and steer_inject forwards steering messages
// to the agent executor.
//
// Design: newline-delimited JSON over TCP. Each message is an Envelope with
// a type field and a payload. The server reads one message at a time per
// connection, allowing both request-response (apply) and streaming (steer)
// patterns on the same protocol.
package cluster

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Server is the gcluster master TCP server. It owns the Store, manages
// client connections, pushes state updates to subscribed steer clients,
// and drives agent execution via the Executor.
type Server struct {
	store    *Store
	executor *Executor
	listener net.Listener
	addr     string

	// steer clients: connections that receive state push updates
	mu           sync.Mutex
	steerClients map[net.Conn]bool

	// agentMethods caches the resolved method bodies for each agent,
	// keyed by agent name → (method name → method body). Populated
	// by apply requests so the executor can start agents and so steer
	// clients can display human-readable method text.
	agentMethods map[string]map[string]string

	// agentPipelines caches the pipeline definitions for each agent,
	// keyed by agent name. Populated by apply requests so steer clients
	// can render pipeline-aware tree views.
	agentPipelines map[string]*PipelineDef

	// done is closed when the server stops
	done chan struct{}
}

// NewServer creates a server bound to the given store.
// If claudeFn is non-nil, an executor is created to manage agent goroutines.
// If claudeFn is nil, agents are stored but not executed (useful for tests
// that don't need execution).
// Call ListenAndServe to start accepting connections.
func NewServer(store *Store, addr string, claudeFn ...ClaudeFunc) *Server {
	if addr == "" {
		addr = DefaultAddr
	}
	s := &Server{
		store:          store,
		addr:           addr,
		steerClients:   make(map[net.Conn]bool),
		agentMethods:   make(map[string]map[string]string),
		agentPipelines: make(map[string]*PipelineDef),
		done:           make(chan struct{}),
	}

	// Create executor if a claude function was provided.
	if len(claudeFn) > 0 && claudeFn[0] != nil {
		s.executor = NewExecutor(store, claudeFn[0])
		// Push state to steer clients after each iteration completes,
		// so they see new iteration data in real time.
		s.executor.OnIteration(func(agentName string) {
			objects := store.ListAgents()
			s.pushState(objects)
		})
	}

	// Wire up state change notifications to push to steer clients.
	store.OnChange(func(objects []ClusterObject) {
		s.pushState(objects)
	})

	return s
}

// ListenAndServe starts the TCP listener and accepts connections.
// It blocks until Stop is called or an unrecoverable error occurs.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}
	s.listener = ln
	log.Printf("gcluster master listening on %s", s.addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil // clean shutdown
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// Executor returns the server's executor, or nil if none was configured.
func (s *Server) Executor() *Executor {
	return s.executor
}

// Addr returns the listener's address, useful in tests where port 0 is used.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// Stop gracefully shuts down the server: stops running agents, closes
// the listener, sends shutdown notices to steer clients, and cleans up.
func (s *Server) Stop() {
	select {
	case <-s.done:
		return // already stopped
	default:
	}
	close(s.done)

	// Stop all running agents before closing connections.
	if s.executor != nil {
		s.executor.StopAll(10 * time.Second)
	}

	if s.listener != nil {
		s.listener.Close()
	}

	// Notify and close steer clients
	s.mu.Lock()
	clients := make([]net.Conn, 0, len(s.steerClients))
	for conn := range s.steerClients {
		clients = append(clients, conn)
	}
	s.steerClients = make(map[net.Conn]bool)
	s.mu.Unlock()

	for _, conn := range clients {
		// Best-effort shutdown notice
		env, err := NewEnvelope(MsgShutdownNotice, ShutdownNoticePayload{Reason: "master shutting down"})
		if err == nil {
			data, _ := json.Marshal(env)
			data = append(data, '\n')
			conn.Write(data)
		}
		conn.Close()
	}
}

// handleConn reads messages from a connection and dispatches by type.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var env Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			log.Printf("malformed message from %s: %v", conn.RemoteAddr(), err)
			continue
		}

		switch env.Type {
		case MsgApplyRequest:
			s.handleApply(conn, &env)
		case MsgSteerSubscribe:
			s.handleSteerSubscribe(conn)
			return // steer connections stay open until disconnect
		case MsgSteerInject:
			s.handleSteerInject(&env)
		default:
			log.Printf("unknown message type %q from %s", env.Type, conn.RemoteAddr())
		}
	}
}

// handleApply processes an apply_request: deserializes agent definitions,
// applies them to the store, starts any pending agents, and sends back
// the summary.
func (s *Server) handleApply(conn net.Conn, env *Envelope) {
	var req ApplyRequest
	if err := env.DecodePayload(&req); err != nil {
		s.sendResponse(conn, MsgApplyResponse, ApplyResponse{Error: fmt.Sprintf("decode error: %v", err)})
		return
	}

	// Cache method bodies and pipeline definitions from the apply request
	// for executor use when starting agents, and for steer clients to
	// display human-readable method text and pipeline structure.
	s.mu.Lock()
	for _, def := range req.Agents {
		if len(def.Methods) > 0 {
			s.agentMethods[def.Name] = def.Methods
		}
		if def.Pipeline != nil {
			s.agentPipelines[def.Name] = def.Pipeline
		}
	}
	s.mu.Unlock()

	// Pass pipeline definitions to executor so it knows step structure.
	if s.executor != nil {
		for _, def := range req.Agents {
			if def.Pipeline != nil {
				s.executor.SetPipeline(def.Name, def.Pipeline)
			}
		}
	}

	summary := s.store.ApplyDefinitions(req.Agents)
	s.sendResponse(conn, MsgApplyResponse, ApplyResponse{Summary: summary})

	// Start any newly-created (pending) agents if we have an executor.
	if s.executor != nil {
		s.mu.Lock()
		methods := make(map[string]map[string]string, len(s.agentMethods))
		for k, v := range s.agentMethods {
			methods[k] = v
		}
		s.mu.Unlock()
		s.executor.StartPending(methods)
	}
}

// handleSteerSubscribe registers a connection for state push updates.
// It immediately sends the current state, then keeps the connection open
// for future pushes. The connection stays open until the client disconnects.
func (s *Server) handleSteerSubscribe(conn net.Conn) {
	s.mu.Lock()
	s.steerClients[conn] = true
	s.mu.Unlock()

	// Send current state immediately (including run data if executor exists)
	objects := s.store.ListAgents()
	payload := SteerStatePayload{Objects: objects}
	if s.executor != nil {
		payload.Runs = s.executor.Snapshot()
	}
	// Include cached methods and pipelines so TUI can display them.
	s.mu.Lock()
	if len(s.agentMethods) > 0 {
		payload.Methods = make(map[string]map[string]string, len(s.agentMethods))
		for k, v := range s.agentMethods {
			payload.Methods[k] = v
		}
	}
	if len(s.agentPipelines) > 0 {
		payload.Pipelines = make(map[string]*PipelineDef, len(s.agentPipelines))
		for k, v := range s.agentPipelines {
			payload.Pipelines[k] = v
		}
	}
	s.mu.Unlock()
	s.sendResponse(conn, MsgSteerState, payload)

	// Keep connection alive — read until EOF or error
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var env Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}
		if env.Type == MsgSteerInject {
			s.handleSteerInject(&env)
		} else if env.Type == MsgSteerEditPrompt {
			s.handleSteerEditPrompt(&env)
		}
	}

	// Client disconnected — remove from push set
	s.mu.Lock()
	delete(s.steerClients, conn)
	s.mu.Unlock()
}

// handleSteerInject processes a steering message injection by forwarding it
// to the executor. The executor queues the message on the agent's inject
// channel; the agent goroutine drains injected messages before each iteration
// and prepends them to the prompt, allowing humans to steer the agent.
func (s *Server) handleSteerInject(env *Envelope) {
	var req SteerInjectRequest
	if err := env.DecodePayload(&req); err != nil {
		log.Printf("steer_inject decode error: %v", err)
		return
	}
	log.Printf("steer inject: agent=%s step=%s iter=%d msg=%q", req.AgentName, req.StepLabel, req.Iteration, req.Message)

	if s.executor == nil {
		log.Printf("steer inject: no executor configured, message dropped")
		return
	}

	if err := s.executor.InjectMessage(req.AgentName, req.Message); err != nil {
		log.Printf("steer inject: forward to executor failed: %v", err)
	}
}

// handleSteerEditPrompt updates a cached method body and notifies the executor.
// The next steer_state push will include the updated Methods map, and the
// executor will use the new body from the next loop iteration onward.
// This allows steer clients to permanently modify an agent's prompt at runtime.
func (s *Server) handleSteerEditPrompt(env *Envelope) {
	var req SteerEditPromptRequest
	if err := env.DecodePayload(&req); err != nil {
		log.Printf("steer_edit_prompt decode error: %v", err)
		return
	}
	log.Printf("steer edit_prompt: agent=%s method=%s (%d bytes)", req.AgentName, req.MethodName, len(req.NewBody))

	// Update the server's cached method body — this is the source of truth
	// for method bodies. When agents restart, StartPending uses this cache.
	s.mu.Lock()
	methods, ok := s.agentMethods[req.AgentName]
	if !ok {
		methods = make(map[string]string)
		s.agentMethods[req.AgentName] = methods
	}
	methods[req.MethodName] = req.NewBody
	s.mu.Unlock()

	// Tell the executor to use the new body for subsequent iterations.
	if s.executor != nil {
		s.executor.UpdateMethodBody(req.AgentName, req.MethodName, req.NewBody)
	}

	// Push updated state so all steer clients see the change reflected.
	objects := s.store.ListAgents()
	s.pushState(objects)
}

// pushState sends the current cluster state to all subscribed steer clients.
// Called by the store's OnChange callback after every mutation, and by the
// executor's OnIteration callback after each iteration completes.
func (s *Server) pushState(objects []ClusterObject) {
	payload := SteerStatePayload{Objects: objects}
	if s.executor != nil {
		payload.Runs = s.executor.Snapshot()
	}

	// Grab cached methods and pipelines under s.mu so TUI can display them.
	s.mu.Lock()
	if len(s.agentMethods) > 0 {
		payload.Methods = make(map[string]map[string]string, len(s.agentMethods))
		for k, v := range s.agentMethods {
			payload.Methods[k] = v
		}
	}
	if len(s.agentPipelines) > 0 {
		payload.Pipelines = make(map[string]*PipelineDef, len(s.agentPipelines))
		for k, v := range s.agentPipelines {
			payload.Pipelines[k] = v
		}
	}
	s.mu.Unlock()

	env, err := NewEnvelope(MsgSteerState, payload)
	if err != nil {
		log.Printf("pushState marshal error: %v", err)
		return
	}
	data, err := json.Marshal(env)
	if err != nil {
		log.Printf("pushState marshal error: %v", err)
		return
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	for conn := range s.steerClients {
		if _, err := conn.Write(data); err != nil {
			log.Printf("pushState write error to %s: %v", conn.RemoteAddr(), err)
			conn.Close()
			delete(s.steerClients, conn)
		}
	}
}

// sendResponse marshals and sends a single envelope to a connection.
func (s *Server) sendResponse(conn net.Conn, msgType MessageType, payload interface{}) {
	env, err := NewEnvelope(msgType, payload)
	if err != nil {
		log.Printf("sendResponse marshal error: %v", err)
		return
	}
	data, err := json.Marshal(env)
	if err != nil {
		log.Printf("sendResponse marshal error: %v", err)
		return
	}
	data = append(data, '\n')
	conn.Write(data)
}
