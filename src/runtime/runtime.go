package runtime

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"p2p/debug"
)

// Execute sends a compiled prompt to the claude CLI.
func Execute(prompt string) error {
	debug.LogPrompt("EXEC", 1, prompt)
	_, err := callClaude(prompt)
	return err
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
