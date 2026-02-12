# gcluster Implementation Plan

## Completed

### Phase 0: Foundations and Reconciliation ✓
- **0.1 Rename `run` to `master`** — DONE. Updated `gcluster/main.go` to use `master` subcommand per spec.
- **0.2 StableID function** — DONE. Added `StableID()` to sexp package (full SHA-256 hex), factored out shared `stableHash()` used by both `StableID()` and `shortcode()`.
- **0.3 ClusterObject model** — DONE. Created `cluster/object.go` with `ClusterObject`, `Revision`, `AgentDef`, `ApplySummary`, and `RunState` types.
- **0.4 Network protocol** — DONE. Created `cluster/protocol.go` with JSON-over-TCP protocol types (`Envelope`, all message types, `NewEnvelope`/`DecodePayload` helpers).

### Phase 1: Cluster State and Persistence ✓
- **1.1 In-memory store** — DONE. Created `cluster/store.go` with thread-safe `Store` (idempotent `ApplyDefinitions`, `GetAgent`, `ListAgents`, `SetRunState`, `LoadState`, `OnChange` callback). 12 comprehensive tests in `store_test.go`.
- **1.2 Persistent storage** — DONE. Created `cluster/persist.go` with `SaveState`/`LoadState` (atomic writes via temp file + rename, corrupt file handling per spec). 5 tests in `persist_test.go`.

**Testing:** All phases covered with comprehensive tests: `store_test.go` (12 tests), `protocol_test.go` (4 tests), `persist_test.go` (5 tests), plus `StableID` tests in `sexp_test.go`.

---

## Current State

Phases 0 and 1 are complete. The cluster state store, persistence layer, and protocol definitions are ready. Next: implement the master TCP server and agent execution.

---

## Phase 0: Foundations and Reconciliation — DONE ✓

### 0.1 Rename `run` subcommand to `master` ✓
- **File:** `src/cmd/gcluster/main.go`
- **Status:** DONE. Subcommand renamed to `master` per spec.

### 0.2 Extend sexp hashing to full SHA-256 ✓
- **File:** `src/sexp/sexp.go`
- **Status:** DONE. Added `StableID(sexpr string) string` returning full SHA-256 hex. Factored out `stableHash()` shared with `shortcode()`.

### 0.3 Define the `ClusterObject` data model ✓
- **File:** `src/cluster/object.go`
- **Status:** DONE. Created with `ClusterObject`, `Revision`, `AgentDef`, `ApplySummary`, and `RunState` types per spec.

### 0.4 Define the network protocol ✓
- **File:** `src/cluster/protocol.go`
- **Status:** DONE. JSON-over-TCP protocol with `Envelope`, all message types (`apply_request`, `apply_response`, `steer_subscribe`, `steer_state`, `steer_inject`, `shutdown_notice`), and helper functions (`NewEnvelope`, `DecodePayload`).

---

## Phase 1: Cluster State and Persistence — DONE ✓

### 1.1 In-memory cluster state store ✓
- **File:** `src/cluster/store.go`
- **Status:** DONE. Thread-safe `Store` with all required methods:
  - `ApplyDefinitions([]AgentDef) -> ApplySummary` — idempotent upsert logic (same SHA-256 = unchanged, changed = new revision)
  - `GetAgent(name) -> ClusterObject`
  - `ListAgents() -> []ClusterObject`
  - `SetRunState(name, state)`
  - `LoadState(state)` — restore from disk
  - `OnChange(callback)` — subscribe to state mutations
- **Testing:** 12 comprehensive tests in `store_test.go` covering idempotency, revision tracking, concurrency, and edge cases.

### 1.2 Persistent storage (disk) ✓
- **File:** `src/cluster/persist.go`
- **Status:** DONE. `SaveState`/`LoadState` functions with atomic writes (temp file + rename), corrupt file handling (start fresh with warning, preserve old file for debugging).
- **Testing:** 5 tests in `persist_test.go` covering round-trip, corruption handling, atomic writes, and directory creation.

