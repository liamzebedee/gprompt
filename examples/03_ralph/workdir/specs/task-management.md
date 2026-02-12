# Task Management (CRUD)

## Purpose
Users need to create, view, edit, and delete todo items. This is the core interaction loop — without it, there is no app. The WHY: people forget things; a todo list externalizes memory.

## Acceptance Criteria
- User can add a task by typing text and submitting
- User can mark a task as complete (and undo it)
- User can edit a task's text inline
- User can delete a task permanently
- Empty task text cannot be submitted
- New tasks appear immediately in the list without page reload

## Edge Cases
- Submitting whitespace-only text (should be rejected)
- Very long task text (should be handled gracefully, not break layout)
- Rapid double-clicks on complete/delete (should not duplicate actions)
- Editing a task to empty string (should revert or be rejected)

## Dependencies
- UI/UX Design (needs a form and list rendering)
- Persistence (changes must survive page reload)
