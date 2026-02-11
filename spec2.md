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


### Looping

Looping is one of the most useful constructs.

We might have our book but it is probably slop as it exists currently.

The core idea of looping is expressing something like:

Idea -> Specifications -> Plan -> Loop(Build)

In fact you could express this in our language as follows:

```p
# ralph1.p
dev:
    Idea -> Specifications -> Plan -> loop(Build)
```

A useful thing to do is actually run a specific stage of the workflow we're in:

```sh
gprompt -s dev.Loop(Build) dev.p
```

Anyways, what does this `loop` construct do?

`loop` basically continually executes the Build function in sequential order.

### Supervising other loops

Because models are sometimes dumb, we often times want to have some supervision in the loop. This can be done by another agent actually. 

How could we express this in P?

```
# ralph2.p
dev:
    Idea -> Specifications -> Plan -> supervise-build(loop(Build))

supervise-build:
    agent(You are an agent in charge of supervising a build loop. Each iteration, make notes. Ensure the build loop continues in a productive path and does not loop indefinitely or see-saw.)
```

 



