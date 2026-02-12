gcluster apply
==============

## Purpose

The cluster needs to know what agents to run. `gcluster apply <file.p>` bridges the gap between a `.p` file on disk and the running cluster — it parses the file, extracts agent definitions, and sends them to the master. This is the only way agents enter the cluster.

The separation between authoring (the `.p` file) and deployment (`apply`) is intentional. It means you can edit a file without affecting running agents, and only push changes when ready.

## Behaviour

`gcluster apply <file.p>` parses the file, extracts all `agent-` prefixed definitions, compiles them to S-expressions, and sends them to the master.

For each agent definition:

1. Hash the S-expression to produce a stable ID.
2. Send the definition to the master.
3. If the stable ID already exists in the cluster — no-op for that agent.
4. If the agent name exists but the hash differs — create a new revision. Existing runs stay on the old revision; new runs use the new definition.
5. If the agent name is new — register it as a new cluster object in pending state.

The command exits after the master acknowledges all definitions.

## Acceptance criteria

- `gcluster apply agents.p` with three `agent-` definitions results in three cluster objects on the master.
- Applying the same file twice produces no new revisions and no errors.
- Changing one agent's body text and reapplying creates a new revision for that agent only.
- Non-agent methods (no `agent-` prefix) in the file are parsed (they may be referenced by agents) but do not become cluster objects themselves.
- The command prints a summary of what happened: how many agents created, updated, or unchanged.
- The command exits with a non-zero status if the master is unreachable.

## Edge cases

- **Master not running**: Exits with a clear error (e.g., "cannot connect to master at 127.0.0.1:43252 — is `gcluster master` running?").
- **Parse error in `.p` file**: Exits with the parse error and applies nothing. Partial application is never allowed — it's all or nothing.
- **File has no `agent-` definitions**: Succeeds with a message indicating zero agents applied. This is not an error (the file might define methods used elsewhere).
- **Agent removed from file**: Agents previously applied but absent from the current file are NOT removed from the cluster. Apply is additive only. This prevents accidental deletion from a missing line.
- **Syntax errors in one agent**: The entire apply fails. No agents from the file are sent to the master.
- **Agent references undefined method**: If `agent-builder` calls `loop(build)` but `build` is not defined in the file, this is a compile error. Apply fails.

## Dependencies

- P language parser — to extract and compile agent definitions.
- S-expression hashing — to compute stable IDs.
- Network connection to the master at `127.0.0.1:43252`.
