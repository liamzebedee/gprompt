package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"p2p/cluster"

	"github.com/stukennedy/tooey/app"
	"github.com/stukennedy/tooey/input"
	"github.com/stukennedy/tooey/node"
)

// --- Tree ---

func TestDeriveTreeBasic(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder", State: cluster.RunStateRunning,
			Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
		{Name: "tester", State: cluster.RunStatePending,
			Definition: `(defagent "tester" (pipeline (step "test" (loop test))))`},
	}
	runs := map[string]cluster.AgentRunSnapshot{
		"builder": {Name: "builder", StartedAt: time.Now(), Iterations: []cluster.IterationResult{
			{Iteration: 1, StartedAt: time.Now(), FinishedAt: time.Now()},
			{Iteration: 2, StartedAt: time.Now(), FinishedAt: time.Now()},
			{Iteration: 3, StartedAt: time.Now(), FinishedAt: time.Now()},
		}},
	}

	entries := deriveTree(objects, runs, nil, "", make(map[string]bool))
	if countKind(entries, NodeAgent) != 2 {
		t.Fatalf("expected 2 agents, got %d", countKind(entries, NodeAgent))
	}
	if entries[0].Agent != "builder" {
		t.Fatalf("expected first agent 'builder', got %q", entries[0].Agent)
	}
	if countKind(entries, NodeIteration) != 3 {
		t.Fatalf("expected 3 iterations, got %d", countKind(entries, NodeIteration))
	}
}

func TestDeriveTreeMaxIterations(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "runner", Definition: `(defagent "runner" (pipeline (step "run" (loop run))))`},
	}
	var iters []cluster.IterationResult
	for i := 1; i <= 10; i++ {
		iters = append(iters, cluster.IterationResult{Iteration: i, StartedAt: time.Now(), FinishedAt: time.Now()})
	}
	runs := map[string]cluster.AgentRunSnapshot{"runner": {Name: "runner", Iterations: iters}}
	entries := deriveTree(objects, runs, nil, "", make(map[string]bool))

	n := countKind(entries, NodeIteration)
	if n != 4 {
		t.Fatalf("expected 4 iterations (max), got %d", n)
	}
}

func TestDeriveTreeSearchFilter(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder"}, {Name: "tester"}, {Name: "bugfixer"},
	}
	entries := deriveTree(objects, nil, nil, "build", make(map[string]bool))
	if countKind(entries, NodeAgent) != 1 {
		t.Fatal("filter 'build' should match only builder")
	}
	entries = deriveTree(objects, nil, nil, "xyz", make(map[string]bool))
	if len(entries) != 0 {
		t.Fatalf("expected 0 for 'xyz', got %d", len(entries))
	}
}

func TestDeriveTreeExpandCollapse(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	runs := map[string]cluster.AgentRunSnapshot{
		"builder": {Name: "builder", Iterations: []cluster.IterationResult{
			{Iteration: 1, StartedAt: time.Now(), FinishedAt: time.Now()},
		}},
	}

	exp := make(map[string]bool)
	if len(deriveTree(objects, runs, nil, "", exp)) != 3 {
		t.Fatal("all expanded: expected 3")
	}
	exp[entryKey("builder", "")] = false
	if len(deriveTree(objects, runs, nil, "", exp)) != 1 {
		t.Fatal("agent collapsed: expected 1")
	}
}

func TestDeriveTreePipeline(t *testing.T) {
	objects := []cluster.ClusterObject{{Name: "planner", State: cluster.RunStateRunning}}
	pipelines := map[string]*cluster.PipelineDef{
		"planner": {Steps: []cluster.PipelineStep{
			{Label: "spec", Kind: cluster.StepKindSimple, Method: "write-spec"},
			{Label: "iterate", Kind: cluster.StepKindLoop, LoopMethod: "review"},
		}},
	}
	entries := deriveTree(objects, nil, pipelines, "", make(map[string]bool))
	if countKind(entries, NodeLoop) != 2 {
		t.Fatalf("expected 2 steps, got %d", countKind(entries, NodeLoop))
	}
}

