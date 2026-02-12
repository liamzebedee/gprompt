package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"p2p/cluster"

	"github.com/stukennedy/tooey/node"
)

// renderToText extracts all text from a node tree, line by line.
// This is what the user actually sees in the terminal.
func renderToText(nodes []node.Node) string {
	var lines []string
	for _, n := range nodes {
		lines = append(lines, nodeText(n))
	}
	return strings.Join(lines, "\n")
}

func nodeText(n node.Node) string {
	if n.Type == node.TextNode {
		return n.Props.Text
	}
	var parts []string
	for _, c := range n.Children {
		parts = append(parts, nodeText(c))
	}
	return strings.Join(parts, "")
}

// Realistic message sequences sampled from actual Claude streaming output.
// Each test case simulates what CallClaudeStreaming produces.

func TestRender_SimpleTextOnly(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "I'll read the backlog "},
		{ID: "msg-2", Type: "text", Content: "and pick an item to work on."},
	}

	out := renderMessages(t, "builder", 3, msgs, false)
	t.Logf("=== SimpleTextOnly ===\n%s", out)

	// Fragments should be assembled into one block with bullet
	if !strings.Contains(out, "● I'll read the backlog and pick an item to work on.") {
		t.Error("text fragments should be assembled with ● bullet")
	}
}

func TestRender_TextThenToolThenText(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "Let me read the backlog first."},
		{ID: "msg-2", Type: "tool_use", Content: "Read", Detail: "BACKLOG.md"},
		{ID: "msg-3", Type: "tool_result", Content: "BACKLOG.md contents here..."},
		{ID: "msg-4", Type: "text", Content: "I'll pick item #3 "},
		{ID: "msg-5", Type: "text", Content: "and implement it."},
	}

	out := renderMessages(t, "builder", 3, msgs, false)
	t.Logf("=== TextThenToolThenText ===\n%s", out)

	if !strings.Contains(out, "● Let me read the backlog first.") {
		t.Error("first text block missing bullet")
	}
	if !strings.Contains(out, "● Read(BACKLOG.md)") {
		t.Error("tool use should show name with detail")
	}
	if !strings.Contains(out, "⎿") {
		t.Error("tool result missing")
	}
	if !strings.Contains(out, "● I'll pick item #3 and implement it.") {
		t.Error("second text block should be assembled with bullet")
	}
}

func TestRender_MultipleTools(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "Now let me run the next iteration.\n"},
		{ID: "msg-2", Type: "tool_use", Content: "Explore", Detail: "codebase structure"},
		{ID: "msg-3", Type: "tool_result", Content: "Done (3 tool uses)"},
		{ID: "msg-4", Type: "text", Content: "Pick item, implement, commit.\n"},
		{ID: "msg-5", Type: "tool_use", Content: "Task", Detail: "Fix failing tests"},
		{ID: "msg-6", Type: "tool_result", Content: "Done (0 tool uses)"},
		{ID: "msg-7", Type: "tool_use", Content: "Write", Detail: "src/feature.go"},
		{ID: "msg-8", Type: "tool_result", Content: "Wrote 41 lines to src/feature.go"},
		{ID: "msg-9", Type: "text", Content: "Done. Summary: shipped one change, tests green."},
	}

	out := renderMessages(t, "builder", 3, msgs, false)
	t.Logf("=== MultipleTools (spec example) ===\n%s", out)

	if !strings.Contains(out, "● Explore(codebase structure)") {
		t.Error("missing Explore tool with detail")
	}
	if !strings.Contains(out, "● Task(Fix failing tests)") {
		t.Error("missing Task tool with detail")
	}
	if !strings.Contains(out, "● Write(src/feature.go)") {
		t.Error("missing Write tool with detail")
	}
	if !strings.Contains(out, "⎿  Wrote 41 lines") {
		t.Error("missing Write result")
	}
}

func TestRender_StreamingFragments(t *testing.T) {
	// Simulates real streaming: many tiny text deltas
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "I"},
		{ID: "msg-2", Type: "text", Content: "'ll"},
		{ID: "msg-3", Type: "text", Content: " start"},
		{ID: "msg-4", Type: "text", Content: " by"},
		{ID: "msg-5", Type: "text", Content: " reading"},
		{ID: "msg-6", Type: "text", Content: " the"},
		{ID: "msg-7", Type: "text", Content: " code"},
		{ID: "msg-8", Type: "text", Content: "base.\n\n"},
		{ID: "msg-9", Type: "text", Content: "Then I'll make changes."},
	}

	out := renderMessages(t, "builder", 1, msgs, false)
	t.Logf("=== StreamingFragments ===\n%s", out)

	if !strings.Contains(out, "● I'll start by reading the codebase.") {
		t.Error("fragments should assemble with bullet")
	}
	if !strings.Contains(out, "Then I'll make changes.") {
		t.Error("continuation lines should be present")
	}
}

