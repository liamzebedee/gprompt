# gcluster Implementation Plan

## Current State

Phases 0–5 are mostly complete. The full apply→execute→steer flow works with multi-step pipeline support: master listens on TCP, apply client parses `.p` files, extracts `agent-` definitions with resolved method bodies and pipeline structure, sends to master, master stores and auto-starts pending agents. Each agent runs in its own goroutine executing its pipeline — simple setup steps thread output sequentially, map steps fan out in parallel, and loop steps iterate forever calling the claude CLI. The steer TUI connects to master, subscribes for real-time state updates (including iteration data), and presents a two-pane view with tree navigation and detail views. Steer clients can inject messages into running agent conversations with automatic reconnection and exponential backoff on disconnection. Next: remaining polish items (concurrent session consistency, terminal resize handling, TUI improvements).

---

## Completed

### Phase 0: Foundations and Reconciliation ✓
### Phase 1: Cluster State and Persistence ✓
### Phase 2: Master Process ✓
### Phase 3: Apply Client ✓
### Phase 4: Steer TUI ✓
### Phase 5: Polish and Edge Cases ✓

- **5.1 Steer inject forwarding** — DONE. Full inject pipeline works.
- **5.2 Disconnection banner + auto-reconnect** — DONE. Exponential backoff, TUI clears banner.
- **5.3 Executor.Start mutex deadlock fix** — DONE.
- **5.4 Multi-step pipeline execution** — DONE. The executor now supports the full pipeline model from the spec:
  - **Simple steps** execute once, threading output to the next step as context
  - **Map steps** split previous output into items (numbered lists, bullets, headings, paragraphs) and call the method in parallel for each item
  - **Loop steps** iterate forever with steering message injection
  - Pipeline structure travels from apply→master via `AgentDef.Pipeline` (`PipelineDef` type in `cluster/object.go`)
  - Setup step failures abort the pipeline and record the error as an iteration result
  - `Executor.SetPipeline()` caches pipeline defs; `Start()` dispatches to `runPipeline()` or legacy `runAgentLoop()`
  - `splitItems()` duplicated from `runtime/runtime.go` into executor (avoids executor→runtime import cycle)
  - `validatePipeline()` checks all referenced methods exist before launching goroutine

**Testing:** 61 total tests passing: `store_test.go` (12), `protocol_test.go` (4), `persist_test.go` (5), `server_test.go` (8), `executor_test.go` (21), `steerclient_test.go` (6), `tui_test.go` (5).

New Phase 5.4 tests:
- `TestPipelineSimpleThenLoop` — verifies setup→loop chaining, context threading, and prompt isolation
- `TestPipelineMapStep` — parallel map execution with item splitting
- `TestPipelineSimpleStepFailure` — setup failure aborts pipeline, records error
- `TestPipelineValidation` — missing method caught at start, agent reverts to pending
- `TestPipelineLoopOnlyIsEquivalentToLegacy` — single loop step matches legacy behavior
- `TestSplitItems` — numbered lists, bullets, paragraphs, single item, empty input

---

## Remaining Work

### Phase 5: Polish and Edge Cases (remaining items)
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
- **Method bodies and pipeline structure travel with AgentDef** — `AgentDef.Methods` maps method name → resolved body text. `AgentDef.Pipeline` carries step order/types as `PipelineDef`. The apply command resolves both at apply time using the registry and pipeline parser. This keeps the executor decoupled from the parser/compiler/sexp/pipeline packages.
- **Pipeline execution: setup then loop** — Simple and map steps are one-shot setup. Only the loop step's iterations are tracked in `AgentRun.Iterations`. Setup step failures abort the pipeline entirely. This matches the TUI's iteration-centric view.
- **splitItems duplication** — The `splitItems` function from `runtime/runtime.go` is duplicated in `cluster/executor.go` because the executor must not import the runtime package (which depends on pipeline and registry). This avoids an import cycle.
- **ClaudeFunc injection for testability** — `type ClaudeFunc func(ctx, prompt) (string, error)`. Production uses `runtime.CallClaudeCapture`; tests inject fakes with controllable timing and failure modes.
- **One goroutine per agent with cancellable context** — Each agent gets a context derived from the executor's root context. `Stop(name)` cancels the agent context; `StopAll()` cancels root.
- **Error recovery** — On `claude` CLI failure mid-iteration, the error is recorded in `IterationResult` and the agent continues to the next iteration. Setup step failures are different: they abort the pipeline.
- **SteerClient uses channels** — `StateCh` (buffered 16) delivers state payloads; `ErrCh` (buffered 4) delivers connection errors. Non-blocking send with drain-and-replace to avoid blocking the read goroutine.

---

*Updated 2026-02-12. Reflects completed Phases 0–5 including multi-step pipeline execution.*
