// Package cluster tui implements the steer TUI using bubbletea.
//
// The TUI presents a two-pane layout: a tree sidebar on the left showing
// agents and their loop iterations, and a detail view on the right that
// renders content based on the selected node type:
//
//   - AgentView: reserved for future metadata
//   - LoopView: method body text + iteration statistics
//   - IterationView: chat message history + input box for steering
//
// The tree is built from SteerStatePayload which contains both declarative
// state (ClusterObject) and runtime iteration data (AgentRunSnapshot).
// State updates arrive through a channel from the SteerClient and are
// consumed via a bubbletea Cmd subscription.
//
// Navigation: up/down moves highlight, left/right collapse/expand,
// Shift+Tab swaps focus between tree and input box.
package cluster

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Tree node types ---

// NodeKind identifies what a tree node represents.
type NodeKind int

const (
	NodeAgent     NodeKind = iota // Top-level agent
	NodeLoop                      // loop(method) step
	NodeIteration                 // Single iteration under a loop
)

// TreeNode is a single node in the sidebar tree.
type TreeNode struct {
	Kind      NodeKind
	Label     string // Display label
	AgentName string // Agent this node belongs to
	StepLabel string // For loop/iteration nodes: the method name
	Iteration int    // For iteration nodes: 1-based iteration number

	Expanded bool // Whether children are visible
	Children []*TreeNode
	Depth    int // Nesting level for indentation
}

// --- Bubbletea messages ---

// stateMsg carries a state update from the SteerClient.
type stateMsg SteerStatePayload

// errMsg carries a connection error.
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

// reconnectMsg signals that the client reconnected to the master.
type reconnectMsg struct{}

// --- TUI Model ---

// TUIModel is the bubbletea model for gcluster steer.
type TUIModel struct {
	client *SteerClient

	// State from master
	objects []ClusterObject
	runs    map[string]AgentRunSnapshot

	// Tree
	tree     []*TreeNode // root nodes (agents)
	flatTree []*TreeNode // flattened visible nodes for navigation
	cursor   int         // index into flatTree

	// Search
	searchInput textinput.Model
	searchQuery string

	// Message input (for injection in iteration view)
	msgInput    textinput.Model
	inputFocused bool // true = focus on input, false = focus on tree

	// Terminal dimensions
	width  int
	height int

	// Error banner
	errText string

	// Ready: received first state
	ready bool
}

// NewTUIModel creates the initial TUI model.
func NewTUIModel(client *SteerClient) TUIModel {
	si := textinput.New()
	si.Placeholder = "Search agents..."
	si.CharLimit = 100

	mi := textinput.New()
	mi.Placeholder = "send message…"
	mi.CharLimit = 4096

	return TUIModel{
		client:      client,
		searchInput: si,
		msgInput:    mi,
		runs:        make(map[string]AgentRunSnapshot),
	}
}

// Init is called once when the program starts.
func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(
		m.waitForState(),
		m.waitForError(),
		m.waitForReconnect(),
	)
}

// waitForState returns a Cmd that waits for the next state update.
func (m TUIModel) waitForState() tea.Cmd {
	return func() tea.Msg {
		payload, ok := <-m.client.StateCh
		if !ok {
			return errMsg{err: fmt.Errorf("state channel closed")}
		}
		return stateMsg(payload)
	}
}

// waitForError returns a Cmd that waits for a connection error.
func (m TUIModel) waitForError() tea.Cmd {
	return func() tea.Msg {
		err, ok := <-m.client.ErrCh
		if !ok {
			return nil
		}
		return errMsg{err: err}
	}
}

// waitForReconnect returns a Cmd that waits for a successful reconnection.
func (m TUIModel) waitForReconnect() tea.Cmd {
	return func() tea.Msg {
		_, ok := <-m.client.ReconnectCh
		if !ok {
			return nil
		}
		return reconnectMsg{}
	}
}

