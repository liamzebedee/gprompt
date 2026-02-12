package tui

import (
	"fmt"
	"strings"
	"time"

	"p2p/cluster"

	"github.com/stukennedy/tooey/input"
	"github.com/stukennedy/tooey/node"
)

func tuiView(m interface{}, focused string) node.Node {
	mdl := m.(*Model)
	mdl.Focused = focused

	if mdl.ErrText != "" && !mdl.Ready {
		return node.Column(node.Spacer(),
			node.TextStyled(fmt.Sprintf("  Error: %s", mdl.ErrText), 1, 0, node.Bold),
			node.Spacer())
	}
	if !mdl.Ready {
		return node.Column(node.Spacer(), node.Text("  Connecting to master..."), node.Spacer())
	}

	entries := deriveTree(mdl.Objects, mdl.Runs, mdl.Pipelines, mdl.Search, mdl.Expanded)
	sel := clamp(mdl.Cursor, 0, len(entries)-1)
	return node.Row(renderSidebar(entries, sel, mdl, focused), renderDetail(entries, sel, mdl, focused))
}

// --- Sidebar ---

func renderSidebar(entries []Entry, sel int, mdl *Model, focused string) node.Node {
	mdl.SearchInput.Focused = focused == focusSidebar
	_, cols := input.TermSize()
	w := cols*3/10 - 6
	if w < 14 {
		w = 14
	}

	header := []node.Node{
		node.TextStyled(" Agents", 0, 0, node.Bold),
		mdl.SearchInput.Render(" / ", 0, 0, w),
	}

	tree := make([]node.Node, 0, len(entries))
	if len(entries) == 0 {
		if len(mdl.Objects) == 0 {
			tree = append(tree, node.TextStyled("  No agents.", 8, 0, 0),
				node.TextStyled("  gcluster apply <file.p>", 8, 0, 0))
		} else {
			tree = append(tree, node.TextStyled("  No matching agents.", 8, 0, 0))
		}
	}
	for i, e := range entries {
		indent := strings.Repeat("  ", e.Depth)
		icon := "  "
		if (e.Kind == NodeAgent || e.Kind == NodeLoop) && e.HasChildren {
			if e.Expanded {
				icon = "▾ "
			} else {
				icon = "▸ "
			}
		}
		label := " " + indent + icon + e.Label

		switch {
		case i == sel && focused == focusSidebar:
			tree = append(tree, node.TextStyled(label, 230, 62, node.Bold))
		case i == sel:
			tree = append(tree, node.TextStyled(label, 0, 0, node.Bold|node.Underline))
		case e.Live:
			tree = append(tree, node.TextStyled(label, 0, 0, node.Bold))
		case e.Kind == NodeIteration:
			tree = append(tree, node.TextStyled(label, 8, 0, 0))
		default:
			tree = append(tree, node.Text(label))
		}
	}

	rows, _ := input.TermSize()
	vis := rows - 6
	if vis < 3 {
		vis = 3
	}
	ensureVisible(&mdl.SidebarScroll, sel, vis)

	treeCol := node.Column(tree...).WithFlex(1).WithScrollOffset(mdl.SidebarScroll)
	help := node.TextStyled(" ↑↓ nav  ←→ fold  S-Tab pane  q quit", 8, 0, 0)

	var all []node.Node
	all = append(all, header...)
	all = append(all, treeCol, help)

	return node.Box(node.BorderRounded,
		node.Column(all...).WithFlex(1),
	).WithFlex(3).WithKey(focusSidebar).WithFocusable()
}

// --- Detail pane ---

func renderDetail(entries []Entry, sel int, mdl *Model, focused string) node.Node {
	if len(entries) == 0 || sel < 0 || sel >= len(entries) {
		return node.Column(
			node.Box(node.BorderRounded,
				node.Column(node.Spacer(), node.Text("  No selection"), node.Spacer()).WithFlex(1),
			).WithFlex(1),
		).WithFlex(7)
	}

	entry := entries[sel]
	var content []node.Node
	var inputNode node.Node
	var hasInput bool

	_, cols := input.TermSize()
	inputW := cols*7/10 - 4
	if inputW < 20 {
		inputW = 20
	}

	switch entry.Kind {
	case NodeAgent:
		content = []node.Node{
			node.Spacer(),
			node.TextStyled("  "+entry.Agent, 0, 0, node.Bold),
			node.Text(""),
			node.TextStyled("  Select a loop or iteration for details.", 8, 0, 0),
			node.Spacer(),
		}
	case NodeLoop:
		content = buildLoopContent(entry, mdl)
		mdl.PromptInput.Focused = focused == focusInput
		inputNode = mdl.PromptInput.Render("❯ ", 0, 0, inputW)
		hasInput = true
	case NodeIteration:
		content = buildIterContent(entry, mdl)
		mdl.MsgInput.Focused = focused == focusInput
		inputNode = mdl.MsgInput.Render("❯ ", 0, 0, inputW)
		hasInput = true
	}

	if mdl.ErrText != "" {
		content = append(content, node.Text(""),
			node.TextStyled("  ⚠ "+mdl.ErrText, 1, 0, node.Bold))
	}

	// Always bottom-anchored. Scroll offset = lines from bottom.
	contentCol := node.Column(content...).WithFlex(1).
		WithScrollToBottom().WithScrollOffset(mdl.Scroll)

	if !hasInput {
		return node.Column(
			node.Box(node.BorderRounded, contentCol).WithFlex(1),
		).WithFlex(7)
	}

	return node.Column(
		node.Box(node.BorderRounded,
			node.Column(
				contentCol,
				inputNode.WithKey(focusInput).WithFocusable(),
			).WithFlex(1),
		).WithFlex(1),
	).WithFlex(7)
}

