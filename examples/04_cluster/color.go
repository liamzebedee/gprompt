package todo

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// ColorEnabled reports whether colored output should be used.
// It returns true when stdout is a terminal (not piped/redirected).
func ColorEnabled() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// ColorStatus returns the status string wrapped in an ANSI color code
// appropriate for its value. If color is false, returns the plain string.
func ColorStatus(s Status, color bool) string {
	if !color {
		return string(s)
	}
	switch s {
	case StatusPending:
		return colorYellow + string(s) + colorReset
	case StatusInProgress:
		return colorCyan + string(s) + colorReset
	case StatusDone:
		return colorGreen + string(s) + colorReset
	default:
		return string(s)
	}
}

// ColorPriority returns the priority string wrapped in an ANSI color code.
// High priority is red+bold, medium is yellow, low is plain.
func ColorPriority(p Priority, color bool) string {
	if p == PriorityNone {
		return "-"
	}
	if !color {
		return string(p)
	}
	switch p {
	case PriorityHigh:
		return colorRed + colorBold + string(p) + colorReset
	case PriorityMedium:
		return colorYellow + string(p) + colorReset
	case PriorityLow:
		return string(p)
	default:
		return string(p)
	}
}

// ColorLabel wraps a label string in bold when color is enabled.
func ColorLabel(label string, color bool) string {
	if !color {
		return label
	}
	return colorBold + label + colorReset
}

// ColorDueDate returns the due date string, colored red+bold if overdue, yellow if
// due today, or plain otherwise.
func ColorDueDate(d DueDate, color bool) string {
	if !d.Valid {
		return "-"
	}
	s := d.String()
	if !color {
		return s
	}
	today := time.Now().Truncate(24 * time.Hour)
	due := d.Time.Truncate(24 * time.Hour)
	switch {
	case due.Before(today):
		return colorRed + colorBold + s + colorReset
	case due.Equal(today):
		return colorYellow + s + colorReset
	default:
		return s
	}
}

// Colorf formats a string with the given color code, or returns it plain if color is false.
func Colorf(color bool, code, format string, args ...any) string {
	s := fmt.Sprintf(format, args...)
	if !color {
		return s
	}
	return code + s + colorReset
}
