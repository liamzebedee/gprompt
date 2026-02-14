## The language.

A programming language for orchestrating LLM inference. Surface syntax is readable, markdown-ish, compiles to a lisp. Prompts are first-class values with typed outputs. map is parallel. divide/conquer/combine is recursive parallel decomposition. where filters. -> pipes and returns. loop iterates with implicit state

The language is deployed to a runtime. Runtime keeps track of state. State can be introspected visually. It is all S-Expr's which map naturally onto views.

## Problems solved:

- Ralph loops, fragile, too much code
- How to elevate inside a CC session and supervise another process?
- How to write algorithms to answer q's like "how to make trees grow 100x faster"?
    - Can use CC but runs out of context. Need variables/funcs for context management
    - Can use CC but still need to design prompts. Separate functions is the right tool for designing new thought algos.
    - Can use CC but it doesn't scale beyond my computer. Can't use cloud- need tool usage (biolabs). Hence need something like a cluster. Cluster requires state management - ie. of agent sessions. Language fits in well here.
- Where to get the prompt for ralph SWE? Prompt libraries are useful. Standard language templating too.

## Why have a separate language for prompting?

1. Prompts will be the way we interact with models, 1yr, 2yr, and 5yrs from now. Language is the best interface to intelligence.
2. Models have 1MB context windows (1M tokens). This is the core bottleneck/reason we need a language:
    - Ralph loops wouldn't be needed if CC could intelligently return <|loop|>[Task]</|loop|>
    - We only need variables and functions to manage model context. If context was infinite, it'd just be one prompt and a library of tools within it.
    - Even if CC could return "special loop tokens", people would begin looking to programming loops of loops. Demand for intelligence is infinite.
    - It is likely programming will be for programming other, long-running agents themselves. Not just ephemeral CC sessions.
    - This requires (1) writing down prompts, (2) a runtime of agents, that we assume are on-premise hosted meaning need for a (3) cluster management system.
2. Local agents are necessary because Anthropic will not own biolabs. Creation of correct knowledge requires verifying it in the environment. Companies will run agents locally forever. But the models will be proprietary and hosted centrally.
    - Local agents mean clusters to manage them.
    - Clusters mean keeping track of agents and their prompts.
    - This might require its own language.
3. The models and their harness software will absorb all features related to the language. This is the prerequisite to increasing their power. Without the dataset, they cannot scale up anyways.
    - This is an important reason why the runtime requires an inherent introspective capability - ie. a debugger. 
    - Because humans offer important insight/steering that models cannot do. A human in the loop is positive sum. 
    - We don't know where they need to be in the loop. Which is why we need a standard runtime to attach a standard debugger to.
4. There may only be a small library of algorithms for learning (ASI), but there will be an infinite expressive calculus for art. Language = only 10k words needed, but we have billions of texts (books, artworks). Hence prompt libraries are useful.
    - There is an infinite amount of ideas in the latent space.

The essence is the creation of new knowledge, which requires grounding, implying tool use, implying on-premise agents.