---

## Phase 2: Master Process

### 2.1 TCP listener and connection handling
- **File:** `src/cmd/gcluster/main.go` (master subcommand)
- **New file:** `src/cluster/server.go`
- **What:**
  - Listen on `127.0.0.1:43252`
  - Accept connections, read newline-delimited JSON messages
  - Dispatch to handler based on message type
  - Maintain a set of connected steer clients for push updates
  - Clear error if port is already in use

### 2.2 Apply handler (server side)
- **File:** `src/cluster/server.go`
- **What:** Receive `apply_request`, deserialize agent definitions, call `store.ApplyDefinitions()`, return `apply_response` with summary.

### 2.3 Agent execution manager
- **New file:** `src/cluster/executor.go`
- **What:**
  - When an agent transitions to `running`, spawn execution in a managed goroutine
  - Wrap the runtime's existing prompt/pipeline/loop execution
  - Track running agents, capture iteration outputs, maintain conversation history per iteration
  - Support stopping agents (send interrupt, wait for cleanup)
  - On `claude` CLI failure mid-iteration: record error, keep agent running, proceed to next iteration
- **Reuse:** `runtime/runtime.go` — `callClaude()`, loop execution logic. Factor out the core so it can be called from both `gprompt` and `gcluster master`.

### 2.4 State push to steer clients
- **File:** `src/cluster/server.go`
- **What:** After any state mutation (apply, run state change, new loop iteration), push state to all subscribed steer clients.

### 2.5 Graceful shutdown
- **File:** `src/cmd/gcluster/main.go` (master subcommand)
- **What:**
  - Trap SIGINT/SIGTERM
  - Stop all running agents (interrupt, wait with timeout)
  - Persist state to disk
  - Send `shutdown_notice` to all connected clients
  - Close listener, exit

---

## Phase 3: Apply Client

### 3.1 Parse and extract agent definitions
- **File:** `src/cmd/gcluster/main.go` (apply subcommand)
- **What:**
  - Parse `.p` file with existing parser
  - Register all methods (agents reference non-agent methods like `build`)
  - Filter for `agent-` prefixed definitions
  - Compile each to S-expression via sexp package
  - Compute stable ID via `StableID()` from 0.2
  - Include referenced non-agent method definitions in the payload (agents call them at execution time)
  - All-or-nothing: if parse fails, print error and exit nonzero. Nothing sent.
- **Reuse:** `parser/parser.go`, `sexp/sexp.go`, `registry/registry.go`, `compiler/compiler.go`

### 3.2 Send to master and print summary
- **File:** `src/cmd/gcluster/main.go` (apply subcommand)
- **What:**
  - Connect to `127.0.0.1:43252`
  - Send `apply_request` with agent definitions and supporting methods
  - Receive `apply_response`
  - Print summary: N created, N updated, N unchanged
  - Clear error on connection refused: "cannot connect to master at 127.0.0.1:43252 — is `gcluster master` running?"
  - Exit nonzero on failure

---

## Phase 4: Steer TUI

### 4.1 Choose and integrate TUI framework
- **Recommended:** `github.com/charmbracelet/bubbletea` with `github.com/charmbracelet/lipgloss` for styling.
- **Rationale:** Best Go TUI ergonomics (Elm architecture), good split pane support, active maintenance.

### 4.2 Network client for steer
- **New file:** `src/cluster/steer_client.go`
- **What:**
  - Connect to master on `127.0.0.1:43252`
  - Send `steer_subscribe`
  - Read state updates in a goroutine, feed into bubbletea as messages
  - Handle disconnect: show banner, attempt reconnect with backoff
  - Send `steer_inject` when user injects a message
  - Clear error on master not running (same as apply)

