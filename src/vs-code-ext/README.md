# P Language — VS Code Extension

Syntax highlighting for `.p` files.

## Install

1. Open VS Code
2. `Ctrl+Shift+P` → **"Extensions: Install from VSIX..."**
3. Select `src/vs-code-ext/p-lang-0.1.0.vsix`

### Rebuilding the VSIX

```bash
cd src/vs-code-ext
npx @vscode/vsce package
```
