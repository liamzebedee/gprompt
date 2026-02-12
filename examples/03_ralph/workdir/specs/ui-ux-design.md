# UI/UX Design

## Purpose
The interface is the product. A todo app lives or dies by how fast and frictionless it feels. The WHY: if it's slower or clunkier than a sticky note, nobody will use it.

## Acceptance Criteria
- Single HTML file with inline CSS and JS (vanilla, no build step, no dependencies)
- Clean, centered layout that works on mobile and desktop
- Input field is auto-focused on page load
- Visual distinction between active and completed tasks (e.g., strikethrough)
- Hover/focus states on all interactive elements
- Keyboard accessible: Enter to submit, interactions reachable via Tab
- Responsive down to 320px viewport width

## Edge Cases
- Very long task text (should wrap, not overflow or cause horizontal scroll)
- Many tasks (100+) should not noticeably lag
- Touch targets on mobile must be at least 44px

## Dependencies
- None (this is the foundation layer)
