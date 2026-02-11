# Agents

## Project

Vanilla JS todo app. Single HTML file (`src/index.html`) with all CSS/JS inline. No build step, no dependencies.

## Commands

- **Run tests**: `node --test tests/test.mjs` (from workdir)
- **View app**: Open `src/index.html` in a browser

## Structure

```
src/index.html     - Complete app (HTML + CSS + JS inline)
tests/test.mjs     - Test suite (node:test, 51 tests)
specs/             - Requirements (4 spec files)
```

## Notes

- Node v24.5.0 with built-in test runner (`node:test`)
- Tests parse HTML file and extract JS for unit testing — no browser needed
- localStorage key: `todos-app-data`
