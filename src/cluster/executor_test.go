package cluster

import (
	"context"
	"fmt"
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
