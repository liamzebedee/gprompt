Alright. Let me think about this as a language design problem, not a philosophy problem.

**The core tension you need to resolve first:**

You have two kinds of operations and the language needs to be honest about the difference.

1. **Deterministic operations** — map, filter, merge, concat, split, parse JSON, read file. These are fast, predictable, zero cost. Normal computation.
2. **LLM completions** — anything that requires inference. These are slow, expensive, nondeterministic, and the thing your whole language exists to orchestrate.

Every expression in your language evaluates to a value. But some expressions require an LLM call and some don't. The runtime needs to know which is which so it can parallelize the LLM calls and run the deterministic stuff synchronously. You don't want to accidentally serialize things that could be parallel.

**Minimal syntax. I'd propose this:**

```
# ---- Prompt definitions ----
# These are templates. They don't execute until called.
# The body is the prompt. [vars] interpolate. That's it.

quantify(question):
  Take this research question and rewrite it as a precise, 
  measurable claim.
  
  Question: [question]
  
  Return:
  - metric (what's measured, with units)
  - baseline (current number)  
  - target (goal number)
  - unknowns (what numbers you don't have)

# A prompt can specify its output shape.
# This matters because the runtime needs to know 
# whether the result is a string, a list, or a record.

decompose(question) -> list:
  Identify the 3-5 independent physical processes that 
  determine [question.metric].
  
  For each, state:
  - name
  - bottleneck (what limits its rate)  
  - rate (order of magnitude number with units)
  - near_max (true/false, is this near theoretical maximum)

criticize(conjecture, anchors) -> {objection, killed}:
  Your job is to DESTROY this conjecture.
  [conjecture]
  
  Known measurements:
  [anchors]
  
  Find the single strongest objection.
  killed = true if the objection is fatal, false if not.
```

So a prompt definition is `name(args) -> shape: body`. The shape annotation is optional, defaults to string. The shapes you need are: `string`, `list`, `record` (named fields), `bool`, `number`. The runtime uses the shape to parse the LLM output. If the shape is `list`, the runtime knows to split on newlines or parse a JSON array. If it's `record`, it looks for the named fields.

This is the first decision that matters: **the output shape is part of the prompt definition, not something you figure out after.** This is what makes everything downstream composable. You can `map` over a list because you know it's a list. You can access `.metric` on a record because you declared the fields.

**Expressions and composition:**

```
# ---- Pipelines ----
# The -> operator pipes output to input.
# This is sequential.

result = quantify("How to grow trees 100x faster") 
         -> decompose 
         -> map(anchor) 
         -> fermi(result.target)

# ---- Parallel map ----
# map() over a list dispatches all LLM calls concurrently.
# This is where your speed comes from.

anchored = map(bottlenecks, anchor)    # N parallel LLM calls
damaged = map(conjectures, criticize)  # N parallel LLM calls

# ---- Expressions inside expressions ----
# A prompt body can contain calls to other prompts.
# These evaluate before the outer prompt is sent.

fermi(target, bottlenecks, anchors):
  For each bottleneck, calculate the maximum rate if 
  only that bottleneck were removed.
  
  Bottlenecks and their anchored rates:
  [map(bottlenecks, format_with_anchors(anchors))]
  
  Target: [target]
  
  Show your work. Use units.

# The inner map() runs first (possibly parallel), 
# its results get interpolated into the prompt, 
# then the outer prompt goes to the LLM.
```

The key insight: **interpolation brackets `[expr]` are not just variable substitution. They're expression evaluation.** `[x]` substitutes a variable. `[map(items, fn)]` runs a parallel map and substitutes the result. `[filter(items, pred)]` filters and substitutes. The prompt body is a template with holes, and the holes can contain arbitrary expressions that execute before the prompt is sent.

This gives you your JSX-like tree evaluation. The runtime walks the expression tree, finds all the leaves (pure values), then works upward, evaluating expressions that depend only on resolved values, sending LLM calls in parallel where possible.

**The native vs. LLM-based question:**

Here's my concrete proposal for what's native and what's LLM:

**Native (deterministic, free, instant):**
- `map(list, fn)` — apply fn to each element. If fn is a prompt, these are parallel LLM calls. If fn is native, it's just a loop.
- `filter(list, fn)` — keep elements where fn returns true.
- `merge(record, record)` — combine two records, later values override.
- `concat(list)` — join strings.
- `split(string, delimiter)` — split string into list.
- `sort(list, key)` — sort by a field.
- `len(list)`, `first(list)`, `last(list)` — basics.
- `read(path)`, `write(path, content)` — file I/O.
- `json(string)` — parse JSON into record.
- `format(template, vars)` — string formatting.

**LLM-based (costs a call, nondeterministic):**
- Any prompt you define. That's it.

**The gray area — things that seem like they should be native but are actually LLM calls:**
- `rank(list, criteria)` — this requires judgment. It's an LLM call. Define it as a prompt.
- `extract_unknowns(text)` — requires reading comprehension. LLM call.
- `summarize(text)` — obviously LLM.

