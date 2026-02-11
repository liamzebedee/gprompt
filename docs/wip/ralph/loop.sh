#!/bin/bash
# Usage: ./loop.sh [keyword] [max_iterations]
# Examples:
#   ./loop.sh              # Uses PROMPT_build.md, unlimited iterations
#   ./loop.sh 20           # Uses PROMPT_build.md, max 20 iterations
#   ./loop.sh plan         # Uses PROMPT_plan.md, unlimited iterations
#   ./loop.sh fix 5        # Uses PROMPT_fix.md, max 5 iterations
#   ./loop.sh reqs         # Uses PROMPT_reqs.md, unlimited iterations
# set -ex

# LLM=amp
LLM=claude

# Get the directory where this script lives
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the formatting library
source "$SCRIPT_DIR/lib/claude-format.sh"

# Parse arguments
if [[ "$1" =~ ^[0-9]+$ ]]; then
    # First arg is a number - use default "build" with max iterations
    MODE="build"
    PROMPT_FILE="ralph/PROMPT_build.md"
    MAX_ITERATIONS=$1
elif [ -n "$1" ]; then
    # First arg is a keyword
    MODE="$1"
    PROMPT_FILE="ralph/PROMPT_${1}.md"
    MAX_ITERATIONS=${2:-0}
else
    # No arguments - default to build
    MODE="build"
    PROMPT_FILE="ralph/PROMPT_build.md"
    MAX_ITERATIONS=1000
fi

ITERATION=0
CURRENT_BRANCH=$(git branch --show-current)

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Mode:   $MODE"
echo "Prompt: $PROMPT_FILE"
echo "Branch: $CURRENT_BRANCH"
[ $MAX_ITERATIONS -gt 0 ] && echo "Max:    $MAX_ITERATIONS iterations"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Verify prompt file exists
if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: $PROMPT_FILE not found"
    exit 1
fi

while true; do
    if [ $MAX_ITERATIONS -gt 0 ] && [ $ITERATION -ge $MAX_ITERATIONS ]; then
        echo "Reached max iterations: $MAX_ITERATIONS"
        break
    fi

    # Run Ralph iteration with selected prompt
    # -p: Headless mode (non-interactive, reads from stdin)
    # --dangerously-skip-permissions: Auto-approve all tool calls (YOLO mode)
    # --output-format=stream-json: Structured output for logging/monitoring
    # --model opus: Primary agent uses Opus for complex reasoning (task selection, prioritization)
    #               Can use 'sonnet' in build mode for speed if plan is clear and tasks well-defined
    # --verbose: Detailed execution logging
    if [ $LLM = "claude" ]; then
        cat "$PROMPT_FILE" | claude -p \
            --dangerously-skip-permissions \
            --output-format=stream-json \
            --model opus \
            --verbose 2>&1 | format_claude_json
    elif [ $LLM = "amp" ]; then
        cat "$PROMPT_FILE" | amp -x \
            --dangerously-allow-all \
            --stream-json 2>&1 | format_claude_json
    fi

    # Push changes after each iteration
    git push origin "$CURRENT_BRANCH" || {
        echo "Failed to push. Creating remote branch..."
        git push -u origin "$CURRENT_BRANCH"
    }

    ITERATION=$((ITERATION + 1))
    echo -e "\n\n======================== LOOP $ITERATION ========================\n"
done