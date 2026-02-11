package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"p2p/debug"
	"p2p/pipeline"
	"p2p/registry"
)

// Execute sends a compiled prompt to the claude CLI (streaming to stdout).
func Execute(prompt string) error {
	debug.LogPrompt("EXEC", 1, prompt)
	_, err := callClaude(prompt)
	return err
}

// ExecutePipeline runs a multi-step pipeline, calling claude for each step.
func ExecutePipeline(p *pipeline.Pipeline, args map[string]string, reg *registry.Registry, preamble string) error {
	context := make(map[string]string)

	// Seed context with initial input from args (if any)
	if p.InitialInput != "" {
		initialValue, ok := args[p.InitialInput]
		if !ok {
			return fmt.Errorf("pipeline initial input %q not found in args", p.InitialInput)
		}
		context[p.InitialInput] = initialValue
		debug.Log("pipeline: initial input %q = %q", p.InitialInput, initialValue)
	}

	// Preamble goes on the initial context stack
	var prevOutput string
	if preamble != "" {
		prevOutput = preamble
		debug.Log("pipeline: preamble = %q", preamble)
	}

	for i, step := range p.Steps {
		stepNum := i + 1
		isLast := i == len(p.Steps)-1

		switch step.Kind {
		case pipeline.StepSimple:
			method := reg.Get(step.Method)
			if method == nil {
				return fmt.Errorf("step %d: unknown method %q", stepNum, step.Method)
			}

			// Build prompt: interpolate params from context, prepend previous output
			prompt := method.Body
			for _, param := range method.Params {
				if val, ok := context[param]; ok {
					prompt = strings.ReplaceAll(prompt, "["+param+"]", val)
				}
			}
			if prevOutput != "" {
				prompt = prevOutput + "\n\n" + prompt
			}

			debug.LogPrompt(fmt.Sprintf("PIPELINE STEP %d: %s (%s)", stepNum, step.Label, step.Method), stepNum, prompt)

			var result string
			var err error
			if isLast {
				result, err = callClaude(prompt)
			} else {
				result, err = callClaudeCapture(prompt)
			}
			if err != nil {
				return fmt.Errorf("step %d (%s): %w", stepNum, step.Label, err)
			}

			context[step.Label] = result
			prevOutput = result
			debug.Log("pipeline: step %d output stored as %q (%d bytes)", stepNum, step.Label, len(result))

		case pipeline.StepMap:
			method := reg.Get(step.MapMethod)
			if method == nil {
				return fmt.Errorf("step %d: unknown map method %q", stepNum, step.MapMethod)
			}

			items := splitItems(prevOutput)
			debug.Log("pipeline: map step %d split into %d items", stepNum, len(items))

			if len(items) == 0 {
				return fmt.Errorf("step %d: map got 0 items from previous output", stepNum)
			}

			results := make([]string, len(items))
			prompts := make([]string, len(items))
			for j, item := range items {
				prompts[j] = item + "\n\n" + method.Body
				debug.LogPrompt(fmt.Sprintf("PIPELINE MAP %d/%d: %s", j+1, len(items), step.MapMethod), stepNum, prompts[j])
			}

			var mu sync.Mutex
			var wg sync.WaitGroup
			var firstErr error

			for j := range items {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					result, err := callClaudeCapture(prompts[idx])

					mu.Lock()
					defer mu.Unlock()
					if err != nil && firstErr == nil {
						firstErr = fmt.Errorf("map item %d: %w", idx+1, err)
					}
					results[idx] = result
				}(j)
			}

			wg.Wait()
			if firstErr != nil {
				return fmt.Errorf("step %d (%s): %w", stepNum, step.Label, firstErr)
			}

			joined := strings.Join(results, "\n\n---\n\n")
			context[step.Label] = joined
			prevOutput = joined

			if isLast {
				fmt.Print(joined)
			}
			debug.Log("pipeline: map step %d collected %d results, stored as %q", stepNum, len(results), step.Label)

		case pipeline.StepLoop:
			method := reg.Get(step.LoopMethod)
			if method == nil {
				return fmt.Errorf("step %d: unknown loop method %q", stepNum, step.LoopMethod)
			}

			iteration := 0
			for {
				iteration++
				prompt := method.Body
				if prevOutput != "" {
					prompt = prevOutput + "\n\n" + prompt
				}

				debug.LogPrompt(fmt.Sprintf("PIPELINE LOOP %d iter %d: %s", stepNum, iteration, step.LoopMethod), stepNum, prompt)

				result, err := callClaude(prompt)
				if err != nil {
					return fmt.Errorf("step %d (%s) iter %d: %w", stepNum, step.Label, iteration, err)
				}

				context[step.Label] = result
				prevOutput = result
				debug.Log("pipeline: loop step %d iter %d complete (%d bytes)", stepNum, iteration, len(result))

				fmt.Fprintf(os.Stderr, "\n══════════════════ LOOP %d ══════════════════\n\n", iteration)
			}
		}
	}

	return nil
}

