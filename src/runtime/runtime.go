package runtime

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Execute runs compiled prompt steps through the claude CLI.
// Each step is sent as a prompt. The response from each step
// becomes context for the next step.
func Execute(steps []string) error {
	context := ""
	for i, step := range steps {
		prompt := step
		if context != "" {
			prompt = context + "\n" + step
		}

		response, err := callClaude(prompt)
		if err != nil {
			return fmt.Errorf("step %d: %w", i+1, err)
		}
		context = response
	}
	return nil
}

func callClaude(prompt string) (string, error) {
	cmd := exec.Command("claude", "-p")
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
