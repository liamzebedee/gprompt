p programming language
======================

p is a programming language/runtime for prompting

```sh
(base) liam@rand:~/Music/p2p$ cat examples/y/y.p
@conversational
how do trees grow?
@listify(n=10)
```

```sh
$ src/bin/gprompt -d examples/y/y.p
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

```
conversational:
	Respond conversationally, only 3 short sentences max, and keep it light, not dense. 
	Do not respond with bulk text unless I ask for detail. We're just talking.

listify(n):
	Convert to [n] items.

@conversational
how do trees grow? 
@listify(n=10)
```

You can define methods like the above. Methods can have args and they are interpolated using `[]` in the body.

Methods are natively multiline markdown prompts. You don't have to do anything but indent them.

A prompt file is a bit like a Python script. Where if you don't contain code within a method, it is run.

This entire block is evaluated into one prompt. This is the compiled prompt that is run:

```
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

```
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

```
book(topic):
	topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)
```

We begin with the topic for the book, generate a brief in detail, generate an outline of the book's chapters, generate the chapters by mapping over each chapter in the outline and fleshing it out, and then finally binding it all together. Note that all of these call methods we've defined, but they are secondary to the shape. 

There are some benefits to defining this workflow:

 - **Each prompt can be tested individually**. Want to improve the chapter generation? You can work on that individual prompt.
 - **The pipeline can be invoked at different stages**. You can stop the pipeline and go back if you don't like the outputs.


For example, here's how to test an individual prompt:

```
$ ./src/bin/gprompt -d examples/book/book.p -e "@book-idea(egyptian llm's)"
```

Which will output for `We are writing a book about egyptian llm's. Generate a briefer on what it should cover and why it's good.`. 

