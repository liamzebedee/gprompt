gcluster master
===============

## Purpose

Agents need a process that outlives the terminal that started them. `gcluster master` is that process — it owns agent definitions, tracks revisions, manages execution, and serves state to any number of connected clients. Without it, there is no persistence, no multi-terminal observation, and no way to update agents without killing them.

## Behaviour

`gcluster master` starts the control plane as a foreground process.

It:

- Listens on `127.0.0.1:43252` by default.
- Accepts agent definitions from `gcluster apply` and stores them as cluster objects.
- Maintains stable IDs, revision history, and run state for each agent.
- Spawns and manages agent execution (delegates to `claude` CLI per the runtime spec).
- Serves cluster state to connected `gcluster steer` clients.
- Persists cluster state to disk so it survives restarts.

Multiple `steer` clients connect simultaneously. The master pushes state updates to all connected clients.

## Acceptance criteria

- Running `gcluster master` starts a process that listens on `127.0.0.1:43252`.
- The process accepts connections from `apply` and `steer` subcommands.
- Agent definitions received via `apply` are stored and survive a master restart.
- When an agent's run state changes (started, iteration completed, stopped), all connected `steer` clients see the update without polling.
- The master can manage at least 10 concurrent agents without degradation.
- Ctrl-C gracefully shuts down: running agents are stopped, state is persisted, clients are disconnected with a clear signal.

## Edge cases

- **Port occupied**: Exits with a clear error message naming the port and suggesting the cause (another master instance, or a different process).
- **Corrupt persisted state**: If the on-disk state is unreadable, the master starts fresh and logs a warning rather than crashing. The old state file is preserved for debugging.
- **Client disconnects abruptly**: The master cleans up the client's session without affecting agents or other clients.
- **No agents applied**: The master runs fine with zero agents — it waits for `apply`.
- **Agent execution failure**: If the `claude` CLI fails mid-iteration, the master records the error on the iteration, keeps the agent in running state, and proceeds to the next iteration (for loop agents).

## Dependencies

- Network listener on `127.0.0.1:43252`.
- Persistent storage for cluster state (location TBD).
- `claude` CLI — used to execute agent prompts.
- P language S-expression format — to store and compare agent definitions.
