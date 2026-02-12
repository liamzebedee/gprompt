// Executor manages agent goroutine lifecycles within the gcluster master.
//
// When the server applies agent definitions, it calls Executor.Start for each
// agent that is in pending state. The executor spawns a goroutine per agent
// that runs the agent's pipeline — either a simple loop or a multi-step
// pipeline with simple, map, and loop steps.
//
// Design decisions:
//
//   - The executor does NOT parse S-expressions or import the pipeline package.
//     The apply command resolves method bodies and pipeline structure at apply
//     time and sends them alongside the definition in AgentDef.Methods and
//     AgentDef.Pipeline. This keeps the executor decoupled from the
//     parser/compiler/sexp/pipeline packages.
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
//   - Multi-step pipelines execute simple and map steps sequentially as setup,
//     threading each step's output into the next. The final step (typically a
//     loop) runs the iteration loop. Iteration tracking counts only the loop
//     step's iterations — setup steps are one-shot initialization.
//
//   - Thread safety: the running map is guarded by a mutex. Store mutations
//     go through Store methods which have their own locks.
package cluster

import (
	"context"
	"fmt"
	"log"
	"strings"
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

// methodUpdate carries a method body update from a steer client to a running
// agent's loop goroutine. The agent drains these between iterations and replaces
// its base prompt so all future iterations use the new text.
type methodUpdate struct {
	MethodName string
	NewBody    string
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

	// injectCh receives steering messages from steer clients. The runAgent
	// goroutine drains this channel between iterations and prepends the
	// messages to the next prompt, allowing humans to nudge the agent.
	injectCh chan string

	// methodCh receives method body updates from steer clients. The runAgent
	// goroutine drains this channel between iterations and replaces the base
	// prompt so all future iterations use the new text.
	methodCh chan methodUpdate

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

// AgentRunSnapshot is a serializable snapshot of a running agent's iteration history.
// It is included in SteerStatePayload for steer clients, NOT persisted to disk.
// This keeps runtime/ephemeral iteration data separate from the declarative
// ClusterObject model that gets persisted.
type AgentRunSnapshot struct {
	Name       string            `json:"name"`
	RevisionID string            `json:"revision_id"`
	StartedAt  time.Time         `json:"started_at"`
	Iterations []IterationResult `json:"iterations"`
}

// Executor manages the lifecycle of running agent goroutines.
// It is owned by the Server and operates on the shared Store.
type Executor struct {
	store    *Store
	claudeFn ClaudeFunc
	rootCtx  context.Context
	rootStop context.CancelFunc

	mu          sync.Mutex
	runs        map[string]*AgentRun   // keyed by agent name
	pipelines   map[string]*PipelineDef // keyed by agent name, cached from apply
	onIteration func(agentName string)  // called after each iteration completes
}

// NewExecutor creates an executor bound to a store and a claude invocation
// function. The rootCtx should be derived from the server's shutdown context;
// cancelling it will stop all running agents.
func NewExecutor(store *Store, claudeFn ClaudeFunc) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		store:     store,
		claudeFn:  claudeFn,
		rootCtx:   ctx,
		rootStop:  cancel,
		runs:      make(map[string]*AgentRun),
		pipelines: make(map[string]*PipelineDef),
	}
}