```
# rank is just a prompt that returns a sorted list
rank(items, criteria) -> list:
  Rank these items by [criteria], best first.
  [items]
  Return the items in order, one per line.

# extract_unknowns is just a prompt
extract_unknowns(analysis) -> list:
  Read this analysis and list every quantity that is 
  marked as UNKNOWN, ESTIMATED, or uncertain.
  [analysis]
  One unknown per line.
```

The principle: **if it requires judgment, it's a prompt. If it's mechanical, it's native.** The language doesn't blur this. You always know when you're spending an LLM call.

**Concurrency model:**

This is where "must be fast" gets real. The rule is simple:

1. `map(list, prompt)` dispatches all calls in parallel. If the list has 10 elements, 10 LLM calls fire simultaneously.
2. A pipeline `a -> b -> c` is sequential — each step waits for the previous.
3. Interpolation expressions within a prompt body evaluate in parallel where independent.
4. Everything else is instant (native operations).

To make this work, the runtime builds a dependency graph from your expression tree. Anything without a data dependency on anything else runs concurrently. Your `investigate` function:

```
investigate(question):
  q = quantify(question)                          # 1 LLM call
  bottlenecks = decompose(q)                       # 1 LLM call, depends on q
  anchored = map(bottlenecks, anchor)              # N parallel calls
  caps = map(bottlenecks, fermi_one(anchored))     # N parallel calls, depends on anchored
  binding = rank(caps, "lowest ceiling")           # 1 LLM call
  ideas = conjecture(q.target, binding, anchored)  # 1 LLM call
  damaged = map(ideas, criticize(anchored))        # N parallel calls
  survivors = filter(damaged, x -> not x.killed)   # instant
  unknowns = map(damaged, extract_unknowns)        # N parallel calls
  -> {survivors, unknowns, anchored, binding}
```

The dependency graph looks like:

```
quantify ─── decompose ─┬─ map(anchor) ─┬─ map(fermi) ─── rank ─── conjecture ─── map(criticize) ─┬─ filter
                         │               │                                                          ├─ map(extract)
                         │               │                                                          └─ output
```

The runtime sees this and knows: `map(anchor)` can't start until `decompose` finishes, but once `decompose` returns 5 bottlenecks, all 5 `anchor` calls fire in parallel. Then all 5 `fermi` calls fire in parallel. Then `rank` runs on the results. Then `conjecture`. Then all N `criticize` calls in parallel. 

Total wall-clock time: roughly 7 sequential LLM call latencies (quantify, decompose, anchor-batch, fermi-batch, rank, conjecture, criticize-batch), regardless of how many bottlenecks or conjectures there are. That's maybe 30-45 seconds with a fast API. Not bad for a full investigation pass.

**Looping:**

```
# The loop construct re-runs until a condition is met.
# Each iteration gets the previous output as input.

deep_investigate(question):
  state = investigate(question)
  loop(state, max=5):
    new_anchors = map(state.unknowns, anchor)
    state.anchored = merge(state.anchored, new_anchors)
    repaired = map(state.survivors, repair(state.anchored))
    re_damaged = map(repaired, criticize(state.anchored))
    state.survivors = filter(re_damaged, x -> not x.killed)
    state.unknowns = map(re_damaged, extract_unknowns)
    break if len(state.unknowns) == 0
    -> state
```

`loop` is a native construct. It re-runs the body, feeding the output back as input. `break if` is the termination condition. `max=5` is a hard cap. Each iteration is another round of the conjecture-criticism cycle.

**The minimal viable language, summarized:**

You need exactly these constructs:

1. **Prompt definition:** `name(args) -> shape: body with [interpolation]`
2. **Pipe:** `expr -> prompt` (sequential composition)
3. **Map:** `map(list, fn)` (parallel when fn is a prompt)
4. **Filter:** `filter(list, predicate)`
5. **Loop:** `loop(state, max=N): body with break condition`
6. **Assignment:** `x = expr`
7. **Field access:** `x.field`
8. **Lambda:** `x -> expr` (for inline filter predicates)
9. **Output:** `-> expr` (return value of a block)
10. **Interpolation:** `[expr]` inside prompt bodies evaluates expr

That's 10 constructs. Everything else is either a native function (merge, concat, sort, len, read, write) or a user-defined prompt. The language is small. The power is in composition.

**What I'd skip for now:**

- Type checking. Just let it fail at runtime if the shape is wrong. You can add types later.
- Error handling. If an LLM call returns garbage that doesn't match the expected shape, retry once, then fail with the raw output so you can debug.
- Agents. Your agent-builder/bugfixer pattern is just `loop(prompt)`. You don't need special agent syntax.
- Shapes of thought. These are just prompt prefixes. `@conversational` is syntactic sugar for prepending text to a prompt body. Nice to have, not core.

**What I'd build first:**

The parallel `map` over LLM calls. That's the single thing that makes this language faster than doing it by hand. If `map(bottlenecks, anchor)` fires 5 API calls in parallel instead of sequentially, you've already justified the language's existence. Everything else can be rough.