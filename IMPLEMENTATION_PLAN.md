# gcluster Implementation Plan

## Current State

Phases 0–4 are complete. The full apply→execute→steer flow works: master listens on TCP, apply client parses `.p` files, extracts `agent-` definitions with resolved method bodies, sends to master, master stores and auto-starts pending agents. Each agent runs in its own goroutine calling `claude` CLI in a loop. The steer TUI connects to master, subscribes for real-time state updates (including iteration data), and presents a two-pane view with tree navigation and detail views. Next: Phase 5 (polish and edge cases).

---

## Completed

### Phase 0: Foundations and Reconciliation ✓
### Phase 1: Cluster State and Persistence ✓
### Phase 2: Master Process ✓
### Phase 3: Apply Client ✓
### Phase 4: Steer TUI ✓

- **4.1 TUI framework** — DONE. `bubbletea` + `lipgloss` + `bubbles` (textinput). Dependencies added to `go.mod`.
- **4.2 Network client for steer** — DONE. `cluster/steerclient.go`: connects to master TCP, sends `steer_subscribe`, reads state pushes in background goroutine, delivers via channels. Supports `Inject()` for message injection. 4 tests in `steerclient_test.go`.
- **4.3 Tree sidebar** — DONE. `cluster/tui.go`: navigable tree of agents → loop(method) → iterations. Up/down moves highlight, left/right collapse/expand, `/` activates search. Maximum 4 most recent iterations shown. Most recent iteration displayed bold. Search filters tree by agent name.
- **4.4 Detail views** — DONE. Three views based on selected node type:
  - **AgentView**: agent name, state, revision count, run info
  - **LoopView**: two-column layout with prompt (80%) and stats (20%) — iterations count, mean/stddev duration
  - **IterationView**: timing, output/error display, message input box for injection
- **4.5 Message injection** — DONE. Input box in iteration view sends `steer_inject` to master via SteerClient. Shift+Tab swaps focus between tree and input. Enter submits.
- **4.6 Iteration data in state pushes** — DONE. Extended `SteerStatePayload` with `Runs map[string]AgentRunSnapshot` carrying iteration results. Executor fires `OnIteration` callback after each iteration, triggering real-time pushes. Snapshot capped at 10 most recent iterations per agent to limit payload size.
- **4.7 cmdSteer wired** — DONE. `cmdSteer` in `main.go` parses `--addr` flag, creates `SteerClient`, runs bubbletea `Program` with `AltScreen`.

**Testing:** 49 total tests passing: `store_test.go` (12), `protocol_test.go` (4), `persist_test.go` (5), `server_test.go` (7), `executor_test.go` (12), `steerclient_test.go` (4), `tui_test.go` (5).

---

## Remaining Work

### Phase 5: Polish and Edge Cases
- Disconnection banner + reconnect in steer TUI (currently shows error, no auto-reconnect)
- Multi-step pipeline execution in executor (currently supports single loop step)
- Steer inject forwarding to running agent conversations (currently logged on server, not delivered to agent)
- Concurrent steer session consistency verification
- Terminal resize reflow testing
- Prompt editing in LoopView (spec mentions "edit prompt…" input)
- Very long chat history scrolling in IterationView

---

## Key Architecture Decisions

- **Server uses port 0 in tests** for parallel-safe testing (no port conflicts)
- **Apply client uses `sexp.EmitProgram` with filter** to emit individual agent S-expressions, then `sexp.StableID` for hashing
- **State push is wired via `Store.OnChange`** callback → `Server.pushState` → writes to all steer client connections
- **Iteration data is runtime-only, not persisted** — `AgentRunSnapshot` lives in `SteerStatePayload.Runs`, separate from `ClusterObject` which gets persisted. This keeps the persistence layer lean and the protocol rich for observation.
- **Executor.OnIteration callback** triggers state pushes on every iteration completion, complementing `Store.OnChange` which only fires on store mutations (apply, state transitions). This ensures steer clients see new iterations in real time.
- **Snapshot caps iterations at 10** per agent to limit payload size. The TUI shows max 4 in the tree.
- **Method bodies travel with AgentDef** — `AgentDef.Methods` maps method name → resolved body text. The apply command resolves bodies at apply time using the registry. This keeps the executor decoupled from the parser/compiler/sexp packages.
- **ClaudeFunc injection for testability** — `type ClaudeFunc func(ctx, prompt) (string, error)`. Production uses `runtime.CallClaudeCapture`; tests inject fakes with controllable timing and failure modes.
- **One goroutine per agent with cancellable context** — Each agent gets a context derived from the executor's root context. `Stop(name)` cancels the agent context; `StopAll()` cancels root.
- **Error recovery** — On `claude` CLI failure mid-iteration, the error is recorded in `IterationResult` and the agent continues to the next iteration.
- **SteerClient uses channels** — `StateCh` (buffered 16) delivers state payloads; `ErrCh` (buffered 4) delivers connection errors. Non-blocking send with drain-and-replace to avoid blocking the read goroutine.

---

*Updated 2026-02-12. Reflects completed Phases 0–4 including steer TUI.*
