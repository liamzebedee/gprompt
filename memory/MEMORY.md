# p2p Project

## Build
- `cd src && make all` — builds all binaries to `src/bin/`
- Go module root is `src/`, not project root

## Claude CLI JSON Structure
- Token usage is nested: `usage.input_tokens`, `usage.output_tokens`, `usage.cache_creation_input_tokens`, `usage.cache_read_input_tokens`
- NOT top-level fields
