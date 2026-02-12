package debug

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var Enabled bool
var mu sync.Mutex

var totalIn int64
var totalOut int64
var totalCost float64
var calls int

var recentLines [3]string
var lineCount int
var footerReserved bool

func termHeight() int {
	_, h, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || h < 10 {
		return 40
	}
	return h
}

func ensureFooterSpace() {
	if footerReserved {
		return
	}
	footerReserved = true
	// push existing content up by printing 4 blank lines
	fmt.Fprint(os.Stderr, "\n\n\n\n")
}

func drawFooter(tokenLine string) {
	ensureFooterSpace()
	h := termHeight()
	row := h - 3

	fmt.Fprint(os.Stderr, "\0337") // save cursor

	for i := 0; i < 3; i++ {
		idx := lineCount - 3 + i
		line := ""
		if idx >= 0 {
			line = recentLines[idx%3]
		}
		fmt.Fprintf(os.Stderr, "\033[%d;1H\033[2K\033[2m  %s\033[0m", row+i, truncate(line, 76))
	}
	fmt.Fprintf(os.Stderr, "\033[%d;1H\033[2K%s", row+3, tokenLine)

	fmt.Fprint(os.Stderr, "\0338") // restore cursor
}

func clearFooter() {
	if !footerReserved {
		return
	}
	h := termHeight()
	row := h - 3
	for i := 0; i < 4; i++ {
		fmt.Fprintf(os.Stderr, "\033[%d;1H\033[2K", row+i)
	}
	footerReserved = false
}

// Cleanup clears the footer. Call from main on exit.
func Cleanup() {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	clearFooter()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// StreamText feeds streaming text deltas into the preview ring buffer.
func StreamText(text string) {
	if !Enabled {
		return
	}
	parts := strings.Split(text, "\n")
	mu.Lock()
	for i, part := range parts {
		if i == 0 && lineCount > 0 {
			recentLines[(lineCount-1)%3] += part
		} else {
			recentLines[lineCount%3] = part
			lineCount++
		}
	}
	mu.Unlock()
}

// UpdateTokens redraws the footer with live in-flight token counts.
func UpdateTokens(inTok, outTok int64) {
	if !Enabled {
		return
	}
	mu.Lock()
	line := fmt.Sprintf("[tokens] call %-2d  in:%-6d  out:%-6d | total: %-6d in  %-6d out",
		calls+1, inTok, outTok, totalIn+inTok, totalOut+outTok)
	drawFooter(line)
	mu.Unlock()
}

// CallEnd records final token usage and redraws the footer.
func CallEnd(in, out int64, cost float64) {
	if !Enabled {
		return
	}
	mu.Lock()
	totalIn += in
	totalOut += out
	totalCost += cost
	calls++
	recentLines = [3]string{}
	lineCount = 0
	line := fmt.Sprintf("[tokens] call %-2d  +%-5d in  +%-5d out  $%.4f | total: %-6d in  %-6d out  $%.4f",
		calls, in, out, cost, totalIn, totalOut, totalCost)
	drawFooter(line)
	mu.Unlock()
}

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
