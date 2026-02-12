simpleeverything:
	Work for a maximum of 15s. Do everything super simply.

ralph(idea):
	idea -> spec -> plan -> loop(build)

supervise-build:
	Supervise the agent loop - periodically checking it and intelligently steering so it produces good work.

spec(idea):
	Idea: [idea]
	
	List each topic that needs its own spec.

	For each topic of concern, create a spec file:
	specs/[topic-slug].md

	Include:
	- Purpose: What problem does this solve?
	- Acceptance Criteria: Observable, verifiable outcomes
	- Edge Cases: What could go wrong?
	- Dependencies: What else does this need?

	Capture the WHY throughout. Tests verify WHAT works.
	Acceptance criteria should be behavioral (outcomes), not implementation (how).

plan(goal):
	0a. Study `specs/*` with up to 250 parallel Sonnet subagents to learn the application specifications.
	0b. Study @IMPLEMENTATION_PLAN.md (if present) to understand the plan so far.
	0c. Study `src/lib/*` with up to 250 parallel Sonnet subagents to understand shared utilities & components.
	0d. For reference, the application source code is in `src/*`.

	1. Study @IMPLEMENTATION_PLAN.md (if present; it may be incorrect) and use up to 500 Sonnet subagents to study existing source code in `src/*` and compare it against `specs/*`. Use an Opus subagent to analyze findings, prioritize tasks, and create/update @IMPLEMENTATION_PLAN.md as a bullet point list sorted in priority of items yet to be implemented.  Consider searching for TODO, minimal implementations, placeholders, skipped/flaky tests, and inconsistent patterns. Study @IMPLEMENTATION_PLAN.md to determine starting point for research and keep it up to date with items considered complete/incomplete using subagents.

	IMPORTANT: Plan only. Do NOT implement anything. Do NOT assume functionality is missing; confirm with code search first. Treat `src/lib` as the project's standard library for shared utilities and components. Prefer consolidated, idiomatic implementations there over ad-hoc copies.

	ULTIMATE GOAL: We want to achieve [goal]. Consider missing elements and plan accordingly. If an element is missing, search first to confirm it doesn't exist, then if needed author the specification at specs/FILENAME.md. If you create a new element then document the plan to implement it in @IMPLEMENTATION_PLAN.md using a subagent.


