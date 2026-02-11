# Persistence (Local Storage)

## Purpose
Todos are useless if they vanish on reload. Users expect their data to survive browser restarts without accounts or servers. The WHY: trust — if the app loses data once, users won't come back.

## Acceptance Criteria
- All tasks and their states survive a full page reload
- Data is stored in localStorage as JSON
- App loads and renders saved tasks on startup
- Corrupted/invalid storage data does not crash the app (graceful fallback to empty list)
- Every mutation (add, edit, delete, toggle) persists immediately

## Edge Cases
- localStorage is full (quota exceeded — should warn, not crash)
- localStorage is disabled/unavailable (app should still work for the session)
- Another tab modifies storage (not required to sync, but should not corrupt)
- Stored data has unexpected shape (migration/validation on load)

## Dependencies
- Task Management (defines the data shape to persist)
