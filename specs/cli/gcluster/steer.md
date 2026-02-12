gcluster steer
==============

## Purpose

Agents run on the master, but humans need to see what's happening and intervene when needed. `gcluster steer <file.p>` provides that window — a TUI that connects to the control plane and lets you observe agent state, read iteration history, and inject messages into a running agent's conversation.

The name "steer" is deliberate: you're not controlling the agents, you're nudging them. The agents run autonomously; steering lets you course-correct.

## Behaviour

`gcluster steer <file.p>` opens a terminal UI connected to the master. The local file is for reference only — the source of truth is the cluster.

### Layout

Two panes side by side:

- **Left (tree sidebar)**: navigable tree of all agents and their substructure.
- **Right (detail view)**: content for the currently highlighted node.

```
┌───────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────┐
│ Agents                                        │                                                                          │
│ ────────────────────────────────────────────  │ ● Now let me run the next iteration.                                     │
│ [ Search agents...                      ]     │                                                                          │
│                                               │ ● Explore(Read BACKLOG.md)                                               │
│ ▾ builder                                     │   ⎿  Done (3 tool uses · 8.2k tokens · 22s)                             │
│   ▾ loop(build)                               │                                                                          │
│     ▸ **iteration 3**                         │ ● Pick item, implement, commit.                                          │
│       iteration 2                             │                                                                          │
│       iteration 1                             │ ● Task(Fix failing tests)                                                │
│       iteration 0                             │   ⎿  Done (0 tool uses · 4.1k tokens · 12s)                             │
│                                               │                                                                          │
│ ▸ bugfixer                                    │ ● Write(src/feature.go)                                                  │
│   ▸ loop(bugfix)                              │   ⎿  Wrote 41 lines to src/feature.go                                   │
│                                               │                                                                          │
│ ▸ release-manager                             │ ● Done. Summary: shipped one change, tests green.                        │
│   ▸ loop(releasemgmt)                         │                                                                          │
│                                               │                                                                          │
│                                               │ ───────────────────────────────────────────────────────────────────────  │
│                                               │ ❯ send message…                                                          │
└───────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────┐
│ Agents                                        │                                                                          │
│ ────────────────────────────────────────────  │ Prompt                                           │ Stats                  │
│ [ Search contexts...                      ]   │                                                  │                        │
│                                               │ build:                                           │ iterations      4      │
│ ▾ builder                                     │   Read BACKLOG.md, pick one item, build it out,  │ mean(duration)  38s    │
│   ▸ **loop(build)**                           │   git commit, then mark as complete.             │ stddev(duration) 9s    │
│       iteration 3                             │                                                  │ mean(tokens)    8.2k   │
│       iteration 2                             │                                                  │ stddev(tokens)  1.1k   │
│       iteration 1                             │                                                  │                        │
│       iteration 0                             │                                                  │                        │
│                                               │                                                  │                        │
│ ▸ bugfixer                                    │                                                  │                        │
│   ▸ loop(bugfix)                              │                                                  │                        │
│                                               │                                                  │                        │
│ ▸ release-manager                             │                                                  │                        │
│   ▸ loop(releasemgmt)                         │                                                  │                        │
│                                               │                                                  │                        │
│                                               │ ───────────────────────────────────────────────  │                        | 
│                                               │ ❯ edit prompt…                                                            │
└───────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────┘
```


### Tree sidebar

Displays all agents in the cluster as a navigable tree.

```
Agents
[Search contexts...]

builder
    ⌄ loop(build)
      › iteration N
        iteration N-1
        iteration N-2
        iteration 0
bugfixer
    › loop(bugfix)
release-manager
    › loop(releasemgmt)
```

- **Navigation**: up/down arrows move the highlight, left/right collapse/expand children.
- **Search**: a text input at the top filters the tree by name.
- **Loop children**: loop nodes show their iterations as children. Maximum 4 most recent iterations displayed. The latest iteration is listed first and displayed in bold.
- **Live updates**: new iterations appear in the tree as they start, without requiring manual refresh.
- **Shift+Tab** to swap between tree and input.

### Detail views

The right pane renders a view based on the highlighted node's type.

**AgentView** (agent node highlighted):
No content currently. Reserved for future agent-level metadata.

**LoopView** (loop node highlighted):
Two columns:

| Column | Width | Content |
|--------|-------|---------|
| Prompt | 80% | The method body text used by this loop. |
| Stats | 20% | `iterations`, `mean(duration)`, `stddev(duration)`, `mean(tokens)`, `stddev(tokens)`. |

**LoopIterationView** (iteration node highlighted):
The chat message history for that iteration. An input box at the bottom allows sending a message into the agent's conversation to steer it.

## Acceptance criteria

- Opening `steer` shows all agents currently in the cluster, matching what `apply` sent.
- Navigating to a loop node shows the prompt text and iteration statistics.
- Navigating to an iteration shows the full chat history for that iteration.
- Typing a message in the iteration input box and submitting it injects that message into the agent's conversation on the master.
- A new iteration started by an agent appears in the tree within 1 second without user action.
- The search bar filters the tree — typing "build" hides agents whose names don't match.
- Two `steer` terminals connected to the same master show consistent state.
- Closing a `steer` terminal does not affect agent execution on the master.

## Edge cases

- **Master not running**: Exits with a clear connection error, same as `apply`.
- **Master disconnects while steer is open**: The TUI shows a disconnection banner and attempts to reconnect. It does not crash or silently go stale.
- **Zero agents in cluster**: The tree is empty. The TUI displays a message like "No agents. Run `gcluster apply <file.p>` to add agents."
- **Agent has zero iterations**: The loop node is visible but has no iteration children. The LoopView shows the prompt and stats with `iterations: 0`.
- **Very long chat history**: The LoopIterationView scrolls. It does not truncate or drop messages.
- **Concurrent steering**: Two users steer the same iteration simultaneously. Both messages are delivered to the agent in arrival order. Both clients see both messages reflected in the chat history.
- **Terminal resize**: The TUI reflows to fit the new terminal dimensions without crashing or corrupting the display.

## Dependencies

- Network connection to the master at `127.0.0.1:43252`.
- Terminal UI framework (TBD) for rendering the tree and detail panes.
- The local `.p` file is optional context — the TUI functions entirely from cluster state.