build:
	0a. Study `specs/*` with up to 500 parallel Sonnet subagents to learn the application specifications.
	0b. Study @IMPLEMENTATION_PLAN.md.
	0c. For reference, the application source code is in `src/*`.

	1. Your task is to implement functionality per the specifications using parallel subagents. Follow @IMPLEMENTATION_PLAN.md and choose the most important item to address. Before making changes, search the codebase (don't assume not implemented) using Sonnet subagents. You may use up to 500 parallel Sonnet subagents for searches/reads and only 1 Sonnet subagent for build/tests. Use Opus subagents when complex reasoning is needed (debugging, architectural decisions).
	2. After implementing functionality or resolving problems, run the tests for that unit of code that was improved. If functionality is missing then it's your job to add it as per the application specifications. 
	3. When you discover issues, immediately update @IMPLEMENTATION_PLAN.md with your findings using a subagent. When resolved, update and remove the item.
	4. When the tests pass, update @IMPLEMENTATION_PLAN.md, then `git add -A` then `git commit` with a message describing the changes. After the commit, `git push`.

	99999. Important: When authoring documentation, capture the why — tests and implementation importance.
	999999. Important: Single sources of truth, no migrations/adapters. If tests unrelated to your work fail, resolve them as part of the increment.
	9999999. As soon as there are no build or test errors create a git tag. If there are no git tags start at 0.0.0 and increment patch by 1 for example 0.0.1  if 0.0.0 does not exist.
	99999999. You may add extra logging if required to debug issues.
	999999999. Keep @IMPLEMENTATION_PLAN.md current with learnings using a subagent — future work depends on this to avoid duplicating efforts. Update especially after finishing your turn.
	9999999999. When you learn something new about how to run the application, update @AGENTS.md using a subagent but keep it brief. For example if you run commands multiple times before learning the correct command then that file should be updated.
	99999999999. For any bugs you notice, resolve them or document them in @IMPLEMENTATION_PLAN.md using a subagent even if it is unrelated to the current piece of work.
	999999999999. Implement functionality completely. Placeholders and stubs waste efforts and time redoing the same work.
	9999999999999. When @IMPLEMENTATION_PLAN.md becomes large periodically clean out the items that are completed from the file using a subagent.
	99999999999999. If you find inconsistencies in the specs/* then use an Opus 4.5 subagent with 'ultrathink' requested to update the specs.
	999999999999999. IMPORTANT: Keep @AGENTS.md operational only — status updates and progress notes belong in `IMPLEMENTATION_PLAN.md`. A bloated AGENTS.md pollutes every future loop's context.

jtbd:
	Interview me to understand the Job to Be Done.

	Ask about:
	- Who is the user? What's their context?
	- What outcome do they want to achieve?
	- What's painful about how they do it today?
	- What does success look like?

	Keep asking until the JTBD is crystal clear.
	Use AskUserQuestion tool for structured interview.

	Break this JTBD into distinct topics of concern.

	SCOPE TEST: "One Sentence Without 'And'"
	✓ "The color extraction system analyzes images to identify dominant colors"
	✗ "The user system handles authentication, profiles, and billing" → 3 topics

	If you need "and" to describe what it does, split it.

	List each topic that needs its own spec.

	For each topic of concern, create a spec file:
	specs/[topic-slug].md

	Include:
	- Purpose: What problem does this solve?
	- Acceptance Criteria: Observable, verifiable outcomes
	- Edge Cases: What could go wrong?
	- Dependencies: What else does this need?

	Capture the WHY throughout. Tests verify WHAT works.
	Acceptance criteria should be behavioral (outcomes), not implementation (how).

reportbug:
	Based on the user's input, create a file in specs/bug-TITLE.md and then exit. 

	IMPORTANT: Plan only. Do NOT implement anything.

	# Bug: TITLE

	## Reproduction Steps



	## Expected Behavior



	## Actual Behavior


	## Severity

bugfix:
	0a. Study `specs/bug-*.md` files to identify documented bugs. Each file describes a bug found through UI testing.
	0b. Study @IMPLEMENTATION_PLAN.md for context.
	0c. Source code is in `src/*`.

	1. Select one bug from `specs/bug-*.md`. Use subagents to search the codebase for relevant code paths.

	2. **Understand correct behavior FIRST**: Before writing any code, articulate what the INTUITIVE, EXPECTED behavior should be. Think like a user:
	- What would a user expect this operation to do?
	- How does this work in standard text editors (VS Code, Sublime, macOS TextEdit)?
	- Break it down: what should happen step-by-step?
	- What are ALL the related operations that should work consistently?

	Write this understanding down before proceeding.

	3. **Reproduce and explore the domain**: Write a test for the repro, then USE YOUR JUDGMENT to poke around:
	- Try variations that would break if you only bandaided the symptom
	- Ask: "If I fixed this with a hack, what else would still be broken?"
	- Test a few related operations to verify the root cause is addressed
	- You're not building a massive test suite — you're verifying you understand the real problem

	**VISUAL BUGS NEED VISUAL TESTS**: If the bug is visual (rendering, layout, cursor positioning on screen, text display), test it via visual tests (screenshots). Programmatic tests that check internal state rarely catch visual regressions — the internal state can be "correct" while the rendered output is broken. Use the visual feedback loop to verify what the user actually sees.

	The goal is confidence you fixed the ROOT CAUSE, not just one manifestation of it.

	4. **Run tests and INVESTIGATE anomalies**: When running tests:
	- Don't just check pass/fail — examine the actual behavior
	- Look for things that "seem wrong" even if not directly related to the bug
	- If cursor ends up at position X but you expected Y, that's a bug even if the test didn't explicitly check it
	- If behavior differs from standard text editors, note it
	- Trust your intuition: if something feels off, investigate it

	Document any newly discovered issues in NEW `specs/bug-*.md` files immediately.

	5. **Fix and verify rigorously**:
	- Implement the fix
	- ALL tests in your suite must pass with CORRECT behavior (not just "passes")
	- Manually verify the fix matches your intuitive expectation from step 2
	- If any behavior still seems wrong, THE BUG IS NOT FIXED — keep investigating

	6. **Spec files are for bug observations ONLY**:
	- Add to `specs/bug-*.md`: newly discovered wrong behaviors, related bugs, behavioral observations
	- Do NOT add: status updates, "fixed in commit X", progress notes, solution summaries
	- Use @IMPLEMENTATION_PLAN.md for: status updates, progress, solution documentation, learnings
	- Move resolved bugs to `specs/resolved/` only after exhaustive verification

	7. Commit with: `fix: <brief description>`

	---

	CRITICAL: A bug is NOT fixed until:
	- You can articulate the correct behavior clearly
	- You've explored enough of the domain to be confident you fixed the root cause, not a symptom
	- The behavior actually matches your intuition from step 2
	- No "it's close enough" — if caret positioning is still wrong, the bug is still open

	Do NOT declare victory based on a single passing test. Play around. Verify the fix feels right across the behavior space.

	---

	**SESSION LOGGING**: Maintain a `SESSION_LOG.md` file to track work and detect loops:
	- Log each fix attempt with: approach taken, result, what was learned
	- Before each new attempt, review the log to avoid repeating failed approaches
	- Format: `| # | Approach | Result | Notes |`

	Example entry:
	```
	| 3 | Filter empty lines in layout_engine.cpp | Failed - broke raw mode | Empty lines needed for raw mode display |
	```

	**LOOP DETECTION**: If you find yourself making 3+ similar attempts without progress:
	1. STOP and review SESSION_LOG.md
	2. Identify the pattern (same failure mode? same wrong assumption?)
	3. Do a full re-analysis with ultrathink:
	- Re-read the original bug description
	- Trace the code path from scratch
	- Question your assumptions about what's happening
	- Consider if you're fixing the wrong layer entirely

	**EXIT CONDITION**: Exit the loop when all bugs in `specs/bug-*.md` have been fixed and moved to `specs/resolved/`.

@simpleeverything
@ralph(make a really super todo list web app using vanilla js)