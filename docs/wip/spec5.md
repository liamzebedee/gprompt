Anything with a “:” at top level defines a named resource.
If the body is plain text, it’s a prompt spec (a Prompt resource).
If the body is an expression like loop(...), supervise(...), it’s an orchestration spec (an Agent or Supervisor resource).

respond more simply. what first-class primitives. currently we only have implicit ones and explicit ones of markdown prompts, pipelines, loops, but no explicit "conversation" primitive for example. what do we need? 