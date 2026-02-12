
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

