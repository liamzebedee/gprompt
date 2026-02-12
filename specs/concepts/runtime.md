Runtime
=======

The P runtime executes compiled prompts and pipelines. It delegates to the `claude` CLI as its LLM backend.

## Execution model

A `.p` file is parsed, compiled into a plan, and executed. There are two plan types:

### Prompt plan

A flat sequence of method invocations and plain text is expanded and concatenated into a single prompt string. This string is sent to the LLM once and the response is streamed to stdout.

For example:

```yaml
@conversational
how do trees grow?
@listify(n=10)
```

Compiles to one prompt:

```
Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
how do trees grow?
Convert to 10 items.
```

### Pipeline plan

A method body containing `->` defines a pipeline. Pipelines execute steps sequentially, threading the output of each step as context into the next.

```yaml
ralph(idea):
	idea -> spec -> plan -> loop(build)
```

1. The initial input is resolved from the invocation arguments.
2. Each step calls its method with the previous step's output as context.
3. Intermediate steps capture output silently. The final step streams to stdout.

### Step types

| Step | Behaviour |
|------|-----------|
| Simple | Call the method once. Pass previous output as context. |
| `map(ref, method)` | Split the previous output into items. Call `method` once per item in parallel. Collect results. |
| `loop(method)` | Call `method` repeatedly forever. Each iteration receives the previous iteration's output. |

## Method resolution

Methods are resolved in this order:

1. Standard library (`stdlib.p`) -- loaded automatically before user code.
2. Imported files (`@file.p`) -- only method definitions are registered.
3. User-defined methods in the current file.

Later definitions shadow earlier ones.

## Parameter interpolation

Method bodies use `[param]` slots. When invoked, slots are replaced with the corresponding argument value. Unbound slots are left as-is.

Arguments can be positional or named:

```yaml
@book-idea(blockchain)       ; positional
@listify(n=10)               ; named
```

## Implicit context passing

In a prompt plan, each invocation receives accumulated output from all previous lines as context. The prompt sent to the LLM is: previous context + newline + current prompt.

## LLM backend

The runtime calls the `claude` CLI:

```
claude -p --model <MODEL>
```

`MODEL` defaults to `claude-opus-4-6` and can be overridden by the `MODEL` environment variable.

## Eval mode

When invoked with `-e`, the runtime loads method definitions from the file but executes the given expression instead. This allows testing individual prompts against a file's method registry:

```sh
gprompt examples/book/book.p -e "@book-idea(egyptian llm's)"
```
