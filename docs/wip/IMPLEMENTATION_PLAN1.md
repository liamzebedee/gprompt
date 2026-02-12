# gcluster Implementation Plan

## Current State

All phases complete (0–5.6). The full apply→execute→steer flow works with multi-step pipeline support, prompt editing, concurrent session consistency, and terminal resize handling. All 95 tests pass across all packages.

---

## Completed

### Phase 0: Foundations and Reconciliation ✓
### Phase 1: Cluster State and Persistence ✓
### Phase 2: Master Process ✓
### Phase 3: Apply Client ✓
### Phase 4: Steer TUI ✓
### Phase 5: Polish and Edge Cases ✓

- **5.1 Steer inject forwarding** — DONE.
- **5.2 Disconnection banner + auto-reconnect** — DONE.
- **5.3 Executor.Start mutex deadlock fix** — DONE.
- **5.4 Multi-step pipeline execution** — DONE.
- **5.5 TUI method bodies, pipeline-aware tree, and scrollable history** — DONE.
- **5.6 Prompt editing, concurrent sessions, terminal resize** — DONE.
  1. **Prompt editing in LoopView** — Full implementation of the spec's `❯ edit prompt…` feature:
     - New protocol message `steer_edit_prompt` with `SteerEditPromptRequest` payload (agent_name, method_name, new_body)
     - `SteerClient.EditPrompt()` sends the request to the server
     - `Server.handleSteerEditPrompt()` updates the server's cached method body, notifies the executor, and pushes updated state to all steer clients
     - `Executor.UpdateMethodBody()` delivers the update to the running agent's loop goroutine via `methodCh` channel
     - `runAgentLoop` drains `methodCh` between iterations and replaces `basePrompt` permanently
     - TUI adds `promptInput` textinput to LoopView with Shift+Tab focus switching and Enter to submit
     - Semantically distinct from inject (one-time prepend): edit prompt permanently changes the base prompt for all future iterations
  2. **Concurrent steer session consistency** — Verified via `TestConcurrentSteerSessionConsistency`: two steer clients inject messages into the same agent concurrently, both receive state pushes, and both injected messages are delivered to the agent
  3. **Terminal resize reflow** — Verified via `TestTUITerminalResize` and `TestTUITerminalResizeZeroSize`: TUI handles WindowSizeMsg at various sizes (large, small, very small, zero) without crashing, renders correctly at each node type

**Testing:** 95 total tests passing across all packages.

Phase 5.6 tests:
- `TestTUILoopViewHasPromptInput` — LoopView renders edit prompt input with separator and indicator
- `TestTUIPromptEditFocusSwitching` — Shift+Tab focuses promptInput for loop nodes, msgInput for iteration nodes
- `TestServerEditPromptUpdatesMethodCache` — edit_prompt updates server cache and pushes to steer clients
- `TestUpdateMethodBody` — executor delivers method body update to running agent's loop
- `TestUpdateMethodBodyNonRunning` — no-op for non-running agents (no panic)
- `TestConcurrentSteerSessionConsistency` — two clients inject concurrently, both see consistent state
- `TestTUITerminalResize` — TUI handles resize at all node types without crashing
- `TestTUITerminalResizeZeroSize` — zero-size terminal shows initialization message

---

## Remaining Work

No remaining items. All spec features are implemented.

---

## Key Architecture Decisions

- **Server uses port 0 in tests** for parallel-safe testing (no port conflicts)
- **Apply client uses `sexp.EmitProgram` with filter** to emit individual agent S-expressions, then `sexp.StableID` for hashing
- **State push is wired via `Store.OnChange`** callback → `Server.pushState` → writes to all steer client connections
- **Iteration data is runtime-only, not persisted** — `AgentRunSnapshot` lives in `SteerStatePayload.Runs`, separate from `ClusterObject` which gets persisted
- **Executor.OnIteration callback** triggers state pushes on every iteration completion
- **Snapshot caps iterations at 10** per agent to limit payload size. The TUI shows max 4 in the tree.
- **Method bodies and pipeline structure travel with AgentDef** — resolved at apply time, keeping executor decoupled from parser/compiler/sexp/pipeline packages
- **Method bodies and pipelines travel in SteerStatePayload** — server caches from apply requests and includes in every state push
- **Pipeline execution: setup then loop** — Simple/map steps are one-shot setup; only loop iterations are tracked
- **splitItems duplication** — duplicated from `runtime/runtime.go` into `cluster/executor.go` to avoid import cycle
- **ClaudeFunc injection for testability** — `type ClaudeFunc func(ctx, prompt) (string, error)`. Tests inject fakes.
- **One goroutine per agent with cancellable context** — `Stop(name)` cancels agent context; `StopAll()` cancels root
- **Error recovery** — Claude CLI failure mid-iteration recorded in IterationResult, agent continues. Setup step failures abort pipeline.
- **SteerClient uses channels** — `StateCh` (buffered 16), `ErrCh` (buffered 4). Non-blocking send with drain-and-replace.
- **Prompt editing uses separate protocol message** — `steer_edit_prompt` is distinct from `steer_inject` because it permanently replaces the base prompt rather than being a one-time prepend. The server's `agentMethods` cache is the source of truth; the executor receives updates via `methodCh` channel on `AgentRun`.

---

*Updated 2026-02-12. All phases complete. 95 tests passing.*







Missing:
- apply should restart agents
- toggles do not work when there are no children (makes sense)









