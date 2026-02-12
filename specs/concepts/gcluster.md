gcluster
========

## Purpose

Running multiple autonomous agents from a single `.p` file requires a process that outlives any individual terminal session. Without a central owner, there is no way to track what agents are running, resume observation from another terminal, or apply updated definitions without restarting everything.

gcluster solves this by separating agent lifecycle from the terminal. A persistent control plane owns agent state, and lightweight clients connect to observe and steer. This is the same split Kubernetes uses between the API server and kubectl — the cluster is the source of truth, not the client.

## Architecture

Three components:

1. **Control plane** (`gcluster master`) — long-running process that owns cluster state.
2. **Applicator** (`gcluster apply`) — sends agent definitions to the control plane.
3. **Steering UI** (`gcluster steer`) — TUI that connects for observation and interaction.

Multiple `steer` terminals connect to the same master simultaneously. The master is the single source of truth.

## Declarative model

Agent definitions are written in `.p` files using the `agent-` prefix convention defined in the [P language spec](/specs/p-lang-spec.md). Applying a file sends those definitions to the control plane, which reconciles them into cluster objects. This is declarative — the file describes desired state, the master converges toward it.

## Stable IDs

Agent definitions are compiled to S-expressions per the P language spec. The hash of an agent's S-expression is its stable ID:

- Identical definitions always produce the same ID.
- Any change produces a new ID and a new revision.
- The control plane maps stable IDs to cluster objects.

This matters because agents accumulate state (iterations, chat history). Without stable identity, reapplying a file would lose the connection between a definition and its running instance.

## Revisions

Applying a changed definition creates a new revision. Existing runs stay attached to their original revision. This prevents a redeploy from corrupting an in-flight agent loop — the old iteration finishes against the old definition, and new iterations pick up the new one.

## Cluster objects

Each agent is stored as a cluster object with:

- **Stable ID** — S-expression hash.
- **Name** — agent name (suffix after `agent-`).
- **Definition** — the full S-expression.
- **Revision history** — ordered list of applied revisions.
- **Run state** — pending, running, or stopped.

## Network

The control plane listens on `127.0.0.1:43252` by default. All subcommands connect to this address by default.

## Acceptance criteria

- A `.p` file with three `agent-` definitions, when applied, results in three distinct cluster objects visible from any connected `steer` terminal.
- Applying the same unchanged file twice is idempotent — no new revisions are created.
- Applying a file with a modified agent definition creates a new revision for that agent only; other agents are unaffected.
- Two `steer` terminals connected to the same master see the same cluster state.
- Stopping the master and restarting it does not require reapplying definitions (state persists).

## Edge cases

- **Master not running**: `apply` and `steer` fail with a clear connection error, not a hang or cryptic message.
- **Duplicate agent names**: Two `agent-builder` blocks in the same file — the second definition wins (same shadowing semantics as method definitions in the P language).
- **Empty file**: Applying a `.p` file with no `agent-` definitions is a no-op. No agents are created or removed.
- **Agent removed from file**: If a previously applied agent is absent from a new apply, the existing cluster object is NOT removed. Removal requires an explicit action (deletion is destructive and should not happen implicitly from a missing line).
- **Hash collision**: Astronomically unlikely with SHA-256, but if it occurs, the apply should reject with an error rather than silently merging unrelated definitions.
- **Port already in use**: `gcluster master` fails with a clear error if `43252` is occupied.

## Dependencies

- P language parser — to compile `.p` files into S-expressions.
- S-expression hashing — to derive stable IDs.
- `claude` CLI — the LLM backend used by agents at execution time.