func TestRender_LiveWithSpinner(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "Working on it"},
		{ID: "msg-2", Type: "tool_use", Content: "Bash"},
	}

	out := renderMessages(t, "builder", 2, msgs, true)
	t.Logf("=== LiveWithSpinner ===\n%s", out)

	if !strings.Contains(out, "⠋") {
		t.Error("should have spinner when live")
	}
	if !strings.Contains(out, "Thinking") {
		t.Error("should show Thinking text")
	}
}

func TestRender_ToolUseNoResult(t *testing.T) {
	// Tool started but no result yet (mid-execution)
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "Let me check.\n"},
		{ID: "msg-2", Type: "tool_use", Content: "Bash"},
	}

	out := renderMessages(t, "builder", 1, msgs, false)
	t.Logf("=== ToolUseNoResult ===\n%s", out)

	if !strings.Contains(out, "● Bash") {
		t.Error("tool use should show even without result")
	}
}

func TestRender_LongToolResult(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "tool_use", Content: "Read"},
		{ID: "msg-2", Type: "tool_result", Content: strings.Repeat("x", 300)},
	}

	out := renderMessages(t, "builder", 1, msgs, false)
	t.Logf("=== LongToolResult ===\n%s", out)

	if !strings.Contains(out, "…") {
		t.Error("long tool result should be truncated")
	}
}

func TestRender_EmptyMessages(t *testing.T) {
	out := renderMessages(t, "builder", 1, nil, false)
	t.Logf("=== EmptyMessages ===\n%s", out)

	// Should still render header
	if !strings.Contains(out, "builder") {
		t.Error("header should be present")
	}
}

// Simulates exactly what the screenshot shows — real streaming with no tool_results.
func TestRender_RealStreaming(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		// First text delta might start with newline (causes lonely bullet)
		{ID: "msg-1", Type: "text", Content: "\n"},
		{ID: "msg-2", Type: "text", Content: "I'll start by reading the BACKLOG.md file "},
		{ID: "msg-3", Type: "text", Content: "to see what items are available.\n"},
		// Tool use — no tool_result comes from streaming
		{ID: "msg-4", Type: "tool_use", Content: "Read", Detail: "BACKLOG.md"},
		// Text after tool
		{ID: "msg-5", Type: "text", Content: "There's one pending item: **Add `todo archive` command**."},
		{ID: "msg-6", Type: "text", Content: " Let me explore the existing structure.\n"},
		// Multiple tools in a row
		{ID: "msg-7", Type: "tool_use", Content: "Glob", Detail: "**/*.go"},
		{ID: "msg-8", Type: "tool_use", Content: "Bash", Detail: "go test ./... 2>&1"},
		{ID: "msg-9", Type: "tool_use", Content: "Read", Detail: "todo.go"},
		{ID: "msg-10", Type: "tool_use", Content: "Read", Detail: "cmd/todo/main.go"},
		{ID: "msg-11", Type: "tool_use", Content: "Read", Detail: "go.mod"},
		// Text after tools
		{ID: "msg-12", Type: "text", Content: "Now I have a thorough understanding of the codebase."},
		{ID: "msg-13", Type: "text", Content: " Let me implement the `todo archive` command.\n"},
		// Final tool (still running)
		{ID: "msg-14", Type: "tool_use", Content: "Write", Detail: "todo.go"},
	}

	out := renderMessages(t, "builder", 1, msgs, true)
	fmt.Println()
	fmt.Println("┌─ REAL STREAMING OUTPUT ─────────────────────────────────────────────────────────┐")
	for _, line := range strings.Split(out, "\n") {
		fmt.Printf("│ %-80s│\n", line)
	}
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")

	// No lonely bullet
	if strings.Contains(out, "● \n") || strings.Contains(out, "●\n") {
		t.Error("should not have lonely bullet on empty text")
	}
	// Spinner present
	if !strings.Contains(out, "Thinking") {
		t.Error("should show spinner when live")
	}
}

