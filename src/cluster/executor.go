// Executor manages agent goroutine lifecycles within the gcluster master.
//
// When the server applies agent definitions, it calls Executor.Start for each
// agent that is in pending state. The executor spawns a goroutine per agent
// that runs the agent's pipeline — typically a loop step that repeatedly calls
// the claude CLI with the method body as prompt.
//
// Design decisions:
//
//   - The executor does NOT parse S-expressions. The apply command resolves
//     method bodies at apply time and sends them alongside the definition in
//     AgentDef.Methods. This keeps the executor simple and decoupled from
//     the parser/compiler/sexp packages.
//
//   - Claude invocation is abstracted behind a ClaudeFunc. Production code
//     injects a function that shells out to the claude CLI; tests inject a
//     fake. This makes the executor fully testable without external processes.
//
//   - Each agent goroutine owns a cancellable context derived from the
//     executor's root context. Stopping an agent cancels its context and
//     waits for the goroutine to finish (bounded by a deadline).
//
//   - On claude failure mid-iteration, the error is recorded on the
//     IterationResult and the agent continues to the next iteration.
//     The agent only stops when explicitly stopped or the executor shuts down.
//
//   - Thread safety: the running map is guarded by a mutex. Store mutations
//     go through Store methods which have their own locks.
package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ClaudeFunc is the signature for invoking claude. It takes a context and a
// prompt string, and returns the output text or an error. Production code
// provides a function that calls the claude CLI; tests provide a fake.
type ClaudeFunc func(ctx context.Context, prompt string) (string, error)

// IterationResult records the outcome of a single loop iteration.
type IterationResult struct {
	// Iteration is the 1-based iteration number.
	Iteration int `json:"iteration"`
	// StartedAt is when this iteration began.
	StartedAt time.Time `json:"started_at"`
	// FinishedAt is when this iteration completed (success or failure).
	FinishedAt time.Time `json:"finished_at"`
	// Output is the claude response text (empty on error).
	Output string `json:"output,omitempty"`
	// Error is the error message if claude failed (empty on success).
	Error string `json:"error,omitempty"`
}

// AgentRun holds the runtime state for a single executing agent.
// It is created when an agent starts and removed when it stops.
type AgentRun struct {
	// Name is the agent name.
	Name string
	// RevisionID is the revision this run is executing.
	RevisionID string
	// StartedAt is when the agent goroutine began.
	StartedAt time.Time
	// Iterations records the outcome of each completed iteration.
	// Protected by mu.
	Iterations []IterationResult

	// cancel stops this agent's goroutine.
	cancel context.CancelFunc
	// done is closed when the agent goroutine exits.
	done chan struct{}

	mu sync.Mutex
}

// addIteration appends an iteration result to the run's history.
func (r *AgentRun) addIteration(ir IterationResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Iterations = append(r.Iterations, ir)
}

// CurrentIteration returns the number of completed iterations.
func (r *AgentRun) CurrentIteration() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Iterations)
}

// SnapshotIterations returns a copy of all iteration results.
func (r *AgentRun) SnapshotIterations() []IterationResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]IterationResult, len(r.Iterations))
	copy(cp, r.Iterations)
	return cp
}

// Executor manages the lifecycle of running agent goroutines.
// It is owned by the Server and operates on the shared Store.
type Executor struct {
	store     *Store
	claudeFn  ClaudeFunc
	rootCtx   context.Context
	rootStop  context.CancelFunc

	mu   sync.Mutex
	runs map[string]*AgentRun // keyed by agent name
}

// NewExecutor creates an executor bound to a store and a claude invocation
// function. The rootCtx should be derived from the server's shutdown context;
// cancelling it will stop all running agents.
func NewExecutor(store *Store, claudeFn ClaudeFunc) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		store:    store,
		claudeFn: claudeFn,
		rootCtx:  ctx,
		rootStop: cancel,
		runs:     make(map[string]*AgentRun),
	}
}