// Update handles messages.
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stateMsg:
		payload := SteerStatePayload(msg)
		m.objects = payload.Objects
		if payload.Runs != nil {
			m.runs = payload.Runs
		}
		m.ready = true
		m.rebuildTree()
		return m, m.waitForState()

	case errMsg:
		m.errText = msg.Error()
		return m, m.waitForError()

	case reconnectMsg:
		m.errText = ""
		return m, m.waitForReconnect()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Pass through to focused input
	if m.inputFocused {
		var cmd tea.Cmd
		if m.selectedNodeKind() == NodeIteration {
			m.msgInput, cmd = m.msgInput.Update(msg)
		} else {
			m.searchInput, cmd = m.searchInput.Update(msg)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m TUIModel) selectedNodeKind() NodeKind {
	if m.cursor >= 0 && m.cursor < len(m.flatTree) {
		return m.flatTree[m.cursor].Kind
	}
	return NodeAgent
}

func (m *TUIModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c", "q":
		if !m.inputFocused {
			return m, tea.Quit
		}
	case "shift+tab":
		m.inputFocused = !m.inputFocused
		if m.inputFocused {
			if m.selectedNodeKind() == NodeIteration {
				m.msgInput.Focus()
				m.searchInput.Blur()
			} else {
				m.searchInput.Focus()
				m.msgInput.Blur()
			}
		} else {
			m.msgInput.Blur()
			m.searchInput.Blur()
		}
		return m, nil
	}

	// Input-focused keys
	if m.inputFocused {
		switch key {
		case "enter":
			if m.selectedNodeKind() == NodeIteration && m.msgInput.Value() != "" {
				node := m.flatTree[m.cursor]
				msg := m.msgInput.Value()
				m.msgInput.Reset()
				if err := m.client.Inject(node.AgentName, node.StepLabel, node.Iteration, msg); err != nil {
					m.errText = fmt.Sprintf("inject error: %v", err)
				}
				return m, nil
			}
			// Search input: just apply filter
			m.searchQuery = m.searchInput.Value()
			m.rebuildTree()
			return m, nil
		case "esc":
			m.inputFocused = false
			m.msgInput.Blur()
			m.searchInput.Blur()
			return m, nil
		}
		// Let input handle the key
		var cmd tea.Cmd
		if m.selectedNodeKind() == NodeIteration {
			m.msgInput, cmd = m.msgInput.Update(msg)
		} else {
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.searchQuery = m.searchInput.Value()
			m.rebuildTree()
		}
		return m, cmd
	}

	// Tree navigation keys
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.flatTree)-1 {
			m.cursor++
		}
	case "right", "l":
		if m.cursor >= 0 && m.cursor < len(m.flatTree) {
			node := m.flatTree[m.cursor]
			if len(node.Children) > 0 && !node.Expanded {
				node.Expanded = true
				m.flattenTree()
			}
		}
	case "left", "h":
		if m.cursor >= 0 && m.cursor < len(m.flatTree) {
			node := m.flatTree[m.cursor]
			if node.Expanded {
				node.Expanded = false
				m.flattenTree()
			}
		}
	case "/":
		m.inputFocused = true
		m.searchInput.Focus()
		return m, textinput.Blink
	case "enter":
		// Toggle expand/collapse
		if m.cursor >= 0 && m.cursor < len(m.flatTree) {
			node := m.flatTree[m.cursor]
			if len(node.Children) > 0 {
				node.Expanded = !node.Expanded
				m.flattenTree()
			}
		}
	}

	return m, nil
}

