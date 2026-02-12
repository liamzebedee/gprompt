Anything with a “:” at top level defines a named resource.
If the body is plain text, it’s a prompt spec (a Prompt resource).
If the body is an expression like loop(...), supervise(...), it’s an orchestration spec (an Agent or Supervisor resource).

respond more simply. what first-class primitives. currently we only have implicit ones and explicit ones of markdown prompts, pipelines, loops, but no explicit "conversation" primitive for example. what do we need? 


Ideas:
[ ] NPM for prompts
    packages
    registry
[ ] deployment platform for running agents like these
[ ] isolation   
    if in a git repo, agent wokrs from a worktree
    each step of pipeline gets new filesystem snapshot (snapshot of all files and state)

    problem:
        we could run and scale up ideas
        but 
        they would overlap each other probably
[ ] Girlfriend/groupchats agent
    [ ] Has one subagent for each DM thread

Issues:
[ ] gcluster steer is unusable if master crashes. can't ctrl+c out
[x] apply should restart agents
[ ] toggles do not work when there are no children (makes sense)
[x] the general tree/content panes don't do what I want them to do
[ ] content pane should render markdown as best as possible
[ ] content pane convo output is not laid out well:
    tool uses should show claude like detail
    each message should be separated by two newlines so one newline is the spacing between them
[ ] tree pane should be 30%, content pane should be 70% width
[ ] scrolling is broken
    tree pane and content pane should have independent scroll positions
    new messages should only scroll down content pane alone
[ ] the bolding is broken for iterations - 3 iterations, shows 1st in gray (in progress one), 2nd in bold (already done thouhg?), 3rd as normal. 
[ ] the prompt pane shows promopt in S-Expr's?!? before I run `gcluster apply agents.p`
[ ] nice formatting for the tree pane labels
    agents are bold
    loop(build) where loop is one colur and build is another? 
    iteration 1 2 3 all share on colour which is relatd to the loop colour








basically

    Loop(Build)
        Convo
            Messages = [ BUILD ]
        
        claude(
            Convo(Messages)
        ) -> more Messages until claude has nothing more to say <END> and exits

        at any point, you can add messages to queue
    

        Supervisor(supervised, supervisor)
        convo = Convo(Messages[SUPERVISE])



        convo = Convo(Messages[])
        Claude(Prompt) -> (Stream[Message], Handle<SendMessage>, Stream<DoneSignal>)
        
        while !<-DoneSignal:
            Msg <- Stream[Message]
            convo.Push(Msg)