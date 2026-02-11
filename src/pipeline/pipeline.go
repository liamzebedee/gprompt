package pipeline

import (
	"fmt"
	"strings"
)

type StepKind int

const (
	StepSimple StepKind = iota
	StepMap
	StepLoop
)

type Step struct {
	Label      string   // output name ("book-outline")
	Method     string   // method to call ("generate-outline")
	Kind       StepKind // StepSimple, StepMap, or StepLoop
	MapRef     string   // for map: descriptive name of items
	MapMethod  string   // for map: method to call per item
	LoopMethod string   // for loop: method to call each iteration
}

type Pipeline struct {
	InitialInput string // first token before first -> ("topic")
	Steps        []Step
}

// IsPipeline checks if a method body contains pipeline syntax.
func IsPipeline(body string) bool {
	if strings.Contains(body, " -> ") {
		return true
	}
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "loop(") || strings.HasPrefix(trimmed, "map(")
}

// Parse splits a pipeline body into its initial input and steps.
// Format: "input -> label (method) -> label (map(ref, method))"
func Parse(body string) (*Pipeline, error) {
	// Pipeline body should be a single line
	line := strings.TrimSpace(body)

	// Handle bare loop(...) or map(...) with no initial input
	if !strings.Contains(line, " -> ") {
		step, err := parseStep(line)
		if err != nil {
			return nil, fmt.Errorf("step 1: %w", err)
		}
		return &Pipeline{Steps: []Step{step}}, nil
	}

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

// parseStep parses "label (method)", "label (map(ref, method))", "label (loop(method))",
// or bare "method" (label and method are the same).
func parseStep(seg string) (Step, error) {
	parenIdx := strings.Index(seg, " (")
	if parenIdx == -1 {
		name := strings.TrimSpace(seg)

		// Check for loop(method) without a label
		if strings.HasPrefix(name, "loop(") && strings.HasSuffix(name, ")") {
			inner := name[5 : len(name)-1]
			return Step{
				Label:      inner,
				Kind:       StepLoop,
				LoopMethod: strings.TrimSpace(inner),
			}, nil
		}

		// Check for map(ref, method) without a label
		if strings.HasPrefix(name, "map(") && strings.HasSuffix(name, ")") {
			inner := name[4 : len(name)-1]
			parts := strings.SplitN(inner, ",", 2)
			if len(parts) != 2 {
				return Step{}, fmt.Errorf("step %q map needs (ref, method)", seg)
			}
			return Step{
				Label:     strings.TrimSpace(parts[0]),
				Kind:      StepMap,
				MapRef:    strings.TrimSpace(parts[0]),
				MapMethod: strings.TrimSpace(parts[1]),
			}, nil
		}

		// Bare word: label = method
		return Step{
			Label:  name,
			Method: name,
			Kind:   StepSimple,
		}, nil
	}

	label := strings.TrimSpace(seg[:parenIdx])
	rest := seg[parenIdx+2:] // skip " ("

	// Strip trailing )
	if !strings.HasSuffix(rest, ")") {
		return Step{}, fmt.Errorf("step %q missing closing paren", seg)
	}
	rest = rest[:len(rest)-1]

	// Check for loop(method)
	if strings.HasPrefix(rest, "loop(") {
		inner := rest[5:] // skip "loop("
		if !strings.HasSuffix(inner, ")") {
			return Step{}, fmt.Errorf("step %q malformed loop expression", seg)
		}
		inner = inner[:len(inner)-1]

		return Step{
			Label:      label,
			Kind:       StepLoop,
			LoopMethod: strings.TrimSpace(inner),
		}, nil
	}

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
