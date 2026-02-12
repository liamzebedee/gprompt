p programming language
======================

p is a programming language/runtime for prompting

```sh
make all
export PATH=$PATH:$(pwd)/src/bin
```

```yaml
@conversational
how do trees grow?
@listify(n=10)
```

```sh
$ gprompt examples/y/y.p
Trees grow by adding new cells at their tips (apical meristems) and widening their trunks through a layer called the cambium. Here are 10 key points:

1. Seeds germinate when moisture, warmth, and light align
2. Roots push downward, anchoring the tree and absorbing water
3. Shoots grow upward toward sunlight
4. Apical meristems at branch tips drive height growth
5. The cambium layer beneath the bark adds girth each year
6. Each year's cambium growth creates a new tree ring
7. Leaves photosynthesize sunlight into sugars that fuel growth
8. Xylem carries water up; phloem carries sugars down
9. Hormones like auxin control which branches grow and which stay dormant
10. Growth slows in winter (or dry seasons) and surges in spring
```

It's my go at trying to build something like PHP was for web dev, but for prompting.

The basic idea is you define prompts in `.p` files. For example:

```yaml
conversational:
	Respond conversationally, only 3 short sentences max, and keep it light, not dense. 
	
	Do not respond with bulk text unless I ask for detail. We're just talking.

listify(n):
	Convert to [n] items.

@conversational
how do trees grow? 
@listify(n=10)
```

You can define prompts like the above. Prompts can have args and they are interpolated using `[]` in the body.

Prompts are **natively multiline** markdown prompts. You don't have to do anything but indent them.

A prompt file is a bit like a Python script. Where if you don't contain code within a method, it is run.

This entire block is evaluated into one prompt. This is the compiled prompt that is run:

```yaml
Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
how do trees grow?
Convert to 10 items.
```

You can run the prompt by using `gprompt file.p`. It uses the local `claude` CLI so no need to pipe.

`gprompt -d file.p` will print debug output, so you can see what is happening.

## Writing a book using P pipelines.

The idea behind P was kinda just making something where I could easily express the ideas I have with LLM's. 

I've used Ralph, OpenProse, OpenCode, Claude Skills, DSPY, etc. And before that, multimodal pipelines with Python and Handlebars, etc. 

The thing is - they're not really *it*, are they? They can be slow, cumbersome, and they just don't feel like **computational thought**. It's not how I would write it in my dream language, it's not intuitive enough.

**Writing a book is an interesting eval for "prompt runtimes".** Here's how it looks in P:

```yaml
# examples/book/book.p
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

P is kind of inspired by declarative programming - Kubernetes and React are both examples of this. 

Kubernetes .yaml files specify what the infrastructure should look like, rather than how to orchestrate it.  

React .tsx files specify what the UI should look like, rather than how to create it

Likewise, my attempt with P is to specify how the end result of your LLM's intelligence should look like, rather than how to achieve it. 

A book is defined by a one-way flow of prompts, some of which are executed in parallel using `map`.

```yaml
book(topic):
	topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)
```

We begin with the topic for the book, generate a brief in detail, generate an outline of the book's chapters, generate the chapters by mapping over each chapter in the outline and fleshing it out, and then finally binding it all together. Note that all of these call methods we've defined, but they are secondary to the shape. 

There are some benefits to defining this workflow:

 - **Each prompt can be tested individually**. Want to improve the chapter generation? You can work on that individual prompt.
 - **The pipeline can be invoked at different stages**. You can stop the pipeline and go back if you don't like the outputs.


For example, here's how to test an individual prompt:

```sh
$ gprompt examples/book/book.p -e "@book-idea(egyptian llm's)"
```

Which will output for `We are writing a book about egyptian llm's. Generate a briefer on what it should cover and why it's good.`. 

## Writing a Ralph loop using P.

P also includes native looping support. When do you want to use looping? Well, Ralph is a pretty interesting example.

Ralph is a way to build software automatically.

`idea -> spec -> plan -> loop(build)`

And that is exactly how it looks in P:

```yaml
ralph(idea):
	idea -> spec -> plan -> loop(build)

