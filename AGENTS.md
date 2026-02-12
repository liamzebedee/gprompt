# project

## Specs

See `specs/` for detailed specifications:
- `specs/concepts/` — language spec, runtime model, gcluster concepts
- `specs/cli/` — CLI tools (gprompt, geval, gcluster)

## Project Structure

```
src/
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

## Notes

- Use `cd src && make all` instead of `go build`

- stdlib.p is embedded
- Requires `claude` CLI in PATH