// claudeCmd builds the base claude command with flags that:
// - prevent project context (AGENT.md) from leaking via --system-prompt ""
// - bypass all permission checks so tools (file read/write) execute without prompting
func claudeCmd(extraArgs ...string) *exec.Cmd {
	args := []string{"-p", "--system-prompt", "", "--dangerously-skip-permissions"}
	if m := os.Getenv("MODEL"); m != "" {
		args = append(args, "--model", m)
	} else {
		args = append(args, "--model", "claude-opus-4-6")
	}
	args = append(args, extraArgs...)
	return exec.Command("claude", args...)
}

// callClaude runs claude -p, streaming output to stdout and capturing it.
func callClaude(prompt string) (string, error) {
	cmd := claudeCmd()
	cmd.Stdin = strings.NewReader(prompt)

	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// callClaudeCapture runs claude -p, capturing output silently (no stdout streaming).
func callClaudeCapture(prompt string) (string, error) {
	cmd := claudeCmd()
	cmd.Stdin = strings.NewReader(prompt)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

// callClaudeJSON runs claude -p --output-format json and extracts the result field.
func callClaudeJSON(prompt string) (string, error) {
	cmd := claudeCmd("--output-format", "json")
	cmd.Stdin = strings.NewReader(prompt)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var resp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		// Fallback: return raw output if JSON parse fails
		return strings.TrimSpace(buf.String()), nil
	}

	return strings.TrimSpace(resp.Result), nil
}

// splitItems splits text into items using heuristics:
// tries numbered lists, markdown headings, bullet points, then paragraphs.
func splitItems(text string) []string {
	lines := strings.Split(text, "\n")

	// Try numbered list (e.g., "1. ", "2. ")
	var numbered []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 2 && trimmed[0] >= '1' && trimmed[0] <= '9' && (strings.HasPrefix(trimmed[1:], ". ") || (len(trimmed) > 3 && trimmed[1] >= '0' && trimmed[1] <= '9' && strings.HasPrefix(trimmed[2:], ". "))) {
			if current.Len() > 0 {
				numbered = append(numbered, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 {
			current.WriteString("\n" + trimmed)
		}
	}
	if current.Len() > 0 {
		numbered = append(numbered, strings.TrimSpace(current.String()))
	}
	if len(numbered) >= 2 {
		return numbered
	}

	// Try markdown headings (## or #)
	var headingSections []string
	current.Reset()
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if current.Len() > 0 {
				headingSections = append(headingSections, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 {
			current.WriteString("\n" + line)
		}
	}
	if current.Len() > 0 {
		headingSections = append(headingSections, strings.TrimSpace(current.String()))
	}
	if len(headingSections) >= 2 {
		return headingSections
	}

	// Try bullet points (- or *)
	var bullets []string
	current.Reset()
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if current.Len() > 0 {
				bullets = append(bullets, strings.TrimSpace(current.String()))
				current.Reset()
			}
			current.WriteString(trimmed)
		} else if current.Len() > 0 && trimmed != "" {
			current.WriteString("\n" + trimmed)
		}
	}
	if current.Len() > 0 {
		bullets = append(bullets, strings.TrimSpace(current.String()))
	}
	if len(bullets) >= 2 {
		return bullets
	}

	// Fallback: split on double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")
	var result []string
	for _, p := range paragraphs {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	if len(result) >= 2 {
		return result
	}

	// Last resort: return the whole thing as one item
	if t := strings.TrimSpace(text); t != "" {
		return []string{t}
	}
	return nil
}
