1. Open Claude Code. Tell it "take the list.txt, download youtube mp3, import into quod libet". It does it. Then tell it "deploy an agent which does that". It writes agent.p, deploys it to cluster. Agent persists beyond CC session, and its in code.

2. Be running a Ralph loop in Claude Code. Then type "supervise this" and it wraps it in another CC process. Then exit and re-login. These agents are still running persistently on a cluster. With the definition in code so others can see it.

