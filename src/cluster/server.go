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
)

// Server is the gcluster master TCP server. It owns the Store, manages
// client connections, and pushes state updates to subscribed steer clients.
type Server struct {
	store    *Store
	listener net.Listener
	addr     string

	// steer clients: connections that receive state push updates
	mu           sync.Mutex
	steerClients map[net.Conn]bool

	// done is closed when the server stops
	done chan struct{}
}

// NewServer creates a server bound to the given store.
// Call ListenAndServe to start accepting connections.
func NewServer(store *Store, addr string) *Server {
	if addr == "" {
		addr = DefaultAddr
	}
	s := &Server{
		store:        store,
		addr:         addr,
		steerClients: make(map[net.Conn]bool),
		done:         make(chan struct{}),
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

// Addr returns the listener's address, useful in tests where port 0 is used.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// Stop gracefully shuts down the server: closes the listener, sends
// shutdown notices to steer clients, and cleans up connections.
func (s *Server) Stop() {
	select {
	case <-s.done:
		return // already stopped
	default:
	}
	close(s.done)

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
// applies them to the store, and sends back the summary.
func (s *Server) handleApply(conn net.Conn, env *Envelope) {
	var req ApplyRequest
	if err := env.DecodePayload(&req); err != nil {
		s.sendResponse(conn, MsgApplyResponse, ApplyResponse{Error: fmt.Sprintf("decode error: %v", err)})
		return
	}

	summary := s.store.ApplyDefinitions(req.Agents)
	s.sendResponse(conn, MsgApplyResponse, ApplyResponse{Summary: summary})
}

// handleSteerSubscribe registers a connection for state push updates.
// It immediately sends the current state, then keeps the connection open
// for future pushes. The connection stays open until the client disconnects.
func (s *Server) handleSteerSubscribe(conn net.Conn) {
	s.mu.Lock()
	s.steerClients[conn] = true
	s.mu.Unlock()

	// Send current state immediately
	objects := s.store.ListAgents()
	payload := SteerStatePayload{Objects: objects}
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
		}
	}

	// Client disconnected — remove from push set
	s.mu.Lock()
	delete(s.steerClients, conn)
	s.mu.Unlock()
}

// handleSteerInject processes a steering message injection.
// TODO: Forward to executor when Phase 2.3 is implemented.
func (s *Server) handleSteerInject(env *Envelope) {
	var req SteerInjectRequest
	if err := env.DecodePayload(&req); err != nil {
		log.Printf("steer_inject decode error: %v", err)
		return
	}
	log.Printf("steer inject: agent=%s step=%s iter=%d msg=%q", req.AgentName, req.StepLabel, req.Iteration, req.Message)
	// Executor integration happens in Phase 2.3
}

// pushState sends the current cluster state to all subscribed steer clients.
// Called by the store's OnChange callback after every mutation.
func (s *Server) pushState(objects []ClusterObject) {
	payload := SteerStatePayload{Objects: objects}
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
