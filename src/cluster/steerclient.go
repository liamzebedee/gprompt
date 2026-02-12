// Package cluster steerclient implements the network client for gcluster steer.
//
// The steer client connects to the master TCP server, subscribes for state
// updates, and provides methods to send steering messages (inject). It runs
// a background goroutine that reads incoming state pushes and delivers them
// through a channel for the TUI to consume.
//
// Design: The client is intentionally simple — it handles the network protocol
// and delivers parsed payloads. The TUI is responsible for rendering. This
// separation allows the client to be tested independently of bubbletea.
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

// SteerClient connects to a gcluster master and receives state updates.
// It is used by the steer TUI to observe and interact with agents.
//
// The client supports auto-reconnect: when the connection drops, it attempts
// to reconnect with exponential backoff (1s, 2s, 4s, capped at 10s). The
// reconnect loop runs until Close() is called or a connection succeeds.
// During reconnect, ErrCh receives periodic status updates so the TUI can
// show a disconnection banner.
type SteerClient struct {
	conn    net.Conn
	addr    string
	scanner *bufio.Scanner

	// StateCh delivers state payloads from the master. The TUI reads
	// from this channel to update its view. Buffered to avoid blocking
	// the read goroutine if the TUI is slow to consume.
	StateCh chan SteerStatePayload

	// ErrCh delivers connection errors (disconnects, protocol errors).
	// The TUI reads from this to show error banners.
	ErrCh chan error

	// ReconnectCh signals when a reconnection succeeds. The TUI uses
	// this to clear the error banner and re-subscribe for state updates.
	ReconnectCh chan struct{}

	mu     sync.Mutex
	closed bool
	done   chan struct{}
}

// NewSteerClient creates a client that connects to the master at the given address.
// It subscribes for state updates and starts reading in the background.
func NewSteerClient(addr string) (*SteerClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to master at %s — is `gcluster master` running?\n%w", addr, err)
	}

	sc := &SteerClient{
		conn:        conn,
		addr:        addr,
		scanner:     bufio.NewScanner(conn),
		StateCh:     make(chan SteerStatePayload, 16),
		ErrCh:       make(chan error, 4),
		ReconnectCh: make(chan struct{}, 1),
		done:        make(chan struct{}),
	}
	sc.scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)

	// Send subscribe message
	env, err := NewEnvelope(MsgSteerSubscribe, SteerSubscribeRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("marshal subscribe: %w", err)
	}
	data, err := json.Marshal(env)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("marshal subscribe: %w", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send subscribe: %w", err)
	}

	// Start background reader
	go sc.readLoop()

	return sc, nil
}

// readLoop reads messages from the master and dispatches them to channels.
// On disconnect, it attempts to reconnect with exponential backoff. The
// reconnect loop continues until Close() is called or reconnection succeeds.
func (sc *SteerClient) readLoop() {
	defer close(sc.done)

	for {
		disconnectErr := sc.readMessages()

		// Check if we were closed intentionally
		sc.mu.Lock()
		if sc.closed {
			sc.mu.Unlock()
			return
		}
		sc.mu.Unlock()

		// Report the disconnect
		if disconnectErr != nil {
			sc.sendErr(disconnectErr)
		}

		// Attempt reconnect with exponential backoff
		if !sc.reconnect() {
			return // closed during reconnect
		}

		// Signal successful reconnection
		select {
		case sc.ReconnectCh <- struct{}{}:
		default:
		}
	}
}

// readMessages reads from the current connection until disconnect or error.
// Returns the disconnect error, or nil for graceful shutdown.
func (sc *SteerClient) readMessages() error {
	for sc.scanner.Scan() {
		line := sc.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var env Envelope
		if err := json.Unmarshal(line, &env); err != nil {
			log.Printf("steer client: malformed message: %v", err)
			continue
		}

		switch env.Type {
		case MsgSteerState:
			var payload SteerStatePayload
			if err := env.DecodePayload(&payload); err != nil {
				log.Printf("steer client: decode state: %v", err)
				continue
			}
			// Non-blocking send — drop old state if TUI is behind.
			select {
			case sc.StateCh <- payload:
			default:
				// Drain and replace with latest state
				select {
				case <-sc.StateCh:
				default:
				}
				sc.StateCh <- payload
			}

		case MsgShutdownNotice:
			var payload ShutdownNoticePayload
			env.DecodePayload(&payload)
			return fmt.Errorf("master shutting down: %s", payload.Reason)

		default:
			log.Printf("steer client: unexpected message type: %s", env.Type)
		}
	}

	// Scanner done — either EOF or error
	if err := sc.scanner.Err(); err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	return fmt.Errorf("master disconnected")
}

// reconnect attempts to re-establish the connection with exponential backoff.
// Returns true if reconnected, false if the client was closed during reconnect.
func (sc *SteerClient) reconnect() bool {
	backoff := 1 * time.Second
	maxBackoff := 10 * time.Second

	for attempt := 1; ; attempt++ {
		sc.mu.Lock()
		if sc.closed {
			sc.mu.Unlock()
			return false
		}
		sc.mu.Unlock()

		log.Printf("steer client: reconnecting to %s (attempt %d, backoff %v)", sc.addr, attempt, backoff)
		sc.sendErr(fmt.Errorf("disconnected — reconnecting (attempt %d)...", attempt))

		time.Sleep(backoff)

		sc.mu.Lock()
		if sc.closed {
			sc.mu.Unlock()
			return false
		}
		sc.mu.Unlock()

		conn, err := net.Dial("tcp", sc.addr)
		if err != nil {
			log.Printf("steer client: reconnect attempt %d failed: %v", attempt, err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Re-subscribe
		env, err := NewEnvelope(MsgSteerSubscribe, SteerSubscribeRequest{})
		if err != nil {
			conn.Close()
			continue
		}
		data, err := json.Marshal(env)
		if err != nil {
			conn.Close()
			continue
		}
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			conn.Close()
			continue
		}

		// Success — swap connection
		sc.mu.Lock()
		sc.conn = conn
		sc.scanner = bufio.NewScanner(conn)
		sc.scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)
		sc.mu.Unlock()

		log.Printf("steer client: reconnected to %s", sc.addr)
		return true
	}
}

// sendErr sends an error to ErrCh without blocking.
func (sc *SteerClient) sendErr(err error) {
	select {
	case sc.ErrCh <- err:
	default:
		// Drain and replace
		select {
		case <-sc.ErrCh:
		default:
		}
		sc.ErrCh <- err
	}
}

// Inject sends a steering message to inject into an agent's conversation.
func (sc *SteerClient) Inject(agentName, stepLabel string, iteration int, message string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return fmt.Errorf("client closed")
	}

	req := SteerInjectRequest{
		AgentName: agentName,
		StepLabel: stepLabel,
		Iteration: iteration,
		Message:   message,
	}
	env, err := NewEnvelope(MsgSteerInject, req)
	if err != nil {
		return fmt.Errorf("marshal inject: %w", err)
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal inject: %w", err)
	}
	data = append(data, '\n')
	if _, err := sc.conn.Write(data); err != nil {
		return fmt.Errorf("send inject: %w", err)
	}
	return nil
}

// Close disconnects from the master and stops the reconnect loop.
// It is safe to call multiple times.
func (sc *SteerClient) Close() error {
	sc.mu.Lock()
	if sc.closed {
		sc.mu.Unlock()
		return nil
	}
	sc.closed = true
	err := sc.conn.Close()
	sc.mu.Unlock()

	// Wait for readLoop to exit. The lock must be released first because
	// readLoop checks sc.closed under the lock during reconnect attempts.
	<-sc.done
	return err
}
