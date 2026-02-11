package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"p2p/compiler"
)

// Runtime executes compiled prompts with implicit context passing
type Runtime struct {
	context string
}

// NewRuntime creates a new runtime with empty context
func NewRuntime() *Runtime {
	return &Runtime{
		context: "",
	}
}

// Execute runs an execution plan
func (r *Runtime) Execute(plan *compiler.ExecutionPlan) error {
	for _, prompt := range plan.Prompts {
		// Build the full prompt with context
		fullPrompt := r.context
		if fullPrompt != "" {
			fullPrompt += "\n"
		}
		fullPrompt += prompt.Prompt

		// Call claude and capture output
		output, err := r.callClaude(fullPrompt)
		if err != nil {
			return fmt.Errorf("line %d: failed to call claude: %w", prompt.LineNum, err)
		}

		// Stream the output
		r.streamOutput(output)

		// Save as context for next iteration
		r.context += "\n" + output
		r.context = strings.TrimSpace(r.context)
	}

	return nil
}

// callClaude executes the claude CLI with the given prompt
func (r *Runtime) callClaude(prompt string) (string, error) {
	cmd := exec.Command("claude", "-")
	cmd.Stderr = os.Stderr

	// Create pipe for stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Create pipe for stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, prompt)
	}()

	// Read output
	var output strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		output.WriteString(scanner.Text())
		output.WriteString("\n")
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("claude command failed: %w", err)
	}

	result := strings.TrimRight(output.String(), "\n")
	return result, nil
}

// streamOutput outputs text word-by-word to stdout
func (r *Runtime) streamOutput(text string) {
	words := strings.Fields(text)
	for i, word := range words {
		fmt.Print(word)
		if i < len(words)-1 {
			fmt.Print(" ")
		}
		// Small delay for streaming effect
		time.Sleep(time.Millisecond * 10)
	}
	fmt.Println()
}