// SetPipeline caches a pipeline definition for an agent. Called by the server
// when processing apply requests so the executor knows the step structure.
func (e *Executor) SetPipeline(name string, p *PipelineDef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p != nil {
		e.pipelines[name] = p
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

	// Grab cached pipeline def if available.
	pdef := e.pipelines[name]
	e.mu.Unlock()

	// Transition to running in the store. This is done outside the executor
	// lock because SetRunState triggers Store.OnChange, which may call
	// pushState → Executor.Snapshot(), creating a lock ordering issue.
	if !e.store.SetRunState(name, RunStateRunning) {
		return fmt.Errorf("agent %q: failed to set running state", name)
	}

	e.mu.Lock()
	// Re-check: another goroutine might have started this agent between
	// our unlock above and this lock.
	if _, running := e.runs[name]; running {
		e.mu.Unlock()
		return nil
	}

	agentCtx, agentCancel := context.WithCancel(e.rootCtx)
	run := &AgentRun{
		Name:       name,
		RevisionID: obj.CurrentRevision,
		StartedAt:  time.Now(),
		injectCh:   make(chan string, 32),
		methodCh:   make(chan methodUpdate, 4),
		cancel:     agentCancel,
		done:       make(chan struct{}),
	}
	e.runs[name] = run
	e.mu.Unlock()

	// Choose execution path based on pipeline structure.
	if pdef != nil && len(pdef.Steps) > 0 {
		// Multi-step pipeline: execute setup steps then loop.
		if err := e.validatePipeline(pdef, methods); err != nil {
			e.store.SetRunState(name, RunStatePending)
			e.mu.Lock()
			delete(e.runs, name)
			e.mu.Unlock()
			agentCancel()
			close(run.done)
			return fmt.Errorf("agent %q: invalid pipeline: %w", name, err)
		}
		go e.runPipeline(agentCtx, run, pdef, methods)
	} else {
		// Legacy single-method path.
		prompt, err := e.resolvePrompt(methods)
		if err != nil {
			e.store.SetRunState(name, RunStatePending)
			e.mu.Lock()
			delete(e.runs, name)
			e.mu.Unlock()
			agentCancel()
			close(run.done)
			return fmt.Errorf("agent %q: resolve prompt: %w", name, err)
		}
		go func() {
			defer close(run.done)
			e.runAgentLoop(agentCtx, run, prompt, prompt)
		}()
	}

	revShort := run.RevisionID
	if len(revShort) > 8 {
		revShort = revShort[:8]
	}
	log.Printf("executor: started agent %q (revision %s)", name, revShort)
	return nil
}

// validatePipeline checks that all methods referenced by pipeline steps exist.
func (e *Executor) validatePipeline(p *PipelineDef, methods map[string]string) error {
	for i, step := range p.Steps {
		var methodName string
		switch step.Kind {
		case StepKindSimple:
			methodName = step.Method
		case StepKindLoop:
			methodName = step.LoopMethod
		case StepKindMap:
			methodName = step.MapMethod
		default:
			return fmt.Errorf("step %d (%s): unknown kind %q", i+1, step.Label, step.Kind)
		}
		if methodName == "" {
			return fmt.Errorf("step %d (%s): no method name", i+1, step.Label)
		}
		if _, ok := methods[methodName]; !ok {
			return fmt.Errorf("step %d (%s): method %q not found in resolved methods", i+1, step.Label, methodName)
		}
	}
	return nil
}

// resolvePrompt extracts a single prompt from the methods map for legacy
// single-method agents (no PipelineDef available).
func (e *Executor) resolvePrompt(methods map[string]string) (string, error) {
	if len(methods) == 0 {
		return "", fmt.Errorf("no method bodies provided")
	}
	if len(methods) == 1 {
		for _, body := range methods {
			return body, nil
		}
	}
	return "", fmt.Errorf("multiple methods provided but no pipeline structure; cannot determine execution order")
}

// runPipeline executes a multi-step pipeline. Steps before the final loop
// run once (simple) or fan-out (map), threading output between steps.
// The final step, if a loop, runs forever with iteration tracking.
// If all steps are non-loop, the pipeline runs once to completion.
//
// Why iteration tracking only covers the loop step: simple and map steps
// are one-shot setup. The loop step is where the agent does ongoing work,
// and it's what steer clients observe and steer. Tracking only loop
// iterations keeps the model simple and matches the TUI's expectations.
func (e *Executor) runPipeline(ctx context.Context, run *AgentRun, p *PipelineDef, methods map[string]string) {
	defer close(run.done)

	var prevOutput string

	for i, step := range p.Steps {
		// Check cancellation between steps.
		select {
		case <-ctx.Done():
			log.Printf("executor: agent %q pipeline cancelled at step %d (%s)", run.Name, i+1, step.Label)
			return
		default:
		}

		switch step.Kind {
		case StepKindSimple:
			body := methods[step.Method]
			prompt := body
			if prevOutput != "" {
				prompt = prevOutput + "\n\n" + body
			}

			log.Printf("executor: agent %q running simple step %d/%d (%s)", run.Name, i+1, len(p.Steps), step.Label)
			output, err := e.claudeFn(ctx, prompt)
			if err != nil {
				if ctx.Err() != nil {
					log.Printf("executor: agent %q step %d (%s) cancelled", run.Name, i+1, step.Label)
					return
				}
				// Setup step failure aborts the pipeline. Record it as a
				// failed iteration so steer clients can see what happened.
				log.Printf("executor: agent %q step %d (%s) failed: %v — pipeline aborted", run.Name, i+1, step.Label, err)
				run.addIteration(IterationResult{
					Iteration:  1,
					StartedAt:  time.Now(),
					FinishedAt: time.Now(),
					Error:      fmt.Sprintf("pipeline step %d (%s): %v", i+1, step.Label, err),
				})
				e.fireOnIteration(run.Name)
				return
			}
			prevOutput = output
			log.Printf("executor: agent %q step %d (%s) complete (%d bytes)", run.Name, i+1, step.Label, len(output))

		case StepKindMap:
			body := methods[step.MapMethod]
			items := splitItems(prevOutput)
			if len(items) == 0 {
				log.Printf("executor: agent %q step %d (%s): map got 0 items — pipeline aborted", run.Name, i+1, step.Label)
				run.addIteration(IterationResult{
					Iteration:  1,
					StartedAt:  time.Now(),
					FinishedAt: time.Now(),
					Error:      fmt.Sprintf("pipeline step %d (%s): map got 0 items from previous output", i+1, step.Label),
				})
				e.fireOnIteration(run.Name)
				return
			}

			log.Printf("executor: agent %q running map step %d/%d (%s) with %d items", run.Name, i+1, len(p.Steps), step.Label, len(items))

			results := make([]string, len(items))
			var mu sync.Mutex
			var wg sync.WaitGroup
			var firstErr error

			mapCtx, mapCancel := context.WithCancel(ctx)
			for j, item := range items {
				wg.Add(1)
				go func(idx int, itemText string) {
					defer wg.Done()
					prompt := itemText + "\n\n" + body
					result, err := e.claudeFn(mapCtx, prompt)
					mu.Lock()
					defer mu.Unlock()
					if err != nil && firstErr == nil {
						firstErr = fmt.Errorf("map item %d: %w", idx+1, err)
						mapCancel() // cancel remaining items on first error
					}
					results[idx] = result
				}(j, item)
			}
			wg.Wait()
			mapCancel() // ensure cancel is always called

			if firstErr != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("executor: agent %q step %d (%s) map failed: %v — pipeline aborted", run.Name, i+1, step.Label, firstErr)
				run.addIteration(IterationResult{
					Iteration:  1,
					StartedAt:  time.Now(),
					FinishedAt: time.Now(),
					Error:      fmt.Sprintf("pipeline step %d (%s): %v", i+1, step.Label, firstErr),
				})
				e.fireOnIteration(run.Name)
				return
			}
			prevOutput = strings.Join(results, "\n\n---\n\n")
			log.Printf("executor: agent %q step %d (%s) map complete (%d items)", run.Name, i+1, step.Label, len(items))

		case StepKindLoop:
			body := methods[step.LoopMethod]
			// First iteration gets previous step output as context.
			// Subsequent iterations use just the method body (plus steering).
			firstPrompt := body
			if prevOutput != "" {
				firstPrompt = prevOutput + "\n\n" + body
			}
			log.Printf("executor: agent %q entering loop step %d/%d (%s)", run.Name, i+1, len(p.Steps), step.Label)
			e.runAgentLoop(ctx, run, firstPrompt, body)
			return // loop never finishes normally
		}
	}

	// Pipeline completed with no loop step (all simple/map).
	log.Printf("executor: agent %q pipeline complete (no loop step)", run.Name)
}

