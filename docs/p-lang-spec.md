# P Language Specification

Version: 0.1.0 — derived from the reference Go implementation.

This document specifies the syntax and semantics of P, a language for composing LLM prompts. It is intended as the authoritative source for writing a parser that compiles `.p` files into an S-expression (Lisp) intermediate representation.

---

## 1. Lexical Structure

### 1.1 Encoding

Source files are UTF-8 text. There is no encoding declaration.

### 1.2 Lines and Indentation

P is **indentation-sensitive**. Indentation MUST use **tabs** (`\t`). Spaces for indentation are a parse error.

A line is one of:

| Kind           | Recognition rule                                                                 |
|----------------|---------------------------------------------------------------------------------|
| Blank          | Contains only whitespace. Skipped at top level; preserved inside method bodies.  |
| Comment        | First non-whitespace character is `;`. Skipped entirely.                         |
| Method header  | Starts at column 0 (no leading whitespace), does not start with `@`, ends with `:`. |
| Body line      | Starts with a tab. Belongs to the most recent method header.                     |
| Execution line | Any other non-blank, non-comment top-level line. Parsed for `@`-expressions and plain text. |

### 1.3 Identifiers

Method names and parameter names are composed of:

```
identifier = letter | digit | "-" | "_"
```

Hyphens are idiomatic (e.g. `book-idea`, `flesh-out-chapter`). Names are case-sensitive.

### 1.4 Comments

Lines whose first non-whitespace character is `;` are comments and are discarded.

```
; This is a comment
```

---

## 2. Top-Level Forms

A `.p` file is an ordered sequence of **top-level forms**. There are four node types:

```
NodeType = MethodDef | Invocation | Import | PlainText
```

### 2.1 Method Definition (`MethodDef`)

```
header    ::= name ":" | name "(" params ")" ":"
params    ::= identifier ("," identifier)*
body      ::= (TAB line NEWLINE)+
method    ::= header NEWLINE body
```

The header starts at column 0, contains no leading whitespace, does not start with `@`, and ends with `:`.

**Examples:**

```
; No parameters
conversational:
	Respond conversationally, only 3 short sentences max.

; With parameters
listify(n):
	Convert to [n] items.

; With multiple parameters
spec(idea):
	Idea: [idea]
	Write specifications.
```

**Body rules:**

- Each body line starts with exactly one tab, which is stripped.
- Blank lines within a body are preserved **only if** the next non-blank line is also indented (i.e., still part of the body).
- The body ends when a non-blank, non-indented line is encountered, or at EOF.
- Body text is inherently multiline markdown. No quoting or escaping is needed.

**Parameter interpolation:**

Parameters are interpolated in the body using `[param]` syntax:

```yaml
book-idea(topic):
	We are writing a book about [topic].
```

When invoked as `@book-idea(blockchain)`, `[topic]` is replaced with `blockchain`. If a slot has no corresponding argument (e.g. `[topic]` is never bound), it is **left as-is** in the output — the literal text `[topic]` passes through unchanged.

### 2.2 Invocation (`Invocation`)

An invocation calls a defined method. It is introduced by `@` and can appear anywhere on an execution line.

```
invocation ::= "@" name
             | "@" name "(" args ")"
```

**Three forms:**

1. **Bare invocation** — `@name` consumes the rest of the line as trailing text:
   ```
   @conversational
   ```

2. **Invocation with arguments** — `@name(arg1, arg2)`:
   ```
   @book(blockchain)
   @listify(n=10)
   ```

3. **Invocation with trailing text** — `@name rest of line`:
   ```
   @conversational how do trees grow?
   ```
   Here `how do trees grow?` becomes the `Trailing` text appended to the method body.

**Argument binding:**

Arguments are either positional or named (`key=value`):

```
args ::= arg ("," arg)*
arg  ::= value              ; positional
       | key "=" value      ; named (keyword)
```

Positional arguments bind to parameters in declaration order. Named arguments bind by key. Both forms may be mixed:

```yaml
@book(blockchain)        ; positional: topic = "blockchain"
@listify(n=10)           ; named: n = "10"
```

### 2.3 Import (`Import`)

An import loads method definitions from another `.p` file. Only `MethodDef` nodes from the imported file are registered; invocations and plain text in the imported file are ignored.

```
import ::= "@" filepath
```

Where `filepath` ends with `.p` and contains no parentheses:

```
@stdlib.p
@../shared/helpers.p
```

Imports are resolved relative to the directory of the importing file.

### 2.4 Plain Text (`PlainText`)

