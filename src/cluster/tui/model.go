// Package tui implements the gcluster steer TUI.
//
// Two panes: tree sidebar (left) + detail view (right).
// Shift+Tab swaps focus between sidebar and input.
package tui

import (
	"math"
	"strings"

	"p2p/cluster"

	"github.com/stukennedy/tooey/component"
)

// NodeKind identifies what a sidebar entry represents.
type NodeKind int

const (
	NodeAgent     NodeKind = iota
	NodeLoop
	NodeIteration
)

// Entry is one row in the sidebar tree. Derived fresh each render.
type Entry struct {
	Kind        NodeKind
	Label       string
	Agent       string
	Step        string
	Iter        int
	Depth       int
	HasChildren bool
	Expanded    bool
	Live        bool
}

// Messages from subscriptions.
type stateMsg cluster.SteerStatePayload
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type reconnectMsg struct{}
type tickMsg struct{}

const (
	focusSidebar = "sidebar"
	focusContent = "content"
	focusInput   = "input"
)

// Model is the TUI state.
type Model struct {
	Client *cluster.SteerClient

	// Domain data (replaced on each server push)
	Objects   []cluster.ClusterObject
	Runs      map[string]cluster.AgentRunSnapshot
	Methods   map[string]map[string]string
	Pipelines map[string]*cluster.PipelineDef

	// Sidebar
	Cursor        int
	SidebarScroll int
	Search        string
	Expanded      map[string]bool

	// Content scroll (offset = lines from bottom in ScrollToBottom mode)
	Scroll int
	Tail   bool

	// Inputs
	SearchInput component.TextInput
	MsgInput    component.TextInput
	PromptInput component.TextInput

	// Focus + status
	Focused   string
	ErrText   string
	Ready     bool
	Started   bool
	SpinFrame int
}

// NewModel creates the initial TUI model.
func NewModel(client *cluster.SteerClient) *Model {
	return &Model{
		Client:      client,
		Tail:        true,
		Runs:        make(map[string]cluster.AgentRunSnapshot),
		Methods:     make(map[string]map[string]string),
		Pipelines:   make(map[string]*cluster.PipelineDef),
		Expanded:    make(map[string]bool),
		SearchInput: component.NewTextInput("search agents..."),
		MsgInput:    component.NewTextInput("send message…"),
		PromptInput: component.NewTextInput("edit prompt…"),
	}
}

// --- Helpers ---

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func entryKey(agent, step string) string { return agent + "/" + step }

func expandKey(e Entry) string {
	if e.Kind == NodeLoop {
		return entryKey(e.Agent, e.Step)
	}
	return entryKey(e.Agent, "")
}

func isExpanded(expanded map[string]bool, key string) bool {
	v, ok := expanded[key]
	return !ok || v
}

func ensureVisible(scroll *int, idx, visibleH int) {
	if idx < *scroll {
		*scroll = idx
	}
	if idx >= *scroll+visibleH {
		*scroll = idx - visibleH + 1
	}
	if *scroll < 0 {
		*scroll = 0
	}
}

func extractLoopMethod(def string) string {
	idx := strings.Index(def, "(loop ")
	if idx < 0 {
		return ""
	}
	rest := def[idx+6:]
	if end := strings.IndexAny(rest, ") "); end >= 0 {
		return rest[:end]
	}
	return rest
}

func stateLabel(s cluster.RunState) string {
	switch s {
	case cluster.RunStatePending:
		return "pending"
	case cluster.RunStateRunning:
		return "running"
	case cluster.RunStateStopped:
		return "stopped"
	default:
		return string(s)
	}
}

func meanStddev(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	if len(values) == 1 {
		return mean, 0
	}
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}