// Start launches execution of an agent. It transitions the agent from pending
// to running in the store and spawns the execution goroutine.
//
// If the agent is already running, Start is a no-op and returns nil.
// If the agent does not exist in the store, Start returns an error.
//
// The methods map provides the resolved method bodies needed by the agent's
// pipeline steps, keyed by method name. For a loop(build) agent, methods
// would contain {"build": "Read BACKLOG.md, pick one item, ..."}.
func (e *Executor) Start(name string, methods map[string]string) error {
	e.mu.Lock()
	if _, running := e.runs[name]; running {
		e.mu.Unlock()
		return nil // already running
	}

	obj := e.store.GetAgent(name)
	if obj == nil {
		e.mu.Unlock()
		return fmt.Errorf("agent %q not found in store", name)
	}

	// Transition to running in the store.
	if !e.store.SetRunState(name, RunStateRunning) {
		e.mu.Unlock()
		return fmt.Errorf("agent %q: failed to set running state", name)
	}

	agentCtx, agentCancel := context.WithCancel(e.rootCtx)
	run := &AgentRun{
		Name:       name,
		RevisionID: obj.CurrentRevision,
		StartedAt:  time.Now(),
		cancel:     agentCancel,
		done:       make(chan struct{}),
	}
	e.runs[name] = run
	e.mu.Unlock()

	// Determine prompt from the agent's pipeline definition.
	// The agent body is a pipeline. We need to resolve it.
	prompt, err := e.resolvePrompt(obj, methods)
	if err != nil {
		// Cannot start: revert to pending.
		e.store.SetRunState(name, RunStatePending)
		e.mu.Lock()
		delete(e.runs, name)
		e.mu.Unlock()
		agentCancel()
		close(run.done)
		return fmt.Errorf("agent %q: resolve prompt: %w", name, err)
	}

	go e.runAgent(agentCtx, run, prompt)
	revShort := run.RevisionID
	if len(revShort) > 8 {
		revShort = revShort[:8]
	}
	log.Printf("executor: started agent %q (revision %s)", name, revShort)
	return nil
}

// resolvePrompt extracts the prompt text for the agent's loop method.
//
// The agent's Definition is an S-expression like:
//   (defagent "builder" (pipeline (step "build" (loop build))))
//
// For now, we support the common pattern: a single loop step. The method
// body for that step comes from the methods map provided at Start time.
//
// Future expansion would handle multi-step pipelines, but agents in practice
// are loop(method) — a single method called repeatedly forever.
func (e *Executor) resolvePrompt(obj *ClusterObject, methods map[string]string) (string, error) {
	// We need to find the loop method name from the definition. But rather
	// than parsing the S-expression here, we rely on the methods map: the
	// apply command sends all method bodies referenced by the agent. For a
	// loop(build) agent, methods contains {"build": "Read BACKLOG.md..."}.
	//
	// If there's exactly one method, use it. If multiple, we need the
	// pipeline structure — which is also sent in AgentDef.Pipeline.
	if len(methods) == 0 {
		return "", fmt.Errorf("no method bodies provided")
	}
	if len(methods) == 1 {
		for _, body := range methods {
			return body, nil
		}
	}
	// Multiple methods: need pipeline info. For now, return error.
	// Multi-step pipeline execution is a future extension.
	return "", fmt.Errorf("multi-step pipelines not yet supported in executor (got %d methods)", len(methods))
}

// runAgent is the goroutine body for a running agent. It loops forever,
// calling claude with the prompt each iteration, until the context is
// cancelled (agent stopped or executor shut down).
func (e *Executor) runAgent(ctx context.Context, run *AgentRun, prompt string) {
	defer close(run.done)

	iteration := 0
	for {
		iteration++

		// Check for cancellation before starting iteration.
		select {
		case <-ctx.Done():
			log.Printf("executor: agent %q stopped before iteration %d", run.Name, iteration)
			return
		default:
		}

		ir := IterationResult{
			Iteration: iteration,
			StartedAt: time.Now(),
		}

		log.Printf("executor: agent %q starting iteration %d", run.Name, iteration)
		output, err := e.claudeFn(ctx, prompt)
		ir.FinishedAt = time.Now()

		if err != nil {
			// Check if the error is from context cancellation (agent stopped).
			if ctx.Err() != nil {
				log.Printf("executor: agent %q iteration %d cancelled", run.Name, iteration)
				ir.Error = "cancelled"
				run.addIteration(ir)
				return
			}
			// Claude failed mid-iteration: record error, continue to next.
			ir.Error = err.Error()
			run.addIteration(ir)
			log.Printf("executor: agent %q iteration %d failed: %v (continuing)", run.Name, iteration, err)
			continue
		}

		ir.Output = output
		run.addIteration(ir)
		log.Printf("executor: agent %q iteration %d complete (%d bytes output)", run.Name, iteration, len(output))
	}
}