// runAgentLoop is the inner loop for a loop step. It calls claude repeatedly
// with the prompt, handling steering injection between iterations.
//
// On the first iteration, firstPrompt is used (may include context from
// previous pipeline steps). On subsequent iterations, basePrompt is used
// (just the loop method body, plus any steering messages).
//
// This is also the execution path for legacy single-method agents where
// firstPrompt == basePrompt.
func (e *Executor) runAgentLoop(ctx context.Context, run *AgentRun, firstPrompt string, basePrompt string) {
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

		iterPrompt := basePrompt
		if iteration == 1 {
			iterPrompt = firstPrompt
		}

		// Drain any injected steering messages and prepend them to the prompt
		// for this iteration. This allows steer clients to influence the agent's
		// next action without restarting it.
		var injected []string
	drain:
		for {
			select {
			case msg := <-run.injectCh:
				injected = append(injected, msg)
			default:
				break drain
			}
		}
		if len(injected) > 0 {
			var sb strings.Builder
			sb.WriteString("[Steering messages from human operator]\n")
			for _, msg := range injected {
				sb.WriteString("- ")
				sb.WriteString(msg)
				sb.WriteString("\n")
			}
			sb.WriteString("[End of steering messages]\n\n")
			sb.WriteString(iterPrompt)
			iterPrompt = sb.String()
			log.Printf("executor: agent %q iteration %d: %d injected message(s) prepended to prompt", run.Name, iteration, len(injected))
		}

		// Drain any method body updates. If the loop's method was updated via
		// the steer TUI's "edit prompt" feature, replace basePrompt so all
		// future iterations use the new text. This is a permanent change,
		// unlike inject which is a one-time prepend.
	drainMethods:
		for {
			select {
			case upd := <-run.methodCh:
				basePrompt = upd.NewBody
				iterPrompt = basePrompt // also update this iteration's prompt
				log.Printf("executor: agent %q updated base prompt from method %q (%d bytes)",
					run.Name, upd.MethodName, len(upd.NewBody))
			default:
				break drainMethods
			}
		}

		ir := IterationResult{
			Iteration: iteration,
			StartedAt: time.Now(),
		}

		log.Printf("executor: agent %q starting iteration %d", run.Name, iteration)
		output, err := e.claudeFn(ctx, iterPrompt)
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
			e.fireOnIteration(run.Name)
			log.Printf("executor: agent %q iteration %d failed: %v (continuing)", run.Name, iteration, err)
			continue
		}

		ir.Output = output
		run.addIteration(ir)
		e.fireOnIteration(run.Name)
		log.Printf("executor: agent %q iteration %d complete (%d bytes output)", run.Name, iteration, len(output))
	}
}

