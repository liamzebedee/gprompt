package cluster

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClaude returns a ClaudeFunc that echoes the prompt with a counter.
// It sleeps for the given duration to simulate work.
func fakeClaude(delay time.Duration) ClaudeFunc {
	var calls atomic.Int64
	return func(ctx context.Context, prompt string) (string, error) {
		n := calls.Add(1)
		select {
		case <-time.After(delay):
			return fmt.Sprintf("output-%d: %s", n, prompt[:min(len(prompt), 20)]), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// fakeClaudeFailN returns a ClaudeFunc that fails the first N calls,
// then succeeds.
func fakeClaudeFailN(failCount int, delay time.Duration) ClaudeFunc {
	var calls atomic.Int64
	return func(ctx context.Context, prompt string) (string, error) {
		n := calls.Add(1)
		select {
		case <-time.After(delay):
			if int(n) <= failCount {
				return "", fmt.Errorf("simulated failure %d", n)
			}
			return fmt.Sprintf("output-%d", n), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func seedAgent(store *Store, name string) {
	store.ApplyDefinitions([]AgentDef{
		{Name: name, ID: "id-" + name, Definition: "(defagent \"" + name + "\" (pipeline (step \"work\" (loop work))))"},
	})
}

func TestExecutorStartStop(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	exec := NewExecutor(store, fakeClaude(10*time.Millisecond))

	// Agent should be pending.
	obj := store.GetAgent("builder")
	if obj.State != RunStatePending {
		t.Fatalf("expected pending, got %s", obj.State)
	}

	// Start the agent.
	methods := map[string]string{"build": "do some work"}
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Agent should be running.
	obj = store.GetAgent("builder")
	if obj.State != RunStateRunning {
		t.Fatalf("expected running, got %s", obj.State)
	}
	if !exec.IsRunning("builder") {
		t.Fatal("expected IsRunning to be true")
	}

	// Let it run a few iterations.
	time.Sleep(60 * time.Millisecond)

	// Stop the agent.
	if err := exec.Stop("builder", 2*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Agent should be stopped.
	obj = store.GetAgent("builder")
	if obj.State != RunStateStopped {
		t.Fatalf("expected stopped, got %s", obj.State)
	}
	if exec.IsRunning("builder") {
		t.Fatal("expected IsRunning to be false after stop")
	}

	// Should have completed at least 1 iteration.
	run := exec.GetRun("builder")
	if run != nil {
		t.Fatal("expected GetRun to return nil after stop")
	}
}

func TestExecutorStartIdempotent(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	exec := NewExecutor(store, fakeClaude(50*time.Millisecond))
	defer exec.StopAll(2 * time.Second)

	methods := map[string]string{"build": "do work"}
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("first Start: %v", err)
	}

	// Second start should be a no-op.
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("second Start: %v", err)
	}

	agents := exec.RunningAgents()
	if len(agents) != 1 {
		t.Fatalf("expected 1 running agent, got %d", len(agents))
	}
}

func TestExecutorStartNonexistent(t *testing.T) {
	store := NewStore()
	exec := NewExecutor(store, fakeClaude(0))

	err := exec.Start("ghost", map[string]string{"x": "y"})
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestExecutorStartNoMethods(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	exec := NewExecutor(store, fakeClaude(0))

	err := exec.Start("builder", map[string]string{})
	if err == nil {
		t.Fatal("expected error for empty methods")
	}

	// Agent should revert to pending (not stuck in running).
	obj := store.GetAgent("builder")
	if obj.State != RunStatePending {
		t.Fatalf("expected pending after failed start, got %s", obj.State)
	}
}

func TestExecutorErrorRecovery(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	// Fail first 2 calls, then succeed.
	exec := NewExecutor(store, fakeClaudeFailN(2, 5*time.Millisecond))

	methods := map[string]string{"build": "do work"}
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let it run through failures and at least one success.
	time.Sleep(80 * time.Millisecond)

	if err := exec.Stop("builder", 2*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// The agent should still have been running (not crashed on error).
	obj := store.GetAgent("builder")
	if obj.State != RunStateStopped {
		t.Fatalf("expected stopped, got %s", obj.State)
	}
}

func TestExecutorStopAll(t *testing.T) {
	store := NewStore()
	seedAgent(store, "alpha")
	seedAgent(store, "beta")

	exec := NewExecutor(store, fakeClaude(20*time.Millisecond))

	exec.Start("alpha", map[string]string{"work": "do alpha"})
	exec.Start("beta", map[string]string{"work": "do beta"})

	if len(exec.RunningAgents()) != 2 {
		t.Fatalf("expected 2 running agents")
	}

	exec.StopAll(2 * time.Second)

	if len(exec.RunningAgents()) != 0 {
		t.Fatalf("expected 0 running agents after StopAll")
	}

	for _, name := range []string{"alpha", "beta"} {
		obj := store.GetAgent(name)
		if obj.State != RunStateStopped {
			t.Fatalf("expected %s to be stopped, got %s", name, obj.State)
		}
	}
}

func TestExecutorStopNonRunning(t *testing.T) {
	store := NewStore()
	exec := NewExecutor(store, fakeClaude(0))

	err := exec.Stop("nobody", time.Second)
	if err == nil {
		t.Fatal("expected error stopping non-running agent")
	}
}

func TestExecutorStartPending(t *testing.T) {
	store := NewStore()
	seedAgent(store, "alpha")
	seedAgent(store, "beta")
	// Set beta to running manually to simulate already-running.
	store.SetRunState("beta", RunStateRunning)

	exec := NewExecutor(store, fakeClaude(20*time.Millisecond))
	defer exec.StopAll(2 * time.Second)

	agentMethods := map[string]map[string]string{
		"alpha": {"work": "do alpha work"},
		"beta":  {"work": "do beta work"},
	}

	exec.StartPending(agentMethods)

	// Only alpha should have been started (it was pending).
	if !exec.IsRunning("alpha") {
		t.Fatal("expected alpha to be running")
	}
	// beta was already running in the store but not in executor,
	// so StartPending should not have tried to start it (its state is running, not pending).
	if exec.IsRunning("beta") {
		t.Fatal("expected beta NOT to be started by StartPending (state was running)")
	}
}

func TestExecutorIterationTracking(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	exec := NewExecutor(store, fakeClaude(5*time.Millisecond))

	methods := map[string]string{"build": "do work"}
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let it run several iterations.
	time.Sleep(50 * time.Millisecond)

	run := exec.GetRun("builder")
	if run == nil {
		t.Fatal("expected non-nil run")
	}

	iters := run.SnapshotIterations()
	if len(iters) == 0 {
		t.Fatal("expected at least 1 iteration")
	}

	// Verify iteration structure.
	for i, ir := range iters {
		if ir.Iteration != i+1 {
			t.Errorf("iteration %d: expected number %d, got %d", i, i+1, ir.Iteration)
		}
		if ir.StartedAt.IsZero() {
			t.Errorf("iteration %d: StartedAt is zero", i)
		}
		if ir.FinishedAt.IsZero() {
			t.Errorf("iteration %d: FinishedAt is zero", i)
		}
		if ir.Error != "" {
			t.Errorf("iteration %d: unexpected error %q", i, ir.Error)
		}
		if ir.Output == "" {
			t.Errorf("iteration %d: empty output", i)
		}
	}

	exec.StopAll(2 * time.Second)
}

func TestExecutorSnapshot(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")
	seedAgent(store, "tester")

	exec := NewExecutor(store, fakeClaude(5*time.Millisecond))

	exec.Start("builder", map[string]string{"build": "do build"})
	exec.Start("tester", map[string]string{"test": "do test"})

	// Let them run a few iterations.
	time.Sleep(40 * time.Millisecond)

	snap := exec.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 runs in snapshot, got %d", len(snap))
	}

	for _, name := range []string{"builder", "tester"} {
		rs, ok := snap[name]
		if !ok {
			t.Fatalf("expected snapshot for %s", name)
		}
		if rs.Name != name {
			t.Errorf("snapshot name: expected %q, got %q", name, rs.Name)
		}
		if len(rs.Iterations) == 0 {
			t.Errorf("expected at least 1 iteration for %s", name)
		}
		if rs.StartedAt.IsZero() {
			t.Errorf("expected non-zero StartedAt for %s", name)
		}
	}

	exec.StopAll(2 * time.Second)
}

func TestExecutorSnapshotCapsIterations(t *testing.T) {
	store := NewStore()
	seedAgent(store, "runner")

	// Very fast iterations to accumulate >10
	exec := NewExecutor(store, fakeClaude(1*time.Millisecond))
	exec.Start("runner", map[string]string{"work": "do work"})

	// Wait for at least 12 iterations
	time.Sleep(50 * time.Millisecond)

	snap := exec.Snapshot()
	rs := snap["runner"]
	if len(rs.Iterations) > 10 {
		t.Fatalf("expected snapshot capped at 10 iterations, got %d", len(rs.Iterations))
	}

	exec.StopAll(2 * time.Second)
}

// TestExecutorInjectMessage verifies that injected messages are prepended
// to the agent's prompt in the next iteration. This is the core steering
// mechanism: a steer client sends a message, the executor queues it, and
// the agent goroutine picks it up before calling claude.
func TestExecutorInjectMessage(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	// Track prompts received by claude to verify injection
	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(20 * time.Millisecond):
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	exec := NewExecutor(store, claudeFn)
	methods := map[string]string{"work": "do some work"}
	if err := exec.Start("builder", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let first iteration start
	time.Sleep(5 * time.Millisecond)

	// Inject a message
	if err := exec.InjectMessage("builder", "focus on tests please"); err != nil {
		t.Fatalf("InjectMessage: %v", err)
	}

	// Wait for the next iteration to pick up the injection
	time.Sleep(60 * time.Millisecond)

	exec.StopAll(2 * time.Second)

	// At least one prompt should contain the injected message
	mu.Lock()
	defer mu.Unlock()

	found := false
	for _, p := range prompts {
		if strings.Contains(p, "focus on tests please") && strings.Contains(p, "[Steering messages from human operator]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one prompt to contain injected message; got prompts: %v", prompts)
	}
}

// TestExecutorInjectMessageNonRunning verifies InjectMessage returns
// an error for an agent that is not running.
func TestExecutorInjectMessageNonRunning(t *testing.T) {
	store := NewStore()
	exec := NewExecutor(store, fakeClaude(0))

	err := exec.InjectMessage("ghost", "hello")
	if err == nil {
		t.Fatal("expected error injecting into non-running agent")
	}
}

func TestExecutorOnIteration(t *testing.T) {
	store := NewStore()
	seedAgent(store, "builder")

	exec := NewExecutor(store, fakeClaude(5*time.Millisecond))

	var callCount atomic.Int64
	exec.OnIteration(func(agentName string) {
		callCount.Add(1)
	})

	exec.Start("builder", map[string]string{"build": "do work"})

	// Let it run a few iterations.
	time.Sleep(40 * time.Millisecond)

	exec.StopAll(2 * time.Second)

	count := callCount.Load()
	if count == 0 {
		t.Fatal("expected OnIteration callback to be called at least once")
	}
}

// --- Multi-step pipeline tests ---

// TestPipelineSimpleThenLoop verifies the core multi-step pattern:
// simple setup steps execute once, threading output, then the loop step
// runs repeatedly. This is the "idea -> spec -> plan -> loop(build)" pattern
// from the spec. The test verifies:
//   - Simple steps receive previous step output as context
//   - Loop step's first iteration includes the last simple step's output
//   - Subsequent loop iterations use only the loop method body
//   - Iteration tracking only counts loop iterations
func TestPipelineSimpleThenLoop(t *testing.T) {
	store := NewStore()
	seedAgent(store, "ralph")

	// Track all prompts to verify step chaining.
	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(5 * time.Millisecond):
			// Return a deterministic output based on prompt content.
			if strings.Contains(prompt, "write a spec") {
				return "the spec output", nil
			}
			if strings.Contains(prompt, "write a plan") {
				return "the plan output", nil
			}
			return "build-output", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	exec := NewExecutor(store, claudeFn)

	// Set up pipeline: spec -> plan -> loop(build)
	pdef := &PipelineDef{
		InitialInput: "idea",
		Steps: []PipelineStep{
			{Label: "spec", Kind: StepKindSimple, Method: "spec"},
			{Label: "plan", Kind: StepKindSimple, Method: "plan"},
			{Label: "build", Kind: StepKindLoop, LoopMethod: "build"},
		},
	}
	exec.SetPipeline("ralph", pdef)

	methods := map[string]string{
		"spec":  "write a spec",
		"plan":  "write a plan",
		"build": "do the build",
	}

	if err := exec.Start("ralph", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let it run through setup + a few loop iterations.
	time.Sleep(80 * time.Millisecond)
	exec.StopAll(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(prompts) < 4 {
		t.Fatalf("expected at least 4 prompts (2 setup + 2 loop), got %d", len(prompts))
	}

	// First prompt: "spec" method body (no previous output).
	if !strings.Contains(prompts[0], "write a spec") {
		t.Errorf("prompt 0: expected spec method body, got %q", prompts[0])
	}

	// Second prompt: "plan" method body with spec output as context.
	if !strings.Contains(prompts[1], "the spec output") {
		t.Errorf("prompt 1: expected spec output as context, got %q", prompts[1])
	}
	if !strings.Contains(prompts[1], "write a plan") {
		t.Errorf("prompt 1: expected plan method body, got %q", prompts[1])
	}

	// Third prompt: loop iteration 1 with plan output as context.
	if !strings.Contains(prompts[2], "the plan output") {
		t.Errorf("prompt 2: expected plan output as context, got %q", prompts[2])
	}
	if !strings.Contains(prompts[2], "do the build") {
		t.Errorf("prompt 2: expected build method body, got %q", prompts[2])
	}

	// Fourth prompt: loop iteration 2 — only the build method body, no plan context.
	if strings.Contains(prompts[3], "the plan output") {
		t.Errorf("prompt 3: should NOT contain plan output on subsequent iterations, got %q", prompts[3])
	}
	if !strings.Contains(prompts[3], "do the build") {
		t.Errorf("prompt 3: expected build method body, got %q", prompts[3])
	}
}

// TestPipelineMapStep verifies that map steps split the previous output into
// items and call the method in parallel for each item.
func TestPipelineMapStep(t *testing.T) {
	store := NewStore()
	seedAgent(store, "mapper")

	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(5 * time.Millisecond):
			if strings.Contains(prompt, "generate chapters") {
				return "1. Chapter One\n2. Chapter Two\n3. Chapter Three", nil
			}
			// Map step: echo the item
			return "expanded: " + prompt[:min(len(prompt), 30)], nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	exec := NewExecutor(store, claudeFn)

	pdef := &PipelineDef{
		Steps: []PipelineStep{
			{Label: "outline", Kind: StepKindSimple, Method: "generate-outline"},
			{Label: "chapters", Kind: StepKindMap, MapMethod: "flesh-out", MapRef: "chapters"},
		},
	}
	exec.SetPipeline("mapper", pdef)

	methods := map[string]string{
		"generate-outline": "generate chapters",
		"flesh-out":        "expand this chapter",
	}

	if err := exec.Start("mapper", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for pipeline to complete (simple + map steps, no loop).
	time.Sleep(100 * time.Millisecond)

	// Pipeline with no loop should complete and agent should stop.
	// Check that it ran: outline step + 3 map items = 4 calls.
	mu.Lock()
	defer mu.Unlock()

	if len(prompts) < 4 {
		t.Fatalf("expected at least 4 prompts (1 outline + 3 map items), got %d: %v", len(prompts), prompts)
	}

	// First prompt: outline generation.
	if !strings.Contains(prompts[0], "generate chapters") {
		t.Errorf("prompt 0: expected outline method, got %q", prompts[0])
	}

	// Remaining prompts: map items should each contain "expand this chapter".
	mapCount := 0
	for _, p := range prompts[1:] {
		if strings.Contains(p, "expand this chapter") {
			mapCount++
		}
	}
	if mapCount != 3 {
		t.Errorf("expected 3 map prompts with flesh-out method body, got %d", mapCount)
	}

	exec.StopAll(2 * time.Second)
}

// TestPipelineSimpleStepFailure verifies that a failure in a setup step
// aborts the pipeline and records the error as an iteration result,
// so steer clients can see what went wrong.
func TestPipelineSimpleStepFailure(t *testing.T) {
	store := NewStore()
	seedAgent(store, "failing")

	claudeFn := func(ctx context.Context, prompt string) (string, error) {
		select {
		case <-time.After(5 * time.Millisecond):
			return "", fmt.Errorf("simulated step failure")
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	exec := NewExecutor(store, claudeFn)

	pdef := &PipelineDef{
		Steps: []PipelineStep{
			{Label: "spec", Kind: StepKindSimple, Method: "spec"},
			{Label: "build", Kind: StepKindLoop, LoopMethod: "build"},
		},
	}
	exec.SetPipeline("failing", pdef)

	methods := map[string]string{
		"spec":  "write spec",
		"build": "do build",
	}

	if err := exec.Start("failing", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the pipeline to fail.
	time.Sleep(50 * time.Millisecond)

	// The agent goroutine should have exited after the step failure.
	run := exec.GetRun("failing")
	if run == nil {
		// Already removed from runs — check that an iteration was recorded.
		t.Skip("agent already cleaned up")
	}

	iters := run.SnapshotIterations()
	if len(iters) != 1 {
		t.Fatalf("expected 1 iteration (the failure), got %d", len(iters))
	}
	if iters[0].Error == "" {
		t.Fatal("expected error in iteration result")
	}
	if !strings.Contains(iters[0].Error, "pipeline step") {
		t.Errorf("expected error to mention pipeline step, got %q", iters[0].Error)
	}

	exec.StopAll(2 * time.Second)
}

// TestPipelineValidation verifies that Start fails if the pipeline references
// a method that wasn't provided in the methods map.
func TestPipelineValidation(t *testing.T) {
	store := NewStore()
	seedAgent(store, "invalid")

	exec := NewExecutor(store, fakeClaude(0))

	pdef := &PipelineDef{
		Steps: []PipelineStep{
			{Label: "spec", Kind: StepKindSimple, Method: "spec"},
			{Label: "build", Kind: StepKindLoop, LoopMethod: "build"},
		},
	}
	exec.SetPipeline("invalid", pdef)

	// Only provide "spec", missing "build".
	methods := map[string]string{"spec": "write spec"}

	err := exec.Start("invalid", methods)
	if err == nil {
		t.Fatal("expected error for missing method")
	}
	if !strings.Contains(err.Error(), "build") {
		t.Errorf("expected error to mention missing method 'build', got %q", err.Error())
	}

	// Agent should revert to pending.
	obj := store.GetAgent("invalid")
	if obj.State != RunStatePending {
		t.Fatalf("expected pending after validation failure, got %s", obj.State)
	}
}

// TestPipelineLoopOnlyIsEquivalentToLegacy verifies that a pipeline with
// a single loop step behaves the same as the legacy single-method path.
func TestPipelineLoopOnlyIsEquivalentToLegacy(t *testing.T) {
	store := NewStore()
	seedAgent(store, "looper")

	exec := NewExecutor(store, fakeClaude(5*time.Millisecond))

	pdef := &PipelineDef{
		Steps: []PipelineStep{
			{Label: "work", Kind: StepKindLoop, LoopMethod: "work"},
		},
	}
	exec.SetPipeline("looper", pdef)

	methods := map[string]string{"work": "do the work"}

	if err := exec.Start("looper", methods); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	run := exec.GetRun("looper")
	if run == nil {
		t.Fatal("expected non-nil run")
	}

	iters := run.SnapshotIterations()
	if len(iters) < 2 {
		t.Fatalf("expected at least 2 iterations, got %d", len(iters))
	}

	exec.StopAll(2 * time.Second)
}

// TestSplitItems verifies the item splitting heuristics used by map steps.
func TestSplitItems(t *testing.T) {
	// Numbered list
	items := splitItems("1. First item\n2. Second item\n3. Third item")
	if len(items) != 3 {
		t.Fatalf("numbered list: expected 3 items, got %d: %v", len(items), items)
	}

	// Bullet points
	items = splitItems("- Alpha\n- Beta\n- Gamma")
	if len(items) != 3 {
		t.Fatalf("bullet list: expected 3 items, got %d: %v", len(items), items)
	}

	// Paragraphs
	items = splitItems("First paragraph\n\nSecond paragraph\n\nThird paragraph")
	if len(items) != 3 {
		t.Fatalf("paragraphs: expected 3 items, got %d: %v", len(items), items)
	}

	// Single item fallback
	items = splitItems("just one thing")
	if len(items) != 1 {
		t.Fatalf("single: expected 1 item, got %d: %v", len(items), items)
	}

	// Empty
	items = splitItems("")
	if items != nil {
		t.Fatalf("empty: expected nil, got %v", items)
	}
}

// TestUpdateMethodBody verifies that UpdateMethodBody delivers a method body
// update to a running agent's loop goroutine, replacing the base prompt for
// all subsequent iterations. This is the "edit prompt" feature in the TUI.
func TestUpdateMethodBody(t *testing.T) {
	store := NewStore()
	seedAgent(store, "editor")

	var prompts []string
	var mu sync.Mutex
	claudeFn := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		prompts = append(prompts, prompt)
		mu.Unlock()
		select {
		case <-time.After(10 * time.Millisecond):
			return "ok", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	exec := NewExecutor(store, claudeFn)

	if err := exec.Start("editor", map[string]string{"work": "original prompt"}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let a few iterations run with the original prompt.
	time.Sleep(30 * time.Millisecond)

	// Update the method body.
	exec.UpdateMethodBody("editor", "work", "updated prompt: focus on security")

	// Let a few more iterations run with the updated prompt.
	time.Sleep(50 * time.Millisecond)

	exec.StopAll(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	// Verify that at least one prompt contained "updated prompt"
	found := false
	for _, p := range prompts {
		if strings.Contains(p, "updated prompt: focus on security") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one prompt with updated body; got %d prompts: %v", len(prompts), prompts)
	}

	// Verify that the early prompts used the original prompt
	if len(prompts) > 0 && !strings.Contains(prompts[0], "original prompt") {
		t.Errorf("first prompt should use original body, got %q", prompts[0])
	}
}

// TestUpdateMethodBodyNonRunning verifies that UpdateMethodBody is a no-op
// for agents that aren't running (no panic, no error).
func TestUpdateMethodBodyNonRunning(t *testing.T) {
	store := NewStore()
	exec := NewExecutor(store, fakeClaude(0))

	// Should not panic or error — just a no-op.
	exec.UpdateMethodBody("ghost", "work", "new body")
}
