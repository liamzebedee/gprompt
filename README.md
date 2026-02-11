p programming language
======================

p is a programming language/runtime for prompting

```sh
(base) liam@rand:~/Music/p2p$ cat y.p 
@conversational
how do trees grow? 
@listify(n=10)
```

```sh
$ src/bin/gprompt -d y.p
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

The basic idea is you define prompts in `.p` files. For example, this is a library `lib.p`:

```
conversational:
	Respond conversationally, only 3 short sentences max, and keep it light, not dense. 
	Do not respond with bulk text unless I ask for detail. We're just talking.

listify(n):
	Convert to [n] items.
```

You can define methods like the above. Methods can have args and they are interpolated using `[]` in the body.

Methods are natively multiline markdown prompts. You don't have to do anything but indent them.

A prompt file is a bit like a Python script. Where if you don't contain code within a method, it is run. This is how `y.p` was evaluated before, look-

```
@conversational
how do trees grow? 
@listify(n=10)
```

This entire block is evaluated into one prompt. This is the compiled prompt that is run:

```
Respond conversationally, only 3 short sentences max, and keep it light, not dense. Do not respond with bulk text unless I ask for detail. We're just talking.
how do trees grow?
Convert to 10 items.
```

You can run the prompt by using `gprompt file.p`. It uses the local `claude` CLI so no need to pipe.

`gprompt -d file.p` will print debug output, so you can see what is happening.





