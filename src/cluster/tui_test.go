package cluster

import (
	"testing"
	"time"
)

func TestExtractLoopMethod(t *testing.T) {
	tests := []struct {
		def    string
		expect string
	}{
		{`(defagent "builder" (pipeline (step "build" (loop build))))`, "build"},
		{`(defagent "tester" (pipeline (step "test" (loop test))))`, "test"},
		{`(defagent "simple" (invoke method))`, ""},
		{``, ""},
	}

	for _, tt := range tests {
		got := extractLoopMethod(tt.def)
		if got != tt.expect {
			t.Errorf("extractLoopMethod(%q) = %q, want %q", tt.def, got, tt.expect)
		}
	}
}

func TestTUIRebuildTree(t *testing.T) {
	m := NewTUIModel(nil) // nil client is ok for tree tests
	m.objects = []ClusterObject{
		{
			Name:       "builder",
			State:      RunStateRunning,
			Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`,
		},
		{
			Name:       "tester",
			State:      RunStatePending,
			Definition: `(defagent "tester" (pipeline (step "test" (loop test))))`,
		},
	}
	m.runs = map[string]AgentRunSnapshot{
		"builder": {
			Name:      "builder",
			StartedAt: time.Now(),
			Iterations: []IterationResult{
				{Iteration: 1, StartedAt: time.Now().Add(-30 * time.Second), FinishedAt: time.Now().Add(-20 * time.Second), Output: "done 1"},
				{Iteration: 2, StartedAt: time.Now().Add(-20 * time.Second), FinishedAt: time.Now().Add(-10 * time.Second), Output: "done 2"},
				{Iteration: 3, StartedAt: time.Now().Add(-10 * time.Second), FinishedAt: time.Now(), Output: "done 3"},
			},
		},
	}

	m.rebuildTree()

	// Should have 2 agent nodes
	if len(m.tree) != 2 {
		t.Fatalf("expected 2 agent nodes, got %d", len(m.tree))
	}

	// First agent (builder) should have loop with 3 iterations
	builderNode := m.tree[0]
	if builderNode.AgentName != "builder" {
		t.Fatalf("expected first agent 'builder', got %q", builderNode.AgentName)
	}
	if len(builderNode.Children) != 1 {
		t.Fatalf("expected 1 loop child, got %d", len(builderNode.Children))
	}

	loopNode := builderNode.Children[0]
	if loopNode.Kind != NodeLoop {
		t.Fatalf("expected loop node, got kind %d", loopNode.Kind)
	}
	if loopNode.Label != "loop(build)" {
		t.Fatalf("expected label 'loop(build)', got %q", loopNode.Label)
	}
	if len(loopNode.Children) != 3 {
		t.Fatalf("expected 3 iteration children, got %d", len(loopNode.Children))
	}

	// Most recent iteration should be first
	firstIter := loopNode.Children[0]
	if firstIter.Iteration != 3 {
		t.Fatalf("expected first iteration to be 3 (most recent), got %d", firstIter.Iteration)
	}
}

func TestTUIRebuildTreeMaxIterations(t *testing.T) {
	m := NewTUIModel(nil)
	m.objects = []ClusterObject{
		{
			Name:       "runner",
			Definition: `(defagent "runner" (pipeline (step "run" (loop run))))`,
		},
	}

	// Create 10 iterations — only 4 should show in tree
	var iters []IterationResult
	for i := 1; i <= 10; i++ {
		iters = append(iters, IterationResult{
			Iteration:  i,
			StartedAt:  time.Now(),
			FinishedAt: time.Now(),
			Output:     "out",
		})
	}
	m.runs = map[string]AgentRunSnapshot{
		"runner": {Name: "runner", Iterations: iters},
	}

	m.rebuildTree()

	loopNode := m.tree[0].Children[0]
	if len(loopNode.Children) != 4 {
		t.Fatalf("expected 4 iteration children (max), got %d", len(loopNode.Children))
	}

	// Most recent should be first
	if loopNode.Children[0].Iteration != 10 {
		t.Fatalf("expected most recent iteration 10, got %d", loopNode.Children[0].Iteration)
	}
	// Oldest shown should be 7
	if loopNode.Children[3].Iteration != 7 {
		t.Fatalf("expected oldest shown iteration 7, got %d", loopNode.Children[3].Iteration)
	}
}

func TestTUISearchFilter(t *testing.T) {
	m := NewTUIModel(nil)
	m.objects = []ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
		{Name: "tester", Definition: `(defagent "tester" (pipeline (step "test" (loop test))))`},
		{Name: "bugfixer", Definition: `(defagent "bugfixer" (pipeline (step "fix" (loop fix))))`},
	}
	m.runs = map[string]AgentRunSnapshot{}

	// No filter: all 3 agents
	m.searchQuery = ""
	m.rebuildTree()
	if len(m.tree) != 3 {
		t.Fatalf("expected 3 agents with no filter, got %d", len(m.tree))
	}

	// Filter for "build"
	m.searchQuery = "build"
	m.rebuildTree()
	if len(m.tree) != 1 {
		t.Fatalf("expected 1 agent matching 'build', got %d", len(m.tree))
	}
	if m.tree[0].AgentName != "builder" {
		t.Fatalf("expected 'builder', got %q", m.tree[0].AgentName)
	}

	// Filter for nonexistent
	m.searchQuery = "xyz"
	m.rebuildTree()
	if len(m.tree) != 0 {
		t.Fatalf("expected 0 agents matching 'xyz', got %d", len(m.tree))
	}
}

func TestTUIFlattenTreeExpandCollapse(t *testing.T) {
	m := NewTUIModel(nil)
	m.objects = []ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	m.runs = map[string]AgentRunSnapshot{
		"builder": {
			Name: "builder",
			Iterations: []IterationResult{
				{Iteration: 1, StartedAt: time.Now(), FinishedAt: time.Now()},
			},
		},
	}

	m.rebuildTree()

	// All expanded: agent + loop + 1 iteration = 3 flat nodes
	if len(m.flatTree) != 3 {
		t.Fatalf("expected 3 flat nodes when expanded, got %d", len(m.flatTree))
	}

	// Collapse agent
	m.tree[0].Expanded = false
	m.flattenTree()
	if len(m.flatTree) != 1 {
		t.Fatalf("expected 1 flat node when agent collapsed, got %d", len(m.flatTree))
	}

	// Expand agent but collapse loop
	m.tree[0].Expanded = true
	m.tree[0].Children[0].Expanded = false
	m.flattenTree()
	if len(m.flatTree) != 2 {
		t.Fatalf("expected 2 flat nodes when loop collapsed, got %d", len(m.flatTree))
	}
}

func TestMeanStddev(t *testing.T) {
	mean, stddev := meanStddev([]float64{10, 20, 30})
	if mean != 20 {
		t.Errorf("expected mean 20, got %f", mean)
	}
	// stddev of [10,20,30] = sqrt(((10-20)^2 + (20-20)^2 + (30-20)^2)/3) = sqrt(200/3) ≈ 8.165
	if stddev < 8.1 || stddev > 8.2 {
		t.Errorf("expected stddev ≈ 8.165, got %f", stddev)
	}

	// Single value: stddev = 0
	mean2, stddev2 := meanStddev([]float64{42})
	if mean2 != 42 || stddev2 != 0 {
		t.Errorf("single value: expected mean=42, stddev=0, got mean=%f, stddev=%f", mean2, stddev2)
	}

	// Empty: both 0
	mean3, stddev3 := meanStddev(nil)
	if mean3 != 0 || stddev3 != 0 {
		t.Errorf("empty: expected 0,0, got %f,%f", mean3, stddev3)
	}
}
