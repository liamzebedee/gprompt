It's time to implement agents.

```
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

        ## 0.9.0 (`g2b28a7`)
        * Did this
        * Changed that.

agent-builder:
    loop(build)

agent-bugfixer:
    loop(bugfix)

agent-release-manager:
    loop(releasemgmt)
```

I want to detect agent[*] blocks as agent blocks.
This is how it works:
- we want to have a terminal based ui for overviewing agents
- it consists of a tree sidebar on the left. and the current convo term for the highlighted node on the right. 

the tree does the following:

agents:
[search bar] [search-icon]

builder
    loop(build)
        iteration N
        iteration N-1
        iteration N-2
        iteration 0
bugfixer
    loop(bugfix)
        iteration N
        iteration N-1
        iteration N-2
        iteration 0
release-manager
    loop(releasemgmt)

and then on rihgt, you have a agent like harness terminal with:

chat messages 

 --------------
| input box    |
 --------------

