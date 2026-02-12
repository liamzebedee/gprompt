# gcluster Implementation Plan

## Current State

Phases 0, 1, 2 (partial), and 3 are complete. The full apply flow works end-to-end: master listens on TCP, apply client parses `.p` files, extracts `agent-` definitions, computes stable IDs, sends to master, receives summary. State push to steer clients works. Graceful shutdown with persistence works. Next: Phase 2.3 (agent executor) and Phase 4 (steer TUI).

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

### Phase 2: Master Process (partial) ✓
- **2.1 TCP listener and connection handling** — DONE. `cluster/server.go` with `Server` type: `NewServer`, `ListenAndServe`, `Stop`, `Addr`. Dispatches by message type, supports concurrent connections. 7 tests in `server_test.go`.
- **2.2 Apply handler** — DONE. Receives `apply_request`, calls `store.ApplyDefinitions()`, returns `apply_response`.
- **2.4 State push to steer clients** — DONE. `Store.OnChange` wired to `pushState` which sends `steer_state` to all subscribed clients. Verified with multiple concurrent steer clients.
- **2.5 Graceful shutdown** — DONE. SIGINT/SIGTERM handler persists state to disk, sends `shutdown_notice` to steer clients, stops listener. `cmdMaster` in `main.go`.

### Phase 3: Apply Client ✓
- **3.1 Parse and extract agent definitions** — DONE. `cmdApply` in `main.go`: parses `.p` file, registers all methods (stdlib + imports + file), filters `agent-` prefixed definitions, emits S-expressions via `sexp.EmitProgram`, computes `sexp.StableID`.
- **3.2 Send to master and print summary** — DONE. Connects to master TCP, sends `apply_request`, receives `apply_response`, prints summary with created/updated/unchanged counts. Clear error on connection refused.

**Testing:** 28 total tests passing: `store_test.go` (12), `protocol_test.go` (4), `persist_test.go` (5), `server_test.go` (7).

---

## Remaining Work

### Phase 2.3: Agent Execution Manager
- **File:** `src/cluster/executor.go`
- **What:**
  - When an agent transitions to `running`, spawn execution in a managed goroutine
  - Wrap the runtime's existing prompt/pipeline/loop execution
  - Track running agents, capture iteration outputs, maintain conversation history per iteration
  - Support stopping agents (send interrupt, wait for cleanup)
  - On `claude` CLI failure mid-iteration: record error, keep agent running, proceed to next iteration
- **Reuse:** `runtime/runtime.go` — `callClaude()`, loop execution logic. Factor out the core so it can be called from both `gprompt` and `gcluster master`.
- **Design notes:** The `runtime.callClaude` is unexported. Need to either export it or create a new execution function in the runtime package that the executor can call. Consider adding `runtime.ExecutePromptCapture` or similar.

### Phase 4: Steer TUI
- **4.1 Choose and integrate TUI framework** — `github.com/charmbracelet/bubbletea` + `lipgloss`
- **4.2 Network client for steer** — Connect to master, subscribe, receive state pushes, send inject
- **4.3 Tree sidebar** — Navigable tree of agents/loops/iterations
- **4.4 Detail view** — Agent/loop/iteration views
- **4.5 Message injection** — Input box in iteration view, sends `steer_inject`

### Phase 5: Polish and Edge Cases
- Idempotency verification, persistence verification, concurrent steer sessions, terminal resize, error handling polish

---

## Key Architecture Decisions

- **Server uses port 0 in tests** for parallel-safe testing (no port conflicts)
- **Apply client uses `sexp.EmitProgram` with filter** to emit individual agent S-expressions, then `sexp.StableID` for hashing
- **State push is wired via `Store.OnChange`** callback → `Server.pushState` → writes to all steer client connections
- **`--addr` and `--state` flags** added to both `master` and `apply` commands for testing flexibility

---

*Updated 2026-02-12. Reflects completed Phases 0-3 and Phase 2 (partial).*