// --- Content builders ---

func buildLoopContent(entry Entry, mdl *Model) []node.Node {
	// Left: prompt
	var promptLines []node.Node
	promptLines = append(promptLines,
		node.TextStyled("  Prompt", 0, 0, node.Bold), node.Text(""))

	displayed := false
	if methods, ok := mdl.Methods[entry.Agent]; ok {
		if body, ok := methods[entry.Step]; ok {
			for _, line := range strings.Split(body, "\n") {
				promptLines = append(promptLines, node.Text("  "+line))
			}
			displayed = true
		}
	}
	if !displayed {
		for _, obj := range mdl.Objects {
			if obj.Name == entry.Agent {
				promptLines = append(promptLines, node.Text("  "+obj.Definition))
				break
			}
		}
	}

	// Right: stats
	var statsLines []node.Node
	statsLines = append(statsLines,
		node.TextStyled("  Stats", 0, 0, node.Bold), node.Text(""))

	run, hasRun := mdl.Runs[entry.Agent]
	if hasRun && len(run.Iterations) > 0 {
		iters := run.Iterations
		statsLines = append(statsLines, node.Text(fmt.Sprintf("  iterations      %d", len(iters))))
		var durations []float64
		for _, ir := range iters {
			if !ir.FinishedAt.IsZero() {
				durations = append(durations, ir.FinishedAt.Sub(ir.StartedAt).Seconds())
			}
		}
		if len(durations) > 0 {
			m, s := meanStddev(durations)
			statsLines = append(statsLines,
				node.Text(fmt.Sprintf("  mean(duration)  %.1fs", m)),
				node.Text(fmt.Sprintf("  stddev(duration) %.1fs", s)))
		}
	} else {
		statsLines = append(statsLines, node.Text("  iterations      0"))
	}

	promptCol := node.Column(promptLines...).WithFlex(4)
	statsCol := node.Column(statsLines...).WithFlex(1)

	return []node.Node{
		node.TextStyled("  "+entry.Label, 0, 0, node.Bold|node.Underline),
		node.Text(""),
		node.Row(promptCol, statsCol).WithFlex(1),
	}
}

func buildIterContent(entry Entry, mdl *Model) []node.Node {
	items := []node.Node{
		node.TextStyled(fmt.Sprintf("  %s — iteration %d", entry.Agent, entry.Iter), 0, 0, node.Bold|node.Underline),
		node.Text(""),
	}

	run, ok := mdl.Runs[entry.Agent]
	if !ok {
		return append(items, node.Text("  No run data available."))
	}

	var iter *cluster.IterationResult
	if run.LiveIter != nil && run.LiveIter.Iteration == entry.Iter {
		iter = run.LiveIter
	} else {
		for i := range run.Iterations {
			if run.Iterations[i].Iteration == entry.Iter {
				iter = &run.Iterations[i]
				break
			}
		}
	}
	if iter == nil {
		return append(items, node.Text("  Iteration not found."))
	}

	if iter.FinishedAt.IsZero() {
		items = append(items, node.TextStyled(
			fmt.Sprintf("  Started: %s (running...)", iter.StartedAt.Format("15:04:05")), 8, 0, 0))
	} else {
		items = append(items, node.TextStyled(
			fmt.Sprintf("  Duration: %s", iter.FinishedAt.Sub(iter.StartedAt).Truncate(time.Millisecond)), 8, 0, 0))
	}
	items = append(items, node.Text(""))

	if iter.Error != "" {
		items = append(items, node.TextStyled("  Error: "+iter.Error, 1, 0, node.Bold), node.Text(""))
	}

	for _, msg := range iter.Messages {
		switch msg.Type {
		case "text":
			for _, line := range strings.Split(msg.Content, "\n") {
				items = append(items, node.Text("  "+line))
			}
		case "tool_use":
			items = append(items, node.TextStyled(fmt.Sprintf("  [Tool: %s]", msg.Content), 33, 0, node.Bold))
		case "tool_result":
			c := msg.Content
			if len(c) > 200 {
				c = c[:200] + "…"
			}
			items = append(items, node.TextStyled("  "+c, 8, 0, 0))
		}
	}
	return items
}
