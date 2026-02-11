package debug

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var Enabled bool
var mu sync.Mutex

func Log(format string, args ...any) {
	if !Enabled {
		return
	}
	mu.Lock()
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	mu.Unlock()
}

func LogPrompt(label string, step int, prompt string) {
	if !Enabled {
		return
	}
	var b strings.Builder
	sep := strings.Repeat("─", 60)
	fmt.Fprintf(&b, "[debug] ┌%s\n", sep)
	fmt.Fprintf(&b, "[debug] │ STEP %d %s\n", step, label)
	fmt.Fprintf(&b, "[debug] ├%s\n", sep)
	for _, line := range strings.Split(prompt, "\n") {
		fmt.Fprintf(&b, "[debug] │ %s\n", line)
	}
	fmt.Fprintf(&b, "[debug] └%s\n", sep)

	mu.Lock()
	fmt.Fprint(os.Stderr, b.String())
	mu.Unlock()
}
