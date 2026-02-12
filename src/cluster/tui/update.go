package tui

import (
	"fmt"

	"p2p/cluster"

	"github.com/stukennedy/tooey/app"
	"github.com/stukennedy/tooey/input"
)

func tuiUpdate(m interface{}, msg app.Msg) app.UpdateResult {
	mdl := m.(*Model)

	switch msg := msg.(type) {
	case stateMsg:
		p := cluster.SteerStatePayload(msg)
		mdl.Objects = p.Objects
		if p.Runs != nil {
			mdl.Runs = p.Runs
		}
		if p.Methods != nil {
			mdl.Methods = p.Methods
		}
		if p.Pipelines != nil {
			mdl.Pipelines = p.Pipelines
		}
		mdl.Ready = true
		return app.NoCmd(mdl)

	case errMsg:
		mdl.ErrText = msg.Error()
		return app.NoCmd(mdl)

	case reconnectMsg:
		mdl.ErrText = ""
		return app.NoCmd(mdl)

	case tickMsg:
		mdl.SpinFrame++
		return app.NoCmd(mdl)

	case app.KeyMsg:
		return handleKey(mdl, msg)

	case app.ScrollMsg:
		mdl.Scroll += msg.Delta
		if mdl.Scroll < 0 {
			mdl.Scroll = 0
		}
		return app.NoCmd(mdl)
	}

	if !mdl.Started {
		mdl.Started = true
		result := app.UpdateResult{Model: mdl, Cmds: []app.Cmd{disableMouseCmd}}
		result.Subs = []app.Sub{tickSub()}
		if mdl.Client != nil {
			result.Subs = append(result.Subs, stateSub(mdl.Client), errSub(mdl.Client), reconnectSub(mdl.Client))
		}
		return result
	}
	return app.NoCmd(mdl)
}

func handleKey(mdl *Model, msg app.KeyMsg) app.UpdateResult {
	entries := deriveTree(mdl.Objects, mdl.Runs, mdl.Pipelines, mdl.Search, mdl.Expanded)
	sel := clamp(mdl.Cursor, 0, len(entries)-1)

	switch mdl.Focused {
	case focusInput:
		return handleInputKey(mdl, msg, entries, sel)
	case focusSidebar:
		return handleSidebarKey(mdl, msg, entries, sel)
	case focusContent:
		return handleContentKey(mdl, msg)
	}
	if msg.Key.Type == input.RuneKey && msg.Key.Rune == 'q' {
		return app.UpdateResult{Model: nil}
	}
	return app.NoCmd(mdl)
}

func handleSidebarKey(mdl *Model, msg app.KeyMsg, entries []Entry, sel int) app.UpdateResult {
	moveCursor := func(delta int) {
		n := mdl.Cursor + delta
		if n < 0 {
			n = 0
		}
		if n >= len(entries) {
			n = len(entries) - 1
		}
		if n != mdl.Cursor {
			mdl.Cursor = n
			mdl.Scroll = 0
			mdl.Tail = true
		}
	}

	switch msg.Key.Type {
	case input.RuneKey:
		switch msg.Key.Rune {
		case 'q':
			return app.UpdateResult{Model: nil}
		case 'k':
			moveCursor(-1)
		case 'j':
			moveCursor(1)
		case '/':
			mdl.SearchInput = mdl.SearchInput.Update(msg.Key)
			mdl.Search = mdl.SearchInput.Value
		}
	case input.Up:
		moveCursor(-1)
	case input.Down:
		moveCursor(1)
	case input.Right:
		if sel >= 0 && sel < len(entries) && entries[sel].HasChildren && !entries[sel].Expanded {
			mdl.Expanded[expandKey(entries[sel])] = true
		}
	case input.Left:
		if sel >= 0 && sel < len(entries) && entries[sel].HasChildren && entries[sel].Expanded {
			mdl.Expanded[expandKey(entries[sel])] = false
		}
	case input.Enter:
		if sel >= 0 && sel < len(entries) && entries[sel].HasChildren {
			k := expandKey(entries[sel])
			mdl.Expanded[k] = !isExpanded(mdl.Expanded, k)
		}
	case input.PageUp:
		mdl.Scroll += 10
	case input.PageDown:
		mdl.Scroll -= 10
		if mdl.Scroll < 0 {
			mdl.Scroll = 0
		}
	}
	return app.NoCmd(mdl)
}

func handleContentKey(mdl *Model, msg app.KeyMsg) app.UpdateResult {
	scroll := func(delta int) {
		mdl.Scroll += delta
		if mdl.Scroll < 0 {
			mdl.Scroll = 0
		}
	}

	switch msg.Key.Type {
	case input.RuneKey:
		switch msg.Key.Rune {
		case 'q':
			return app.UpdateResult{Model: nil}
		case 'k':
			scroll(1)
		case 'j':
			scroll(-1)
		case 'G':
			mdl.Scroll = 0 // bottom
		case 'g':
			mdl.Scroll = 99999 // top
		}
	case input.Up:
		scroll(1)
	case input.Down:
		scroll(-1)
	case input.PageUp:
		scroll(10)
	case input.PageDown:
		scroll(-10)
	}
	return app.NoCmd(mdl)
}

func handleInputKey(mdl *Model, msg app.KeyMsg, entries []Entry, sel int) app.UpdateResult {
	if sel < 0 || sel >= len(entries) {
		return app.NoCmd(mdl)
	}
	entry := entries[sel]

	if msg.Key.Type == input.Enter {
		switch entry.Kind {
		case NodeIteration:
			text, ti := mdl.MsgInput.Submit()
			mdl.MsgInput = ti
			if text != "" && mdl.Client != nil {
				if err := mdl.Client.Inject(entry.Agent, entry.Step, entry.Iter, text); err != nil {
					mdl.ErrText = fmt.Sprintf("inject error: %v", err)
				}
			}
		case NodeLoop:
			text, ti := mdl.PromptInput.Submit()
			mdl.PromptInput = ti
			if text != "" && mdl.Client != nil {
				if err := mdl.Client.EditPrompt(entry.Agent, entry.Step, text); err != nil {
					mdl.ErrText = fmt.Sprintf("edit prompt error: %v", err)
				}
			}
		}
		return app.NoCmd(mdl)
	}

	switch entry.Kind {
	case NodeIteration:
		mdl.MsgInput = mdl.MsgInput.Update(msg.Key)
	case NodeLoop:
		mdl.PromptInput = mdl.PromptInput.Update(msg.Key)
	}
	return app.NoCmd(mdl)
}