Any text on an execution line that is not an `@`-expression is plain text. It is included verbatim in the compiled prompt.

```
how do trees grow?
```

Multiple node types can appear on the same line:

```
@conversational how do trees grow? @listify(n=10)
```

This parses as: `Invocation(conversational)`, `PlainText("how do trees grow?")`, `Invocation(listify, [n=10])`.

Wait — the actual parser treats `@name rest` as consuming the rest of the line as trailing for the invocation. Let's be precise:

**Inline parsing rules for a single line:**

1. Scan left to right for `@`.
2. Text before `@` is `PlainText`.
3. After `@`:
   - If the next token ends with `.p` and has no parens → `Import`.
   - If `(` follows the name (no space between name and paren) → `Invocation` with args. Consume through `)`. Continue scanning remainder of line.
   - Otherwise → `Invocation` with bare name. Everything after the first space becomes `Trailing` text. **Scanning stops** (trailing consumes rest of line).

---

## 3. Pipelines

A method whose body contains ` -> ` or starts with `loop(` or `map(` is a **pipeline method**. Pipelines define a directed dataflow graph of prompt steps.

### 3.1 Pipeline Syntax

```
pipeline     ::= initial_input (" -> " step)+
               | step                              # bare loop/map with no initial input

initial_input ::= identifier                       # name of the pipeline parameter that seeds the flow

step         ::= bare_step | labeled_step

bare_step    ::= identifier                        # label = method = the identifier
               | "loop(" method ")"                # infinite loop
               | "map(" ref "," method ")"         # parallel map

labeled_step ::= label " (" method ")"             # simple step with explicit method
               | label " (loop(" method "))"       # labeled loop
               | label " (map(" ref "," method "))"# labeled map
```

Note the **space before `(`** in labeled steps: `brief (book-idea)` — the space distinguishes `label (method)` from `name(args)`.

### 3.2 Pipeline Examples

**Simple pipeline (ralph):**

```yaml
ralph(idea):
	idea -> spec -> plan -> loop(build)
```

Parses as:

```
InitialInput: "idea"
Steps:
  1. Simple  { label: "spec",  method: "spec"  }
  2. Simple  { label: "plan",  method: "plan"  }
  3. Loop    { label: "build", loop_method: "build" }
```

**Complex pipeline (book):**

```yaml
book(topic):
	topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)
```

Parses as:

```
InitialInput: "topic"
Steps:
  1. Simple  { label: "brief",           method: "book-idea"             }
  2. Simple  { label: "chapter-outline",  method: "generate-chapter-index" }
  3. Map     { label: "chapters",         map_ref: "chapters", map_method: "flesh-out-chapter" }
  4. Simple  { label: "final",            method: "concat"               }
```

**Bare loop (no initial input):**

```yaml
joker:
	loop(joke)
```

Parses as:

```
InitialInput: ""
Steps:
  1. Loop { label: "joke", loop_method: "joke" }
```

### 3.3 Agents

A method whose name is prefixed with `agent-` is an **agent definition**. The suffix after `agent-` becomes the agent's name. Agents are concurrent — when multiple agent definitions exist in a file, they run in parallel.

```yaml
agent-builder:
	loop(build)

agent-bugfixer:
	loop(bugfix)

agent-release-manager:
	loop(releasemgmt)
```

Here `agent-builder` defines an agent named `builder` that loops `build`, `agent-bugfixer` defines an agent named `bugfixer` that loops `bugfix`, etc.

Agent bodies are not restricted to `loop()` — they can contain any valid method body (plain prompt text, pipelines, etc.). The `agent-` prefix signals to the runtime that these should be spawned concurrently rather than invoked sequentially.

### 3.4 Step Kinds

| Kind     | Semantics |
|----------|-----------|
| `Simple` | Call `method` once. Pass previous output as context. Store result as `label`. |
| `Map`    | Split previous output into items (heuristic: numbered list, headings, bullets, paragraphs). Call `method` once per item in parallel. Collect results. |
| `Loop`   | Call `method` repeatedly forever. Each iteration receives the previous iteration's output. |

### 3.4 Pipeline Execution Model

1. `initial_input` is looked up in the invocation arguments to seed `context[initial_input]`.
2. Steps execute sequentially (except `map` items, which are parallel).
3. Each step receives the previous step's output as context prepended to its prompt.
4. Each step's result is stored in `context[label]`.
5. The last step streams to stdout; intermediate steps capture silently.

---

## 4. Compilation Model

Compilation takes parsed nodes and produces a **Plan**:

