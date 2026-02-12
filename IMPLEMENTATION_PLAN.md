# gcluster Implementation Plan

## Current State

Phases 0–3 and 2.3 are complete. The full apply→execute flow works: master listens on TCP, apply client parses `.p` files, extracts `agent-` definitions with resolved method bodies, sends to master, master stores and auto-starts pending agents. Each agent runs in its own goroutine calling `claude` CLI in a loop. Graceful shutdown stops agents, persists state, notifies steer clients. Next: Phase 4 (steer TUI).

---

## Completed

### Phase 0: Foundations and Reconciliation ✓
- **0.1 Rename `run` to `master`** — DONE.
- **0.2 StableID function** — DONE.
- **0.3 ClusterObject model** — DONE.
- **0.4 Network protocol** — DONE.

### Phase 1: Cluster State and Persistence ✓
- **1.1 In-memory store** — DONE. 12 tests in `store_test.go`.
- **1.2 Persistent storage** — DONE. 5 tests in `persist_test.go`.

### Phase 2: Master Process ✓
- **2.1 TCP listener and connection handling** — DONE. `cluster/server.go` with `Server` type: `NewServer`, `ListenAndServe`, `Stop`, `Addr`. Dispatches by message type, supports concurrent connections. 7 tests in `server_test.go`.
- **2.2 Apply handler** — DONE. Receives `apply_request`, calls `store.ApplyDefinitions()`, returns `apply_response`. After apply, auto-starts pending agents via executor.
- **2.3 Agent Execution Manager** — DONE. `cluster/executor.go` with `Executor` type. 9 tests in `executor_test.go`. See architecture notes below.
- **2.4 State push to steer clients** — DONE. `Store.OnChange` wired to `pushState` which sends `steer_state` to all subscribed clients.
- **2.5 Graceful shutdown** — DONE. SIGINT/SIGTERM handler persists state, executor stops all agents, sends `shutdown_notice` to steer clients, stops listener.

### Phase 3: Apply Client ✓
- **3.1 Parse and extract agent definitions** — DONE. `cmdApply` in `main.go`: parses `.p` file, registers all methods, filters `agent-` prefixed definitions, emits S-expressions, computes stable IDs, resolves method bodies for executor.
- **3.2 Send to master and print summary** — DONE. Connects to master TCP, sends `apply_request` with `AgentDef.Methods` populated, receives `apply_response`, prints summary.

**Testing:** 37 total tests passing: `store_test.go` (12), `protocol_test.go` (4), `persist_test.go` (5), `server_test.go` (7), `executor_test.go` (9).

---

## Remaining Work

### Phase 4: Steer TUI
- **4.1 Choose and integrate TUI framework** — `github.com/charmbracelet/bubbletea` + `lipgloss`
- **4.2 Network client for steer** — Connect to master, subscribe, receive state pushes, send inject
- **4.3 Tree sidebar** — Navigable tree of agents/loops/iterations
- **4.4 Detail view** — Agent/loop/iteration views
- **4.5 Message injection** — Input box in iteration view, sends `steer_inject`

### Phase 5: Polish and Edge Cases
- Idempotency verification, persistence verification, concurrent steer sessions, terminal resize, error handling polish
- Multi-step pipeline execution in executor (currently supports single loop step)
- Steer inject forwarding to running agent conversations

---

## Key Architecture Decisions

- **Server uses port 0 in tests** for parallel-safe testing (no port conflicts)
- **Apply client uses `sexp.EmitProgram` with filter** to emit individual agent S-expressions, then `sexp.StableID` for hashing
- **State push is wired via `Store.OnChange`** callback → `Server.pushState` → writes to all steer client connections
- **`--addr` and `--state` flags** added to both `master` and `apply` commands for testing flexibility
- **Method bodies travel with AgentDef** — `AgentDef.Methods` maps method name → resolved body text. The apply command resolves bodies at apply time using the registry. This keeps the executor decoupled from the parser/compiler/sexp packages.
- **ClaudeFunc injection for testability** — `type ClaudeFunc func(ctx, prompt) (string, error)`. Production uses `runtime.CallClaudeCapture`; tests inject fakes with controllable timing and failure modes.
- **One goroutine per agent with cancellable context** — Each agent gets a context derived from the executor's root context. `Stop(name)` cancels the agent context; `StopAll()` cancels root. Goroutines check `ctx.Done()` before each iteration.
- **Error recovery** — On `claude` CLI failure mid-iteration, the error is recorded in `IterationResult` and the agent continues to the next iteration. Only explicit stop or executor shutdown halts an agent.
- **NewServer accepts optional ClaudeFunc** — `NewServer(store, addr, claudeFn...)`. Tests omit it (no executor created); production passes `runtime.CallClaudeCapture`.
- **runtime.CallClaudeCapture exported** — Previously unexported `callClaudeCapture` renamed to `CallClaudeCapture` for use by the cluster executor.

---

*Updated 2026-02-12. Reflects completed Phases 0–3 including Phase 2.3 (executor).*