// rebuildTree reconstructs the tree from current state.
func (m *TUIModel) rebuildTree() {
	// Sort agents by name
	sort.Slice(m.objects, func(i, j int) bool {
		return m.objects[i].Name < m.objects[j].Name
	})

	var nodes []*TreeNode
	for _, obj := range m.objects {
		// Search filter
		if m.searchQuery != "" && !strings.Contains(
			strings.ToLower(obj.Name),
			strings.ToLower(m.searchQuery),
		) {
			continue
		}

		agentNode := &TreeNode{
			Kind:      NodeAgent,
			Label:     obj.Name,
			AgentName: obj.Name,
			Expanded:  true, // agents expanded by default
			Depth:     0,
		}

		// Find the loop method for this agent from run data
		run, hasRun := m.runs[obj.Name]

		// Determine loop step label from the agent definition.
		// For a simple loop(method) agent, we extract the method name.
		stepLabel := extractLoopMethod(obj.Definition)
		if stepLabel == "" {
			stepLabel = "loop"
		}

		loopNode := &TreeNode{
			Kind:      NodeLoop,
			Label:     fmt.Sprintf("loop(%s)", stepLabel),
			AgentName: obj.Name,
			StepLabel: stepLabel,
			Expanded:  true,
			Depth:     1,
		}

		// Add iteration children (most recent first, max 4)
		if hasRun && len(run.Iterations) > 0 {
			iters := run.Iterations
			start := 0
			if len(iters) > 4 {
				start = len(iters) - 4
			}
			for i := len(iters) - 1; i >= start; i-- {
				ir := iters[i]
				label := fmt.Sprintf("iteration %d", ir.Iteration)
				iterNode := &TreeNode{
					Kind:      NodeIteration,
					Label:     label,
					AgentName: obj.Name,
					StepLabel: stepLabel,
					Iteration: ir.Iteration,
					Depth:     2,
				}
				loopNode.Children = append(loopNode.Children, iterNode)
			}
		}

		agentNode.Children = []*TreeNode{loopNode}
		nodes = append(nodes, agentNode)
	}

	m.tree = nodes
	m.flattenTree()
}