// fireOnIteration calls the onIteration callback if set.
func (e *Executor) fireOnIteration(agentName string) {
	e.mu.Lock()
	fn := e.onIteration
	e.mu.Unlock()
	if fn != nil {
		fn(agentName)
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

// InjectMessage delivers a steering message to a running agent. The message
// is queued on the agent's inject channel and will be prepended to the prompt
// for the agent's next iteration. This allows steer clients to influence
// running agents without restarting them.
//
// Returns an error if the agent is not currently running.
func (e *Executor) InjectMessage(agentName string, message string) error {
	e.mu.Lock()
	run, ok := e.runs[agentName]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("agent %q is not running", agentName)
	}

	// Non-blocking send. If the buffer is full (32 messages), log a warning
	// and drop the oldest message to make room.
	select {
	case run.injectCh <- message:
		log.Printf("executor: injected message for agent %q (%d bytes)", agentName, len(message))
	default:
		// Channel full — drain one and retry
		select {
		case <-run.injectCh:
		default:
		}
		run.injectCh <- message
		log.Printf("executor: injected message for agent %q (dropped oldest, buffer was full)", agentName)
	}

	return nil
}

// UpdateMethodBody sends a method body update to a running agent. The agent's
// loop goroutine will pick up the change before the next iteration and replace
// its base prompt. If the agent is not running, this is a no-op — the server's
// cached method bodies will be used when the agent next starts.
func (e *Executor) UpdateMethodBody(agentName, methodName, newBody string) {
	e.mu.Lock()
	run, ok := e.runs[agentName]
	e.mu.Unlock()

	if !ok {
		return // agent not running; cache update in server is sufficient
	}

	select {
	case run.methodCh <- methodUpdate{MethodName: methodName, NewBody: newBody}:
		log.Printf("executor: queued method update for agent %q method %q", agentName, methodName)
	default:
		// Channel full — drain one and retry
		select {
		case <-run.methodCh:
		default:
		}
		run.methodCh <- methodUpdate{MethodName: methodName, NewBody: newBody}
		log.Printf("executor: queued method update for agent %q method %q (replaced old)", agentName, methodName)
	}
}

// OnIteration registers a callback invoked after each iteration completes.
// This is used by the server to trigger state pushes to steer clients when
// new iteration data is available, since Store.OnChange only fires on
// store mutations (apply, state changes), not iteration completions.
func (e *Executor) OnIteration(fn func(agentName string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onIteration = fn
}

// Snapshot returns a map of agent name → run snapshot for all running agents.
// Used by the server to include iteration data in SteerStatePayload.
func (e *Executor) Snapshot() map[string]AgentRunSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make(map[string]AgentRunSnapshot, len(e.runs))
	for name, run := range e.runs {
		iters := run.SnapshotIterations()
		// Cap to last 10 iterations to limit payload size.
		if len(iters) > 10 {
			iters = iters[len(iters)-10:]
		}
		result[name] = AgentRunSnapshot{
			Name:       run.Name,
			RevisionID: run.RevisionID,
			StartedAt:  run.StartedAt,
			Iterations: iters,
		}
	}
	return result
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

// splitItems splits text into items using heuristics:
// tries numbered lists, markdown headings, bullet points, then paragraphs.
// This is a copy of the logic from runtime/runtime.go, duplicated here
// because the executor must not import the runtime package (which depends
// on the pipeline and registry packages).
func splitItems(text string) []string {
	lines := strings.Split(text, "\n")

	// Try numbered list (e.g., "1. ", "2. ")
	var numbered []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 2 && trimmed[0] >= '1' && trimmed[0] <= '9' &&
			(strings.HasPrefix(trimmed[1:], ". ") ||
				(len(trimmed) > 3 && trimmed[1] >= '0' && trimmed[1] <= '9' && strings.HasPrefix(trimmed[2:], ". "))) {
			if current.Len() > 0 {
				numbered = append(numbered, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 {
			current.WriteString("\n" + trimmed)
		}
	}
	if current.Len() > 0 {
		numbered = append(numbered, strings.TrimSpace(current.String()))
	}
	if len(numbered) >= 2 {
		return numbered
	}

	// Try markdown headings (## or #)
	var headingSections []string
	current.Reset()
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if current.Len() > 0 {
				headingSections = append(headingSections, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 {
			current.WriteString("\n" + line)
		}
	}
	if current.Len() > 0 {
		headingSections = append(headingSections, strings.TrimSpace(current.String()))
	}
	if len(headingSections) >= 2 {
		return headingSections
	}

	// Try bullet points (- or *)
	var bullets []string
	current.Reset()
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if current.Len() > 0 {
				bullets = append(bullets, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 && trimmed != "" {
			current.WriteString("\n" + trimmed)
		}
	}
	if current.Len() > 0 {
		bullets = append(bullets, strings.TrimSpace(current.String()))
	}
	if len(bullets) >= 2 {
		return bullets
	}

	// Fallback: split on double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")
	var result []string
	for _, p := range paragraphs {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	if len(result) >= 2 {
		return result
	}

	// Last resort: return the whole thing as one item
	if t := strings.TrimSpace(text); t != "" {
		return []string{t}
	}
	return nil
}