```
Plan = PromptPlan { prompt: string }
     | PipelinePlan { pipeline: Pipeline, args: map, preamble: string }
```

### 4.1 Registration Phase

All `MethodDef` nodes are registered in a **Registry** (a name → method map). Registration also occurs for:
- The **standard library** (`stdlib.p`), loaded automatically before user code.
- **Imported files** (`@file.p`), which contribute only their method definitions.

### 4.2 Expansion Phase

Non-method, non-import nodes (invocations and plain text) form the **execution nodes**.

If an invocation refers to a pipeline method, it becomes the plan's pipeline. All other execution nodes are **expanded** (invocations replaced with their method bodies, with arguments interpolated) and concatenated with newlines to form the prompt or preamble.

If no pipeline is found, the result is a simple `PromptPlan`.

### 4.3 Inline Pipeline Syntax

`@loop(method)` and `@map(ref, method)` at the invocation site (not inside a method body) are recognized as inline pipelines without needing a named pipeline method.

---

## 5. Standard Library

The file `stdlib.p` is loaded automatically. It ships with the runtime and defines reusable prompt methods. Current contents:

```yaml
conversational:
	Respond conversationally, only 3 short sentences max, and keep it
	light, not dense. Do not respond with bulk text unless I ask for
	detail. We're just talking.

listify(n):
	Convert to [n] items.
```

User code can shadow stdlib methods by redefining them.

---

## 6. Execution Runtime

### 6.1 CLI Interface

```
gprompt [-d] [-e expr] <file.p>
```

| Flag    | Effect |
|---------|--------|
| `-d`    | Enable debug logging to stderr. |
| `-e`    | Evaluate `expr` as the execution nodes instead of the file's own. Methods from the file are still registered. |

### 6.2 Backend

The runtime delegates to the `claude` CLI:

```
claude -p --system-prompt "" --dangerously-skip-permissions --model <MODEL>
```

`MODEL` defaults to `claude-opus-4-6` and can be overridden by the `MODEL` environment variable.

---

## 7. Target Lisp IR

This section defines the S-expression representation that a `.p` parser should emit.

### 7.1 Method Definition

```lisp
(defmethod name (param1 param2 ...)
  "body text with [param1] interpolation slots")
```

### 7.2 Pipeline Method

```lisp
(defpipeline name (param1)
  (pipeline param1
    (step "label1" (call method1))
    (step "label2" (call method2))
    (step "label3" (map ref method3))
    (step "label4" (loop method4))))
```

### 7.2.1 Agent Definition

A method named `agent-<name>` emits a `defagent` form. The `agent-` prefix is stripped to produce the agent name:

```lisp
(defagent "builder"
  (pipeline
    (step "build" (loop build))))
```

### 7.3 Invocation

```lisp
;; bare
(invoke name)

;; with positional args
(invoke name "arg1" "arg2")

;; with keyword args
(invoke name :key1 "val1" :key2 "val2")

;; with trailing text
(invoke name :trailing "rest of line")
```

### 7.4 Import

```lisp
(import "path/to/file.p")
```

### 7.5 Plain Text

```lisp
(text "how do trees grow?")
```

### 7.6 Complete Example

Source (`y.p`):

```yaml
@conversational
how do trees grow?
@listify(n=10)
```

Target Lisp:

```lisp
(program
  (invoke conversational)
  (text "how do trees grow?")
  (invoke listify :n "10"))
```

Source (`book.p`):