// flattenTree builds the flatTree from expanded nodes.
func (m *TUIModel) flattenTree() {
	var flat []*TreeNode
	var walk func(nodes []*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, n := range nodes {
			flat = append(flat, n)
			if n.Expanded {
				walk(n.Children)
			}
		}
	}
	walk(m.tree)
	m.flatTree = flat

	// Clamp cursor
	if m.cursor >= len(m.flatTree) {
		m.cursor = len(m.flatTree) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// extractLoopMethod parses the S-expression to find the loop method name.
// For (defagent "name" (pipeline (step "label" (loop method)))), extracts "method".
func extractLoopMethod(definition string) string {
	// Simple substring search for (loop ...)
	idx := strings.Index(definition, "(loop ")
	if idx < 0 {
		return ""
	}
	rest := definition[idx+6:]
	end := strings.IndexAny(rest, ") ")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// View renders the TUI.
func (m TUIModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	if !m.ready {
		return m.centeredMessage("Connecting to master...")
	}

	if m.errText != "" && !m.ready {
		return m.centeredMessage(fmt.Sprintf("Error: %s", m.errText))
	}

	// Layout: left pane (40%) | right pane (60%)
	leftWidth := m.width * 2 / 5
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := m.width - leftWidth - 3 // -3 for border/separator

	// Content height: total - 2 for top/bottom borders - 1 for error banner
	contentHeight := m.height - 2
	if m.errText != "" {
		contentHeight -= 1
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	leftContent := m.renderTree(leftWidth, contentHeight)
	rightContent := m.renderDetail(rightWidth, contentHeight)

	// Style panes
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight)

	left := leftStyle.Render(leftContent)
	right := rightStyle.Render(rightContent)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Error banner at bottom
	if m.errText != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		main = lipgloss.JoinVertical(lipgloss.Left, main, errStyle.Render("⚠ "+m.errText))
	}

	return main
}

// renderTree renders the left sidebar.
func (m TUIModel) renderTree(width, height int) string {
	var sb strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true)
	sb.WriteString(headerStyle.Render("Agents"))
	sb.WriteString("\n")

	// Separator
	sb.WriteString(strings.Repeat("─", width-2))
	sb.WriteString("\n")

	// Search input
	m.searchInput.Width = width - 4
	sb.WriteString("[ ")
	sb.WriteString(m.searchInput.View())
	sb.WriteString(" ]")
	sb.WriteString("\n")

	if len(m.objects) == 0 {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
			"No agents.\nRun `gcluster apply <file.p>`\nto add agents."))
		return sb.String()
	}

	if len(m.flatTree) == 0 {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
			"No matching agents."))
		return sb.String()
	}

	// Tree nodes
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	boldStyle := lipgloss.NewStyle().Bold(true)

	// Calculate visible window (scroll if needed)
	treeHeight := height - 3 // header + separator + search
	if treeHeight < 1 {
		treeHeight = 1
	}

	startIdx := 0
	if m.cursor >= treeHeight {
		startIdx = m.cursor - treeHeight + 1
	}
	endIdx := startIdx + treeHeight
	if endIdx > len(m.flatTree) {
		endIdx = len(m.flatTree)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.flatTree[i]
		indent := strings.Repeat("  ", node.Depth)

		// Icon
		var icon string
		switch node.Kind {
		case NodeAgent:
			if node.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		case NodeLoop:
			if node.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		case NodeIteration:
			icon = "  "
		}

		label := node.Label

		// Most recent iteration is bold
		if node.Kind == NodeIteration {
			run, ok := m.runs[node.AgentName]
			if ok && len(run.Iterations) > 0 {
				latest := run.Iterations[len(run.Iterations)-1]
				if node.Iteration == latest.Iteration {
					label = boldStyle.Render(label)
				} else {
					label = dimStyle.Render(label)
				}
			}
		}

		line := indent + icon + label

		// Truncate to width
		if lipgloss.Width(line) > width-2 {
			line = line[:width-4] + "…"
		}

		if i == m.cursor {
			// Pad to fill width for selection highlight
			pad := width - 2 - lipgloss.Width(line)
			if pad > 0 {
				line = line + strings.Repeat(" ", pad)
			}
			sb.WriteString(selectedStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderDetail renders the right pane based on selected node.
func (m TUIModel) renderDetail(width, height int) string {
	if len(m.flatTree) == 0 || m.cursor < 0 || m.cursor >= len(m.flatTree) {
		return ""
	}

	node := m.flatTree[m.cursor]

	switch node.Kind {
	case NodeAgent:
		return m.renderAgentView(node, width, height)
	case NodeLoop:
		return m.renderLoopView(node, width, height)
	case NodeIteration:
		return m.renderIterationView(node, width, height)
	}

	return ""
}

// renderAgentView renders the detail for an agent node.
func (m TUIModel) renderAgentView(node *TreeNode, width, height int) string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	sb.WriteString(headerStyle.Render(node.AgentName))
	sb.WriteString("\n\n")

	// Find the object
	var obj *ClusterObject
	for i := range m.objects {
		if m.objects[i].Name == node.AgentName {
			obj = &m.objects[i]
			break
		}
	}
	if obj == nil {
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("State: %s\n", stateLabel(obj.State)))
	sb.WriteString(fmt.Sprintf("Revisions: %d\n", len(obj.Revisions)))

	if run, ok := m.runs[node.AgentName]; ok {
		sb.WriteString(fmt.Sprintf("Running since: %s\n", run.StartedAt.Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("Iterations: %d\n", len(run.Iterations)))
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("Select a loop or iteration for details."))

	return sb.String()
}

func stateLabel(s RunState) string {
	switch s {
	case RunStatePending:
		return "⏳ pending"
	case RunStateRunning:
		return "🟢 running"
	case RunStateStopped:
		return "⏹ stopped"
	default:
		return string(s)
	}
}

// renderLoopView renders the detail for a loop node: prompt + stats.
func (m TUIModel) renderLoopView(node *TreeNode, width, height int) string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	labelStyle := lipgloss.NewStyle().Bold(true)

	sb.WriteString(headerStyle.Render(node.Label))
	sb.WriteString("\n\n")

	// Two columns: Prompt (80%) | Stats (20%)
	promptWidth := width * 4 / 5
	statsWidth := width - promptWidth - 2

	// Prompt column
	var promptContent strings.Builder
	promptContent.WriteString(labelStyle.Render("Prompt"))
	promptContent.WriteString("\n\n")

	// Find agent definition to show the method body
	for _, obj := range m.objects {
		if obj.Name == node.AgentName {
			// Show definition
			promptContent.WriteString(obj.Definition)
			break
		}
	}

	// Stats column
	var statsContent strings.Builder
	statsContent.WriteString(labelStyle.Render("Stats"))
	statsContent.WriteString("\n\n")

	run, hasRun := m.runs[node.AgentName]
	if hasRun && len(run.Iterations) > 0 {
		iters := run.Iterations
		statsContent.WriteString(fmt.Sprintf("iterations      %d\n", len(iters)))

		// Compute duration stats
		var durations []float64
		for _, ir := range iters {
			if !ir.FinishedAt.IsZero() {
				d := ir.FinishedAt.Sub(ir.StartedAt).Seconds()
				durations = append(durations, d)
			}
		}
		if len(durations) > 0 {
			mean, stddev := meanStddev(durations)
			statsContent.WriteString(fmt.Sprintf("mean(duration)  %.1fs\n", mean))
			statsContent.WriteString(fmt.Sprintf("stddev(duration) %.1fs\n", stddev))
		}
	} else {
		statsContent.WriteString("iterations      0\n")
	}

	// Layout columns
	promptStyle := lipgloss.NewStyle().Width(promptWidth)
	statsStyle := lipgloss.NewStyle().Width(statsWidth)

	left := promptStyle.Render(promptContent.String())
	right := statsStyle.Render(statsContent.String())

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "│ ", right))

	return sb.String()
}

