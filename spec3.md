desired output: `make` produces bin/gprompt from gprompt.go, running `gprompt x.p" will evaluate x.p. 

language/runtime spec: x.p is a language like python, it has one indentation which is tabs (\t). here is an example file:
```p
# lib.p
conversational:
	Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking. 

listify(n):
	Convert to [n] items.
```

methods are defined in blocks with method-name separated by dashes, optional args inside brackets, and then a single body which is all indented.

the language has an interpolation syntax. [n] is replaced by the contents of the variable n passed in the argument. 

## runtime evaluation
now time for evaluation. you can also run 
`gprompt y.p`

```
# y.p
@conversational how do trees grow?
```

which will invoke the top-level interpreter. this will first compile the code and then run it using the local LLM model available (it just runs `claude` and outputs the word-by-word output)

compilation turns `y.p` into:
```
Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
how do trees grow?      
```

you could also make it more powerful:

```
# y2.p
@conversational how do trees grow?
@listify(5)
```

this evaluates to a single prompt which includes listifying the output.

```
(base) liam@rand:~/Music/p2p$ ./src/bin/gprompt -d y.p 
[debug] parsing y.p
[debug] parsed 3 nodes
[debug]   node[0] type=1 name="conversational"
[debug]   node[1] type=3 name=""
[debug]   node[2] type=1 name="listify"
[debug] stdlib search: stdlib.p
[debug] stdlib search: stdlib.p
[debug] stdlib search: /home/liam/Music/p2p/src/bin/stdlib.p
[debug] stdlib not found on disk, using embedded
[debug] stdlib method: "conversational"
[debug] stdlib method: "listify"
[debug] compiling 3 exec nodes
[debug] ┌────────────────────────────────────────────────────────────
[debug] │ STEP 1 COMPILED
[debug] ├────────────────────────────────────────────────────────────
[debug] │ Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
[debug] │ how do trees grow?
[debug] │ Convert to 10 items.
[debug] └────────────────────────────────────────────────────────────
[debug] ┌────────────────────────────────────────────────────────────
[debug] │ STEP 1 EXEC
[debug] ├────────────────────────────────────────────────────────────
[debug] │ Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
[debug] │ how do trees grow?
[debug] │ Convert to 10 items.
[debug] └────────────────────────────────────────────────────────────
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

## Workflows

the final bit of power comes in compositional computational thinking.
say you wanted to explore a topic in depth. like build a template for a book which is about how to build a blockchain.

```
# y3.p
book(topic):
    topic -> fleshed-out-topic (details) -> book-outline (generate-outline) -> all-chapters (map(chapters, flesh-out-chapter))

detail(topic):
	We are writing a book about [topic]. Write a general outline of a good introduction to such a topic.

generate-outline:
	From this material, generate an outline for a book and save to `book-outline.md`.
	e.g.
	* Chapter 1: Title
		* Core topic sentence
		* Point 1..[3-5]
		* Conclusion
	* Chapter 2: Title
	* Chapter 3: Title

flesh-out-chapter:
	We have a book topic and outline. 
	Your job is to "flesh out" a chapter into its full contents and save it in chapters/[index].md

	Chapter: [chapter]
```

`book` declaratively defines a sort of workflow. This is how it works:

-> defines a series of steps
fleshed-out-topic, book-outline are simple labels to make the logic easy to read
details, generate-outline are the actual methods which are called
topic is the variable
`map` is a special stdlib function which maps over each chapter and fleshes it out. this invokes the runtime in parallel. chapters is a name given to the last step's output, flesh-out-chapter is the function called

This construct allows you to construct higher-level chains of prompts together, as well as run them in parallel (`map`)

## Loops and background workers.

Up until now, workloads have been short-lived with defined lifetimes. That's not AI! 

AI is supposed to be about digital things that have a mind of their own, living forever! Or something like that.

This is usually called **agents**. 

If you think of Kubernetes (which is actually Greek for cybernetics, lol), Kubernetes is kind of already a digital intelligence orchestration platform. I mean - being an Nginx worker node is not necessarily Leonardo Da Vinci, but the fact is that these Kubernetes container instances live rather independently of the actual runtime itself. When you specify, "the infrastructure shall have 20 Nginx workers", you can replace "nginx workers" with "agents" and the meaning remains largely unchanged.

So how should agents look? 

Again, let's go back to this idea of what's an interesting eval for an agent orchestration runtime? 

If you start with K8s, we might think of defining a city or a village:

```
# city.p
city:
	100 agents (generic-agent)
```

Running `gprompt city.p` would thus connect to our master, which tells us the state of the system (0 agents), and then promptly spin up 100 agents. 

What would this `generic-agent` do, exactly? One important design consideration of P was single responsibility. It's something the father of ralphing (Geoff Huntley) spoke about. Models can really focus very well at one task, and are more prone to fail when there are multiple. So what would an agent be doing? One task, I guess.

```
# city.p
generic-agent:
	task:
		talk to other agents I guess???
```

(Again we're using the beauty of P being an indentation-based language. So we can have multiline markdown prompts, for free.)

Talk to other agents - I mean, that's what you would do, right? But how? We haven't wired up our runtime's worth of agents. 

I guess one benefit of this "YAML-like" idented syntax is we could declare multiple fields:

```
# city.p
generic-agent:
	task:
		talk to other agents I guess???
	tools:
		...
	memory: ...
```

This seems cool but more of an idea tarpit. I want simplicity. 

Let's go back to designing one agent. What does our agent do? 

 - read the news
 - schedule things
 - runs in a loop
 - responds to events

Generically, we could say our agent:

 - receives information
 - acts

As two parallel things. 

This is boring. Let's consider software engineering.




How do we do Ralph-ing? 

What are the problems we are solving? 

- agents working autonomously -> loops, queues, external memory, idempotency
- agents intelligently fixing themselves -> supervision
- designing and adding concurrent loops and loops of loops 
	- build agent
	- tester agent
	- bug fixer agent

What does moltbot teach us? 
- scheduling is a must
- memory is a must
- periodic memory summarisation is a must

Docker:
- encapsulation/containerisation - each step of prompting has immutable workdir

K8's and Google:
- scaling up processes - scaling up more agents
- mapping agents onto workloads.

Erlang:
- agents communicating with other agents.
- hot code update over the wire



Ideally what you want to do: 
Agents, subagents, etc. are all tracked by the scheduler/runtime layer
Possible to introspect/pipe in to any agent in the terrain.
`gprompt apply` will apply changes.



## First idea for agents.

Let's just begin by building a writer's studio. Let's build an autonomous NY Times. That's easier than software because all we need is writing.

What might we need for a writer's studio:

- information about the world (read different websites)
- a team of 5 writers
- 