### 4.3 Tree sidebar (left pane)
- **What:**
  - Render navigable tree of all agents from cluster state
  - Each agent node expands to show loop steps, each step expands to show iterations (max 4 most recent, latest first in bold)
  - Navigation: up/down arrows move highlight, left/right collapse/expand
  - Search: text input at top filters tree by name
  - Live updates: new iterations appear in tree within 1 second without refresh

### 4.4 Detail view (right pane)
- **What:** Render based on selected tree node type:
  - **AgentView:** Reserved for future agent-level metadata (currently empty per spec)
  - **LoopView:** Two columns — prompt text (80%) + stats (20%): iterations, mean/stddev duration, mean/stddev tokens
  - **LoopIterationView:** Full chat message history, scrollable. Input box at bottom for message injection.
- **Live updates:** Refresh when new data arrives for the currently selected item

### 4.5 Message injection
- **What:**
  - In LoopIterationView, input box at bottom for steering messages
  - On submit, send `steer_inject` to master
  - Master injects message into agent's conversation
  - Both sending and receiving clients see the message reflected in chat history

---

## Phase 5: Polish and Edge Cases

### 5.1 Idempotency verification
- Test: apply same file twice, verify no new revisions created and no errors.
- Test: change one agent's body, reapply, verify only that agent gets a new revision.

### 5.2 Persistence verification
- Test: apply agents, stop master, restart, verify agents survive.
- Test: corrupt state file, verify master starts fresh with warning.

### 5.3 Concurrent steer sessions
- Test: two steer terminals see same cluster state.
- Test: two users steer the same iteration, both messages delivered in arrival order.

### 5.4 Terminal resize handling
- TUI reflows on resize without crashing or corrupting display.

### 5.5 Error handling polish
- Master not running → clear connection error for both apply and steer
- Port already in use → clear error naming port
- Agent references undefined method → compile error, apply fails
- Very long chat history → scrollable, not truncated

---

## Dependency Graph

```
Phase 0 (foundations)
  0.1 rename run -> master
  0.2 full SHA-256
  0.3 ClusterObject model
  0.4 protocol definition
    ↓
Phase 1 (state)              Phase 3 (apply client)
  1.1 in-memory store          3.1 parse + extract  [needs 0.2, 0.3]
  1.2 persistence              3.2 send + print     [needs 0.4, 3.1]
    ↓
Phase 2 (master)
  2.1 TCP listener           [needs 0.4]
  2.2 apply handler          [needs 1.1, 2.1]
  2.3 executor               [needs 1.1, runtime.go]
  2.4 state push             [needs 2.1, 1.1]
  2.5 graceful shutdown      [needs 2.1, 2.3, 1.2]
    ↓
Phase 4 (steer TUI)
  4.1 framework choice
  4.2 network client         [needs 0.4]
  4.3 tree sidebar           [needs 4.1, 4.2]
  4.4 detail view            [needs 4.3]
  4.5 message injection      [needs 4.2, 4.4]
    ↓
Phase 5 (polish)             [needs all above]
```

## Reuse Summary

| Existing Package | Reused In | How |
|---|---|---|
| `parser/parser.go` | Phase 3 (apply) | Parse `.p` files, extract `agent-` definitions |
| `sexp/sexp.go` | Phase 0.2, Phase 3 | Canonical S-expression emission, SHA-256 stable IDs |
| `pipeline/pipeline.go` | Phase 2.3 (executor) | Pipeline structure for agent loop definitions |
| `compiler/compiler.go` | Phase 3 (apply) | Compile parsed nodes to Plans |
| `runtime/runtime.go` | Phase 2.3 (executor) | Execute prompts and loops via `claude` CLI |
| `registry/registry.go` | Phase 2.3 (executor), Phase 3 | Method resolution during execution and apply |
| `stdlib/stdlib.go` | Phase 2.3 (executor) | Standard library definitions |
| `debug/debug.go` | All phases | Debug logging throughout |

---

*Generated 2026-02-12. Reflects specs in `specs/cli/gcluster/` and `specs/concepts/gcluster.md` against source in `src/`.*
