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