// renderIterationView renders the detail for an iteration node.
func (m TUIModel) renderIterationView(node *TreeNode, width, height int) string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	sb.WriteString(headerStyle.Render(fmt.Sprintf("%s — iteration %d", node.AgentName, node.Iteration)))
	sb.WriteString("\n\n")

	// Find the iteration data
	run, hasRun := m.runs[node.AgentName]
	if !hasRun {
		sb.WriteString("No run data available.\n")
		return m.withInput(sb.String(), node, width, height)
	}

	var iter *IterationResult
	for i := range run.Iterations {
		if run.Iterations[i].Iteration == node.Iteration {
			iter = &run.Iterations[i]
			break
		}
	}

	if iter == nil {
		sb.WriteString("Iteration not found.\n")
		return m.withInput(sb.String(), node, width, height)
	}

	// Timing
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	duration := iter.FinishedAt.Sub(iter.StartedAt)
	if iter.FinishedAt.IsZero() {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Started: %s (running...)", iter.StartedAt.Format("15:04:05"))))
	} else {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Duration: %s", duration.Truncate(time.Millisecond))))
	}
	sb.WriteString("\n\n")

	// Output or error
	if iter.Error != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		sb.WriteString(errStyle.Render("Error: " + iter.Error))
		sb.WriteString("\n")
	}

	if iter.Output != "" {
		// Show output as chat message history
		sb.WriteString(iter.Output)
		sb.WriteString("\n")
	}

	return m.withInput(sb.String(), node, width, height)
}

// withInput appends the message input box to iteration view content.
func (m TUIModel) withInput(content string, node *TreeNode, width, height int) string {
	// Calculate space for input
	lines := strings.Count(content, "\n")
	inputHeight := 3 // separator + input + padding

	// Ensure content doesn't overflow
	maxContentLines := height - inputHeight
	if maxContentLines < 3 {
		maxContentLines = 3
	}
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > maxContentLines {
		// Scroll to show the most recent content
		contentLines = contentLines[len(contentLines)-maxContentLines:]
		content = strings.Join(contentLines, "\n")
		_ = lines // suppress unused warning
	}

	var sb strings.Builder
	sb.WriteString(content)

	// Separator
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width-2))
	sb.WriteString("\n")

	// Input
	m.msgInput.Width = width - 4
	sb.WriteString("❯ ")
	sb.WriteString(m.msgInput.View())
	sb.WriteString("\n")

	return sb.String()
}

// centeredMessage renders a centered message for the full screen.
func (m TUIModel) centeredMessage(msg string) string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(msg)
}

// --- Utility functions ---

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
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return mean, math.Sqrt(variance)
}
