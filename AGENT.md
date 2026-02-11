# gprompt Language Interpreter - Implementation Notes

## Project Structure

```
/home/liam/Music/p2p/
├── AGENT.md                  # Architecture documentation (this file)
├── spec.md                   # Original spec
├── spec2.md                  # Language specification
└── src/                      # Everything lives here
    ├── go.mod                # Go module definition (module root)
    ├── Makefile              # Build system
    ├── gprompt.go            # Main entry point (package main)
    ├── stdlib.p              # Standard library methods
    ├── parser/parser.go      # AST parser with file imports
    ├── compiler/compiler.go  # Compilation logic
    ├── runtime/runtime.go    # Execution runtime
    ├── registry/registry.go  # Method registry
    └── bin/
        └── gprompt           # Compiled binary (after build)
```

## Implementation Details

### Package Structure (src/ subdirectory)
- **parser/**: Converts .p files to AST, handles @file.p imports recursively
- **registry/**: Manages method definitions from stdlib.p + custom files
- **compiler/**: Resolves @method calls, interpolates [param] placeholders
- **runtime/**: Executes via `claude` CLI with implicit context passing

### Build System
- `cd src && make` - Compiles to `src/bin/gprompt`
- `cd src && make clean` - Removes binary
- Go module: `p2p` (go.mod is in `src/`)
- Imports: `p2p/parser`, `p2p/compiler`, `p2p/registry`, `p2p/runtime`

## Language Features Implemented

### 1. Method Definitions
```p
methodname(param1, param2):
	Method body with [param1] interpolation
	Can be multiple lines
```

### 2. Method Invocations
```p
@methodname arg1 arg2    # With trailing text
@methodname(arg1, arg2)  # With explicit arguments
```

### 3. File Imports
```p
@stdlib.p                # Imports methods from stdlib.p
@custom.p               # Imports methods from custom.p
```
- Detected by `.p` suffix and no parentheses
- Resolved relative to current file's directory
- Imports are compile-time only (no invocations generated)

### 4. Parameter Interpolation
```p
listify(n):
	Convert to [n] items

@listify(7)  # -> "Convert to 7 items"
```
- Simple string replacement of [paramname] with argument values

### 5. Implicit Context Passing
```p
@conversational What is a cat?
@listify(3)  # Receives previous output as context
```
- Each invocation receives accumulated output from previous lines as context
- Prompt = context + "\n" + current_prompt

## Stdlib Methods

- **conversational**: Respond in 3 sentences max, conversational tone
- **listify(n)**: Convert to n items

## Testing

```bash
cd src
make
./bin/gprompt ../y.p
```

## Architecture Overview

**4-Stage Pipeline:**
```
.p file → Parser → Registry → Compiler → Runtime → Output
```

1. **Parse** - Convert .p file to AST with imports resolved
2. **Register** - Load stdlib, register custom methods
3. **Compile** - Convert invocations to execution plan with interpolation
4. **Execute** - Run compiled prompts via claude CLI

## Known Limitations

- stdlib.p is searched relative to the input file's directory, CWD, and binary location
- Requires `claude` CLI to be available in PATH
- No error recovery - first error stops execution

## Future Enhancements

See spec2.md for full language specification. Current implementation covers core features:
- ✅ Method definitions and invocations
- ✅ Parameter interpolation
- ✅ File imports
- ✅ Implicit context passing
- ⏳ Advanced features (pipes, map/reduce, control flow) from spec2.md
