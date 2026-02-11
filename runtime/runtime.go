package runtime

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"p2p/compiler"
)

type Runtime struct {
	context string // Accumulated output from previous lines
}

// NewRuntime creates a new runtime
func NewRuntime() *Runtime {
	return &Runtime{
		context: "",
	}
}

// Execute runs the execution plan sequentially
func (r *Runtime) Execute(plan *compiler.ExecutionPlan) error {
	for i, compiledPrompt := range plan.Prompts {
		// Build the full prompt with context
		fullPrompt := compiledPrompt.Text
		if i > 0 && r.context != "" {
			// Prepend previous context for lines after the first
			fullPrompt = r.context + "\n" + compiledPrompt.Text
		}

		// Execute and capture output
		output, err := r.callClaude(fullPrompt)
		if err != nil {
			return fmt.Errorf("error executing line %d: %w", compiledPrompt.LineNum, err)
		}

		// Stream output to stdout word-by-word
		if err := r.streamOutput(output); err != nil {
			return err
		}

		// Add a newline after each output for readability
		fmt.Println()

		// Update context for next iteration
		r.context = output
	}

	return nil
}

// callClaude invokes the claude CLI with the given prompt
func (r *Runtime) callClaude(prompt string) (string, error) {
	cmd := exec.Command("claude", "-")
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// streamOutput outputs text word-by-word to stdout
func (r *Runtime) streamOutput(text string) error {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)

	first := true
	for scanner.Scan() {
		word := scanner.Text()
		if !first {
			fmt.Print(" ")
		}
		fmt.Print(word)
		first = false
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error streaming output: %w", err)
	}

	return nil
}
