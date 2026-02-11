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

the language is evaluated line by line, as is most programming lanuguages

the @listify line gets the completion from previous line and listifies it. context is implicitly passed


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

```
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





