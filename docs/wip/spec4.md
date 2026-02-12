It's time to implement agents.

## Software architecture.

Everything below will form part of a Kubernetes-like architecture.

term 1
`gprompt steer agents.p`

term 2
`gprompt steer agents.p`

term 3
`gprompt steer agents.p`

each terminal you can kinda switch between agents usnig the tree pane on the left

The terminals don't actually start agents themselves. They connect to the control plane run by `gcluster`

```
gcluster
gprompt apply agents.p
``` 

Like Kubernetes, everything is declarative.

`agents.p` describes the desired set of agents.

`gprompt apply agents.p` sends these agent definitions to the cluster and applies them as real objects with stable IDs.

If `agents.p` changes later, applying again creates a new revision. Existing runs still refer to the old revision.

`gprompt steer` shows the current cluster state and lets you attach to and steer agents within it. The source of truth is gcluster, not the local file.


## UX.

### P language.

This is how the new prompt file will support native agent orchestration to start with:

```
# agents.p
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

### TUI.

`gprompt --ui agents.p` will show a new type of interface.

A terminal based ui for overviewing contexts
- it consists of a tree sidebar on the left. 
- and a view for the highlighted node on the right

There is one view for each node currently:
- AgentView for agent nodes
- LoopView for loop() nodes
- LoopIterationView for one iteration of a loop

#### Left sidebar tree pane.

**UI**:

Agents
[search bar label "Search contexts..."]

builder
    ⌄ loop(build) 
      › **iteration N**
        iteration N-1
        iteration N-2
        iteration 0
        [display maximum of latest 4 iterations here]
bugfixer
    › loop(bugfix) [iteration #4]
release-manager
    › loop(releasemgmt)

You can navigate this list using the arrow keys, incl left/right to toggle a node's children.


On the right, you have the views:

**AgentView**:

Does nothing atm

**LoopView**:

col 1: build prompt (80%)
col 2: loop stats
    iterations
    mean(duration)
    stddev(duration)
    mean(tokens)
    stddev(tokens)

**LoopIterationView**:

chat messages...

 --------------
| input box    |
 --------------

