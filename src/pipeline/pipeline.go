package pipeline

import (
	"fmt"
	"strings"
)

type StepKind int

const (
	StepSimple StepKind = iota
	StepMap
)

type Step struct {
	Label     string   // output name ("book-outline")
	Method    string   // method to call ("generate-outline")
	Kind      StepKind // StepSimple or StepMap
	MapRef    string   // for map: descriptive name of items
	MapMethod string   // for map: method to call per item
}

type Pipeline struct {
	InitialInput string // first token before first -> ("topic")
	Steps        []Step
}

// IsPipeline checks if a method body contains pipeline syntax.
func IsPipeline(body string) bool {
	return strings.Contains(body, " -> ")
}

// Parse splits a pipeline body into its initial input and steps.
// Format: "input -> label (method) -> label (map(ref, method))"
func Parse(body string) (*Pipeline, error) {
	// Pipeline body should be a single line
	line := strings.TrimSpace(body)

	segments := strings.Split(line, " -> ")
	if len(segments) < 2 {
		return nil, fmt.Errorf("pipeline must have at least one step after initial input")
	}

	p := &Pipeline{
		InitialInput: strings.TrimSpace(segments[0]),
	}

	for i := 1; i < len(segments); i++ {
		seg := strings.TrimSpace(segments[i])
		step, err := parseStep(seg)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i, err)
		}
		p.Steps = append(p.Steps, step)
	}

	return p, nil
}

// parseStep parses "label (method)" or "label (map(ref, method))"
func parseStep(seg string) (Step, error) {
	parenIdx := strings.Index(seg, " (")
	if parenIdx == -1 {
		return Step{}, fmt.Errorf("step %q missing method in parens", seg)
	}

	label := strings.TrimSpace(seg[:parenIdx])
	rest := seg[parenIdx+2:] // skip " ("

	// Strip trailing )
	if !strings.HasSuffix(rest, ")") {
		return Step{}, fmt.Errorf("step %q missing closing paren", seg)
	}
	rest = rest[:len(rest)-1]

	// Check for map(ref, method)
	if strings.HasPrefix(rest, "map(") {
		inner := rest[4:] // skip "map("
		if !strings.HasSuffix(inner, ")") {
			return Step{}, fmt.Errorf("step %q malformed map expression", seg)
		}
		inner = inner[:len(inner)-1]

		parts := strings.SplitN(inner, ",", 2)
		if len(parts) != 2 {
			return Step{}, fmt.Errorf("step %q map needs (ref, method)", seg)
		}

		return Step{
			Label:     label,
			Kind:      StepMap,
			MapRef:    strings.TrimSpace(parts[0]),
			MapMethod: strings.TrimSpace(parts[1]),
		}, nil
	}

	return Step{
		Label:  label,
		Method: rest,
		Kind:   StepSimple,
	}, nil
}