func TestDeriveTreeLive(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	runs := map[string]cluster.AgentRunSnapshot{
		"builder": {Name: "builder",
			LiveIter:   &cluster.IterationResult{Iteration: 5, StartedAt: time.Now()},
			Iterations: []cluster.IterationResult{{Iteration: 4, StartedAt: time.Now(), FinishedAt: time.Now()}},
		},
	}
	entries := deriveTree(objects, runs, nil, "", make(map[string]bool))
	var found bool
	for _, e := range entries {
		if e.Live && e.Iter == 5 && strings.Contains(e.Label, "live") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected live entry for iteration 5")
	}
}

// --- View ---

func TestSidebarNoFlexOnText(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	entries := deriveTree(objects, nil, nil, "", make(map[string]bool))
	mdl := NewModel(nil)
	mdl.Objects = objects
	sidebar := renderSidebar(entries, 0, mdl, focusSidebar)

	var check func(n node.Node, path string)
	check = func(n node.Node, path string) {
		if n.Type == node.TextNode && n.Props.FlexWeight > 0 {
			t.Errorf("%s: TextNode has FlexWeight=%d", path, n.Props.FlexWeight)
		}
		for i, c := range n.Children {
			check(c, fmt.Sprintf("%s[%d]", path, i))
		}
	}
	check(sidebar, "sidebar")
}

func TestLoopViewTwoColumns(t *testing.T) {
	objects := []cluster.ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	runs := map[string]cluster.AgentRunSnapshot{
		"builder": {Name: "builder", Iterations: []cluster.IterationResult{
			{Iteration: 1, StartedAt: time.Now(), FinishedAt: time.Now()},
		}},
	}
	entries := deriveTree(objects, runs, nil, "", make(map[string]bool))
	mdl := NewModel(nil)
	mdl.Objects = objects
	mdl.Runs = runs
	mdl.Methods = map[string]map[string]string{"builder": {"build": "do work"}}
	mdl.Ready = true

	// Loop entry is at index 1
	content := buildLoopContent(entries[1], mdl)
	// Should contain a Row node (prompt | stats)
	var hasRow bool
	for _, n := range content {
		if n.Type == node.RowNode {
			hasRow = true
			if len(n.Children) != 2 {
				t.Errorf("expected 2 columns in row, got %d", len(n.Children))
			}
		}
	}
	if !hasRow {
		t.Error("loop view should have a Row with prompt and stats columns")
	}
}

func TestTwoFocusableRegions(t *testing.T) {
	mdl := NewModel(nil)
	mdl.Ready = true
	mdl.Started = true
	mdl.Objects = []cluster.ClusterObject{
		{Name: "builder", Definition: `(defagent "builder" (pipeline (step "build" (loop build))))`},
	}
	mdl.Runs = map[string]cluster.AgentRunSnapshot{
		"builder": {Name: "builder", Iterations: []cluster.IterationResult{
			{Iteration: 1, StartedAt: time.Now(), FinishedAt: time.Now(),
				Messages: []cluster.ConvoMessage{{ID: "m1", Type: "text", Content: "done"}}},
		}},
	}
	mdl.Cursor = 2 // iteration

	tree := tuiView(mdl, focusSidebar)
	var focusable []string
	var find func(n node.Node)
	find = func(n node.Node) {
		if n.Props.Focusable && n.Props.Key != "" {
			focusable = append(focusable, n.Props.Key)
		}
		for _, c := range n.Children {
			find(c)
		}
	}
	find(tree)

	if len(focusable) != 2 {
		t.Fatalf("expected 2 focusable regions, got %d: %v", len(focusable), focusable)
	}
	expected := map[string]bool{focusSidebar: true, focusInput: true}
	for _, k := range focusable {
		delete(expected, k)
	}
	if len(expected) > 0 {
		t.Errorf("missing focusable keys: %v", expected)
	}
}