spec(idea):
	Think about [idea]
	Write a series of specifications
	Put them in specs/*.md

plan:
	Read specs/*.md and the existing code.
	Figure out what’s missing or wrong.
	Write a plan in `IMPLEMENTATION_PLAN.md`

build:
	Pick the top next step from `IMPLEMENTATION_PLAN.md`.
	Implement it, run tests, and commit.
	Update the plan as you learn.

@ralph(build a simple html todo list app)
```

You can read the full [ralph.p](./examples/ralph/ralph.p) for the Ralph prompts. 

**I mean, this is cool, but it's basically the same as using `./loop.sh build`**.

Well, almost. Firstly, it's nicer to organise your prompts in one file, rather than a folder full of markdown files, in my opinion. It's not like we define code as one function per file, so why do we do prompts that way?

Secondly, what if we want multiple Ralph loops concurrently? 

While I was learning ralph, I started building another ralph loop for bugfixing. This is different to a build prompt - bug fixes are mainly about reading bug reports, identifying a root cause, writing a repro as a test, and then finding out how to fix the bug.

Still, you can do that with `./loop.sh bugfix` and `./loop.sh build`

But what if you didn't have to?

## Designing agent clusters (teams)

The most annoying part of Ralph and Claude is managing your terminals and sessions.

You already have a folder full of prompts, which you have to manually copy-paste/cat into claude. If you want loops, that's another level - writing `loop.sh`. P fixes all that.

The next problem is now you have to run `./loop.sh bugfix` and `./loop.sh build` in different terminals. And restart them when you improve your prompts.

It'd be great if you could just write out the "layout" of your team and just run it. Something like: 

```yaml
agent-builder:
    loop(build)

agent-bugfixer:
    loop(bugfix)

agent-release-manager:
    loop(releasemgmt)
```

And then run `start agents.p` and it starts them all up.

Something like this:

```txt
┌───────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────┐
│ Agents                                        │                                                                          │
│ ────────────────────────────────────────────  │ ● Now let me run the next iteration.                                     │
│ [ Search agents...                      ]     │                                                                          │
│                                               │ ● Explore(Read BACKLOG.md)                                               │
│ ▾ builder                                     │   ⎿  Done (3 tool uses · 8.2k tokens · 22s)                             │
│   ▾ loop(build)                               │                                                                          │
│     ▸ **iteration 3**                         │ ● Pick item, implement, commit.                                          │
│       iteration 2                             │                                                                          │
│       iteration 1                             │ ● Task(Fix failing tests)                                                │
│       iteration 0                             │   ⎿  Done (0 tool uses · 4.1k tokens · 12s)                             │
│                                               │                                                                          │
│ ▸ bugfixer                                    │ ● Write(src/feature.go)                                                  │
│   ▸ loop(bugfix)                              │   ⎿  Wrote 41 lines to src/feature.go                                   │
│                                               │                                                                          │
│ ▸ release-manager                             │ ● Done. Summary: shipped one change, tests green.                        │
│   ▸ loop(releasemgmt)                         │                                                                          │
│                                               │                                                                          │
│                                               │ ───────────────────────────────────────────────────────────────────────  │
│                                               │ ❯ send message…                                                          │
└───────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────┘
```

On the right, you've got something new - it's a tree of contexts. And on the right, you have your regular Claude Code view. 

You can swap into any agent to intercept and steer them. 

Or you could just have a look.

Steering might happen at different levels too. Just like how software sometimes has bugs at different levels - we want to drop into different agents, subagents, and prompts.

That's why we want to see a tree of contexts, not just a list of agents.

**What's different to most approaches** is that the tree allows you to explore all parts of your autonomous loops: an agent, a build loop, individual iterations of a loop. 

For example, what if your build loop isn't working? Could we steer the loop prompt instead of just the loop iteration? Yes. We just jump one level up. 

```
┌───────────────────────────────────────────────┬──────────────────────────────────────────────────────────────────────────┐
│ Agents                                        │                                                                          │
│ ────────────────────────────────────────────  │ Prompt                                           │ Stats                  │
│ [ Search contexts...                      ]   │                                                  │                        │
│                                               │ build:                                           │ iterations      4      │
│ ▾ builder                                     │   Read BACKLOG.md, pick one item, build it out,  │ mean(duration)  38s    │
│   ▸ **loop(build)**                           │   git commit, then mark as complete.             │ stddev(duration) 9s    │
│       iteration 3                             │                                                  │ mean(tokens)    8.2k   │
│       iteration 2                             │                                                  │ stddev(tokens)  1.1k   │
│       iteration 1                             │                                                  │                        │
│       iteration 0                             │                                                  │                        │
│                                               │                                                  │                        │
│ ▸ bugfixer                                    │                                                  │                        │
│   ▸ loop(bugfix)                              │                                                  │                        │
│                                               │                                                  │                        │
│ ▸ release-manager                             │                                                  │                        │
│   ▸ loop(releasemgmt)                         │                                                  │                        │
│                                               │                                                  │                        │
│                                               │ ───────────────────────────────────────────────  │                        | 
│                                               │ ❯ edit prompt…                                                            │
└───────────────────────────────────────────────┴──────────────────────────────────────────────────────────────────────────┘
```

Here we can issue commands to edit the prompt. We might even run simulations to test the prompt is better than the new one. On the right you see performance characteristics - imagine our AI as a Kaizen manufacturing system! Number of iterations, mean duration and standard deviation of each iteration's duration.

How do we make this work? Well, we take inspiration from Kubernetes. We define a cluster of agents declaratively in source code (`.p` files). We spin them up by sending that definition to a cluster manager, which creates the agent processes and manages them. We login to the cluster and steer it by connecting to that cluster manager node. 

```sh
# Start cluster manager
gcluster master

# Deploy the agents
gcluster apply agents.p

# Login and steer
gcluster steer agents.p # term 1
gcluster steer agents.p # term 2
gcluster steer agents.p # term 3
```

If we think of another agent we need, we can simply add it to `agents.p` and re-run `gcluster apply agents.p`. It will be added to the cluster. 



## Scaling agent clusters (autonomous creation of subagents)




---

But what about supervising those loops with Claude also? 

What about UI's that show you what's going on here?

What about autoscaling agents to match workloads?

What about A/B testing prompts?

TBC