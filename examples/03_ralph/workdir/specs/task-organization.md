# Task Organization (Filtering & Sorting)

## Purpose
A flat list becomes unusable as it grows. Users need to focus on what matters NOW — active tasks — while still being able to review completed ones. The WHY: attention is limited; the app should help direct it.

## Acceptance Criteria
- User can filter tasks by: All, Active, Completed
- Current filter is visually indicated
- Filter persists across interactions (not across reloads — that's persistence's job)
- A count of remaining (active) tasks is always visible
- User can clear all completed tasks in one action

## Edge Cases
- Filtering when list is empty (should show helpful empty state, not a blank void)
- Clearing completed when none exist (button should be hidden or disabled)
- Completing the last active task while filtering "Active" (list should empty gracefully)

## Dependencies
- Task Management (needs tasks to exist before organizing them)
- UI/UX Design (filter controls and count display)
