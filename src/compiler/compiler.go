package compiler

import (
	"strings"

	"p2p/parser"
	"p2p/registry"
)

// Compile takes execution nodes (invocations + plain text) and a registry,
// and produces a list of prompt strings. Each prompt is one LLM call.
// Each @invocation starts a new step. Plain text accumulates into the current step.
func Compile(nodes []parser.Node, reg *registry.Registry) []string {
	var steps []string
	var current strings.Builder
	hasInvocation := false

	for _, node := range nodes {
		switch node.Type {
		case parser.NodeInvocation:
			if hasInvocation {
				steps = append(steps, current.String())
				current.Reset()
			}
			expanded := expand(node, reg)
			current.WriteString(expanded)
			hasInvocation = true

		case parser.NodePlainText:
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(node.Text)
		}
	}

	if current.Len() > 0 {
		steps = append(steps, current.String())
	}

	return steps
}

func expand(node parser.Node, reg *registry.Registry) string {
	method := reg.Get(node.Name)
	if method == nil {
		// Unknown method — pass through raw
		text := "@" + node.Name
		if len(node.Args) > 0 {
			text += "(" + strings.Join(node.Args, ", ") + ")"
		}
		if node.Trailing != "" {
			text += " " + node.Trailing
		}
		return text
	}

	body := method.Body

	// Interpolate [param] with arg values
	for i, param := range method.Params {
		if i < len(node.Args) {
			body = strings.ReplaceAll(body, "["+param+"]", node.Args[i])
		}
	}

	// Append trailing text
	if node.Trailing != "" {
		body += "\n" + node.Trailing
	}

	return body
}