func TestRender_LeadingWhitespaceText(t *testing.T) {
	// Edge case: text that's just whitespace/newlines
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "\n\n"},
		{ID: "msg-2", Type: "tool_use", Content: "Read"},
		{ID: "msg-3", Type: "text", Content: "Got it."},
	}

	out := renderMessages(t, "builder", 1, msgs, false)
	t.Logf("=== LeadingWhitespaceText ===\n%s", out)

	// Should NOT have a bullet for the empty text
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "●" {
			t.Error("should not have lonely ● bullet")
		}
	}
}

func TestRender_TextWithNewlines(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		{ID: "msg-1", Type: "text", Content: "Line one.\nLine two.\nLine three."},
	}

	out := renderMessages(t, "builder", 1, msgs, false)
	t.Logf("=== TextWithNewlines ===\n%s", out)

	if !strings.Contains(out, "Line one.") || !strings.Contains(out, "Line two.") || !strings.Contains(out, "Line three.") {
		t.Error("multi-line text should preserve line breaks")
	}
}

// --- Helper ---

func renderMessages(t *testing.T, agent string, iter int, msgs []cluster.ConvoMessage, live bool) string {
	t.Helper()
	entry := Entry{Kind: NodeIteration, Agent: agent, Step: "build", Iter: iter, Live: live}
	mdl := NewModel(nil)
	mdl.SpinFrame = 0

	iterResult := cluster.IterationResult{
		Iteration: iter,
		StartedAt: time.Now().Add(-10 * time.Second),
		Messages:  msgs,
	}
	if !live {
		iterResult.FinishedAt = time.Now()
	}

	if live {
		mdl.Runs = map[string]cluster.AgentRunSnapshot{
			agent: {Name: agent, LiveIter: &iterResult},
		}
	} else {
		mdl.Runs = map[string]cluster.AgentRunSnapshot{
			agent: {Name: agent, Iterations: []cluster.IterationResult{iterResult}},
		}
	}

	nodes := renderIteration(entry, mdl)
	return renderToText(nodes)
}

// TestRender_FullSpecExample renders the exact scenario from the spec mockup
// and prints it so we can visually compare.
func TestRender_FullSpecExample(t *testing.T) {
	msgs := []cluster.ConvoMessage{
		// Text block
		{ID: "msg-1", Type: "text", Content: "Now let me run the next iteration.\n"},
		// Tool: Explore
		{ID: "msg-2", Type: "tool_use", Content: "Explore", Detail: "Read BACKLOG.md"},
		{ID: "msg-3", Type: "tool_result", Content: "Done (3 tool uses · 8.2k tokens · 22s)"},
		// Text block
		{ID: "msg-4", Type: "text", Content: "Pick item, implement, commit.\n"},
		// Tool: Task
		{ID: "msg-5", Type: "tool_use", Content: "Task", Detail: "Fix failing tests"},
		{ID: "msg-6", Type: "tool_result", Content: "Done (0 tool uses · 4.1k tokens · 12s)"},
		// Tool: Write
		{ID: "msg-7", Type: "tool_use", Content: "Write", Detail: "src/feature.go"},
		{ID: "msg-8", Type: "tool_result", Content: "Wrote 41 lines to src/feature.go"},
		// Final text
		{ID: "msg-9", Type: "text", Content: "Done. Summary: shipped one change, tests green."},
	}

	out := renderMessages(t, "builder", 3, msgs, false)

	// Print with a box to simulate the pane
	fmt.Println()
	fmt.Println("┌─ SPEC TARGET ───────────────────────────────────────────┐")
	fmt.Println("│ ● Now let me run the next iteration.                   │")
	fmt.Println("│                                                        │")
	fmt.Println("│ ● Explore(Read BACKLOG.md)                             │")
	fmt.Println("│   ⎿  Done (3 tool uses · 8.2k tokens · 22s)          │")
	fmt.Println("│                                                        │")
	fmt.Println("│ ● Pick item, implement, commit.                        │")
	fmt.Println("│                                                        │")
	fmt.Println("│ ● Task(Fix failing tests)                              │")
	fmt.Println("│   ⎿  Done (0 tool uses · 4.1k tokens · 12s)          │")
	fmt.Println("│                                                        │")
	fmt.Println("│ ● Write(src/feature.go)                                │")
	fmt.Println("│   ⎿  Wrote 41 lines to src/feature.go                │")
	fmt.Println("│                                                        │")
	fmt.Println("│ ● Done. Summary: shipped one change, tests green.      │")
	fmt.Println("└────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("┌─ ACTUAL OUTPUT ────────────────────────────────────────┐")
	for _, line := range strings.Split(out, "\n") {
		fmt.Printf("│ %-55s│\n", line)
	}
	fmt.Println("└────────────────────────────────────────────────────────┘")
	fmt.Println()
}
