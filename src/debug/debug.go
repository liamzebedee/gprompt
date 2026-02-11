package debug

import (
	"fmt"
	"os"
	"strings"
)

var Enabled bool

func Log(format string, args ...any) {
	if !Enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}

func LogPrompt(label string, step int, prompt string) {
	if !Enabled {
		return
	}
	sep := strings.Repeat("─", 60)
	fmt.Fprintf(os.Stderr, "[debug] ┌%s\n", sep)
	fmt.Fprintf(os.Stderr, "[debug] │ STEP %d %s\n", step, label)
	fmt.Fprintf(os.Stderr, "[debug] ├%s\n", sep)
	for _, line := range strings.Split(prompt, "\n") {
		fmt.Fprintf(os.Stderr, "[debug] │ %s\n", line)
	}
	fmt.Fprintf(os.Stderr, "[debug] └%s\n", sep)
}