// --- Update ---

func TestUpdateStateMsg(t *testing.T) {
	mdl := NewModel(nil)
	mdl.Started = true
	result := tuiUpdate(mdl, stateMsg(cluster.SteerStatePayload{
		Objects: []cluster.ClusterObject{{Name: "builder"}},
		Runs:    map[string]cluster.AgentRunSnapshot{"builder": {Name: "builder"}},
	}))
	m := result.Model.(*Model)
	if !m.Ready {
		t.Error("should be ready")
	}
}

func TestUpdateErrAndReconnect(t *testing.T) {
	mdl := NewModel(nil)
	mdl.Started = true
	result := tuiUpdate(mdl, errMsg{err: fmt.Errorf("disconnected")})
	m := result.Model.(*Model)
	if m.ErrText != "disconnected" {
		t.Errorf("got %q", m.ErrText)
	}
	result = tuiUpdate(m, reconnectMsg{})
	m = result.Model.(*Model)
	if m.ErrText != "" {
		t.Errorf("should be cleared, got %q", m.ErrText)
	}
}

func TestSidebarNav(t *testing.T) {
	mdl := NewModel(nil)
	mdl.Started = true
	mdl.Ready = true
	mdl.Focused = focusSidebar
	mdl.Objects = []cluster.ClusterObject{
		{Name: "a", Definition: `(defagent "a" (pipeline (step "s" (loop s))))`},
		{Name: "b", Definition: `(defagent "b" (pipeline (step "s" (loop s))))`},
	}

	r := tuiUpdate(mdl, app.KeyMsg{Key: input.Key{Type: input.Down}})
	if r.Model.(*Model).Cursor != 1 {
		t.Error("down should move cursor to 1")
	}
	r = tuiUpdate(r.Model, app.KeyMsg{Key: input.Key{Type: input.Up}})
	if r.Model.(*Model).Cursor != 0 {
		t.Error("up should move cursor to 0")
	}
}

func TestQuit(t *testing.T) {
	for _, pane := range []string{focusSidebar, ""} {
		mdl := NewModel(nil)
		mdl.Started = true
		mdl.Focused = pane
		r := tuiUpdate(mdl, app.KeyMsg{Key: input.Key{Type: input.RuneKey, Rune: 'q'}})
		if r.Model != nil {
			t.Errorf("pane=%q: q should quit", pane)
		}
	}
}

// --- Helpers ---

func TestHelpers(t *testing.T) {
	if clamp(5, 0, 10) != 5 || clamp(-1, 0, 10) != 0 || clamp(15, 0, 10) != 10 {
		t.Error("clamp")
	}
	if extractLoopMethod(`(defagent "b" (pipeline (step "build" (loop build))))`) != "build" {
		t.Error("extractLoopMethod")
	}
	if extractLoopMethod("") != "" {
		t.Error("extractLoopMethod empty")
	}
	m, s := meanStddev([]float64{10, 20, 30})
	if m != 20 || s < 8.1 || s > 8.2 {
		t.Errorf("meanStddev: %f, %f", m, s)
	}
	if expandKey(Entry{Kind: NodeLoop, Agent: "a", Step: "b"}) != "a/b" {
		t.Error("expandKey loop")
	}
	if expandKey(Entry{Kind: NodeAgent, Agent: "a"}) != "a/" {
		t.Error("expandKey agent")
	}

	scroll := 0
	ensureVisible(&scroll, 15, 10)
	if scroll != 6 {
		t.Errorf("ensureVisible: got %d", scroll)
	}
}

func countKind(entries []Entry, kind NodeKind) int {
	n := 0
	for _, e := range entries {
		if e.Kind == kind {
			n++
		}
	}
	return n
}
