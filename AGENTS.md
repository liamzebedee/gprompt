# gprompt Language Interpreter

## Specs

See `specs/` for detailed specifications:
- `specs/concepts/` — language spec, runtime model, gcluster concepts
- `specs/cli/` — CLI tools (gprompt, geval, gcluster)

## Project Structure

```
src/
├── go.mod                # Go module `p2p` (module root)
├── Makefile              # Build system
├── stdlib/               # Shared embedded stdlib
│   ├── stdlib.go         # Exports stdlib.Source
│   └── stdlib.p          # Standard library methods
├── parser/parser.go      # AST parser with file imports
├── compiler/compiler.go  # Compilation logic
├── runtime/runtime.go    # Execution runtime
├── registry/registry.go  # Method registry
├── cmd/
│   ├── gprompt/main.go   # Prompt interpreter
│   ├── geval/main.go     # S-expression evaluator
│   └── gcluster/main.go  # Cluster management (apply, run, steer)
└── bin/                  # Compiled binaries (after build)
```

## Build

```bash
cd src && make          # Compiles all commands to src/bin/
cd src && make clean    # Removes binaries
```

- All commands: `src/cmd/<name>/main.go` → `src/bin/<name>`
- Imports: `p2p/parser`, `p2p/compiler`, `p2p/registry`, `p2p/runtime`, `p2p/stdlib`

## Architecture

**4-Stage Pipeline:** `.p file → Parser → Registry → Compiler → Runtime → Output`

1. **Parse** — Convert .p file to AST, resolve `@file.p` imports
2. **Register** — Load stdlib + custom method definitions
3. **Compile** — Resolve `@method` calls, interpolate `[param]` placeholders
4. **Execute** — Run compiled prompts via `claude` CLI with implicit context passing

## Notes

- stdlib.p is embedded via `p2p/stdlib`; also searched on disk relative to input file, CWD, and binary location
- Requires `claude` CLI in PATH