// Stop halts a running agent. It cancels the agent's context and waits
// for the goroutine to exit (up to the given timeout). The agent's state
// is transitioned to stopped in the store.
//
// Returns an error if the agent is not running or the wait times out.
func (e *Executor) Stop(name string, timeout time.Duration) error {
	e.mu.Lock()
	run, ok := e.runs[name]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("agent %q is not running", name)
	}
	// Remove from runs immediately to prevent double-stop.
	delete(e.runs, name)
	e.mu.Unlock()

	// Signal the goroutine to stop.
	run.cancel()

	// Wait for goroutine to finish with timeout.
	select {
	case <-run.done:
		// Clean exit.
	case <-time.After(timeout):
		log.Printf("executor: agent %q did not stop within %v", name, timeout)
	}

	e.store.SetRunState(name, RunStateStopped)
	log.Printf("executor: stopped agent %q (%d iterations completed)", name, run.CurrentIteration())
	return nil
}

// StopAll stops all running agents and waits for them to finish.
// Used during graceful shutdown. Returns after all agents have stopped
// or the timeout expires.
func (e *Executor) StopAll(timeout time.Duration) {
	e.mu.Lock()
	names := make([]string, 0, len(e.runs))
	for name := range e.runs {
		names = append(names, name)
	}
	e.mu.Unlock()

	if len(names) == 0 {
		return
	}

	log.Printf("executor: stopping %d running agent(s)", len(names))

	// Cancel the root context — this signals all agent goroutines at once.
	e.rootStop()

	// Wait for all goroutines to finish.
	deadline := time.After(timeout)
	for _, name := range names {
		e.mu.Lock()
		run, ok := e.runs[name]
		e.mu.Unlock()
		if !ok {
			continue
		}

		select {
		case <-run.done:
			// Agent stopped cleanly.
		case <-deadline:
			log.Printf("executor: timeout waiting for agent %q during shutdown", name)
		}

		e.store.SetRunState(name, RunStateStopped)
	}

	// Clear runs map.
	e.mu.Lock()
	e.runs = make(map[string]*AgentRun)
	e.mu.Unlock()

	log.Printf("executor: all agents stopped")
}

// IsRunning reports whether the named agent has an active execution goroutine.
func (e *Executor) IsRunning(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.runs[name]
	return ok
}

// GetRun returns a snapshot of the named agent's run state, or nil if not running.
func (e *Executor) GetRun(name string) *AgentRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[name]
	if !ok {
		return nil
	}
	// Return the pointer directly — callers use the mutex-protected accessors
	// (SnapshotIterations, CurrentIteration) for thread-safe access.
	return run
}

// RunningAgents returns the names of all currently running agents.
func (e *Executor) RunningAgents() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	names := make([]string, 0, len(e.runs))
	for name := range e.runs {
		names = append(names, name)
	}
	return names
}

// StartPending scans the store for agents in pending state and starts them.
// This is called after applying definitions to auto-start new agents.
// The methods argument maps agent name -> (method name -> method body).
func (e *Executor) StartPending(agentMethods map[string]map[string]string) {
	agents := e.store.ListAgents()
	for _, obj := range agents {
		if obj.State != RunStatePending {
			continue
		}
		methods, ok := agentMethods[obj.Name]
		if !ok {
			log.Printf("executor: agent %q is pending but no methods provided, skipping", obj.Name)
			continue
		}
		if err := e.Start(obj.Name, methods); err != nil {
			log.Printf("executor: failed to start agent %q: %v", obj.Name, err)
		}
	}
}