```yaml
book(topic):
	topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)

book-idea(topic):
	We are writing a book about [topic]. Generate a briefer on what it should cover and why it's good.

generate-chapter-index:
	From this briefer on a book, generate an index of chapters - 1 per line with a title. Max 5 chapters.

flesh-out-chapter:
	Expand this chapter into a title, 2 paragraphs, and conclusion.
	Save it to chapters/IDX.md

concat:
	Take all the chapters of this book in `chapters/*.md` and put it into one markdown file `book.md`, adding structure as needed.

@book(blockchain)
```

Target Lisp:

```lisp
(program
  (defpipeline book (topic)
    (pipeline topic
      (step "brief" (call book-idea))
      (step "chapter-outline" (call generate-chapter-index))
      (step "chapters" (map chapters flesh-out-chapter))
      (step "final" (call concat))))

  (defmethod book-idea (topic)
    "We are writing a book about [topic]. Generate a briefer on what it should cover and why it's good.")

  (defmethod generate-chapter-index ()
    "From this briefer on a book, generate an index of chapters - 1 per line with a title. Max 5 chapters.")

  (defmethod flesh-out-chapter ()
    "Expand this chapter into a title, 2 paragraphs, and conclusion.\nSave it to chapters/IDX.md")

  (defmethod concat ()
    "Take all the chapters of this book in `chapters/*.md` and put it into one markdown file `book.md`, adding structure as needed.")

  (invoke book "blockchain"))
```

Source (`joker.p`):

```yaml
joker:
	loop(joke)

joke:
	Tell a knock-knock joke and write it to jokes.txt.

@joker
```

Target Lisp:

```lisp
(program
  (defpipeline joker ()
    (pipeline
      (step "joke" (loop joke))))

  (defmethod joke ()
    "Tell a knock-knock joke and write it to jokes.txt.")

  (invoke joker))
```

Source (`agents.p` — concurrent agents):

```yaml
build:
    Read BACKLOG.md, pick one item, build it out, git commit, then mark as complete.

bugfix:
    Read BUG_BACKLOG.md, pick one item, identify root cause, write unit test, implement fix, git commit, then mark as complete.

releasemgmt:
    Your job is to update changelog.md for any new changes.

    changelog.md contains a list of changes like the following:
        # Changelog.
        ## 1.0.0 (`6abfe2`)
        * Did this
        * Changed that.

agent-builder:
    loop(build)

agent-bugfixer:
    loop(bugfix)

agent-release-manager:
    loop(releasemgmt)
```

Target Lisp:

```lisp
(program
  (defmethod build ()
    "Read BACKLOG.md, pick one item, build it out, git commit, then mark as complete.")

  (defmethod bugfix ()
    "Read BUG_BACKLOG.md, pick one item, identify root cause, write unit test, implement fix, git commit, then mark as complete.")

  (defmethod releasemgmt ()
    "Your job is to update changelog.md for any new changes.\n\nchangelog.md contains a list of changes like the following:\n    # Changelog.\n    ## 1.0.0 (`6abfe2`)\n    * Did this\n    * Changed that.")

  (defagent "builder"
    (pipeline
      (step "build" (loop build))))

  (defagent "bugfixer"
    (pipeline
      (step "bugfix" (loop bugfix))))

  (defagent "release-manager"
    (pipeline
      (step "releasemgmt" (loop releasemgmt)))))
```

---

## 8. Grammar (EBNF)

```ebnf
program        = { top_level_form } ;

top_level_form = method_def
               | exec_line
               | comment
               | blank_line ;

comment        = ";" { any_char } newline ;

blank_line     = { whitespace } newline ;

method_def     = method_header newline method_body ;

method_header  = identifier [ "(" param_list ")" ] ":" ;

param_list     = identifier { "," identifier } ;

method_body    = body_line { body_line | body_blank } ;

body_line      = TAB { any_char } newline ;

body_blank     = newline ;  (* only if followed by another body_line *)

exec_line      = { exec_element } newline ;

exec_element   = import | invocation | plain_text ;

import         = "@" filepath ;  (* filepath ends with ".p", no parens *)

invocation     = "@" identifier [ "(" arg_list ")" ]
               | "@" identifier " " trailing_text ;

arg_list       = arg { "," arg } ;

arg            = identifier "=" value    (* named *)
               | value ;                 (* positional *)

value          = { any_char - "," - ")" } ;

trailing_text  = { any_char } ;  (* consumes rest of line *)

plain_text     = { any_char - "@" } ;

identifier     = ( letter | digit | "-" | "_" )+ ;

filepath       = { any_char - whitespace }+ ".p" ;
```

---

## 9. Semantic Summary

| Concept | Syntax | Semantics |
|---------|--------|-----------|
| Define a reusable prompt | `name(params):` + indented body | Registers a method in the registry |
| Call a prompt | `@name` or `@name(args)` | Expands the method body with args interpolated |
| Import definitions | `@file.p` | Loads methods from another file |
| Sequential pipeline | `a -> b -> c` | Execute prompts in order, threading output |
| Parallel map | `map(ref, method)` | Split output into items, process each in parallel |
| Infinite loop | `loop(method)` | Repeat a prompt step indefinitely |
| Parameter slots | `[param]` in body | Replaced with argument value at expansion time. If no value is bound for `param`, the slot is left verbatim as `[param]` in the output. |
| Concurrent agent | `agent-name:` + body | Defines a named agent that runs concurrently. Body can be any valid method body. |
| Trailing text | `@name rest of line` | Appended to method body |
| Plain text | Bare text on exec line | Included verbatim in final prompt |
