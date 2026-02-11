ralph
=====

The future of software engineering is here - model-first develpment.

This repo contains my model-driven development toolbox:

 * Ralph loop with nice logging.
 * Prompts:
   * Plan
   * Build
   * Bugfix

See [Ralph playbook](https://github.com/ClaytonFarr/ralph-playbook)

## Usage.

```sh
# Term 1: Loop.
./ralph/loop.sh build
./ralph/loop.sh fix

# Term 2: Steer.
#

# 2.1 Bugs
@ralph/PROMPT_reportbug.md text isn't rendering after scroll 
# -> creates a ticket in specs/, loop picks it up

# 2.2 Features
@ralph/PROMPT_reqs.md read
# -> interviews you on project, creates specs for build loop
```
