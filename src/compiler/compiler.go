package compiler

import (
	"strings"

	"p2p/parser"
	"p2p/registry"
)

// Compile takes execution nodes and a registry, expands all invocations,
// and concatenates everything into a single prompt string.
func Compile(nodes []parser.Node, reg *registry.Registry) string {
	var out strings.Builder

	for _, node := range nodes {
		switch node.Type {
		case parser.NodeInvocation:
			if out.Len() > 0 {
				out.WriteString("\n")
			}
			out.WriteString(expand(node, reg))

		case parser.NodePlainText:
			if out.Len() > 0 {
				out.WriteString("\n")
			}
			out.WriteString(node.Text)
		}
	}

	return out.String()
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

	// Interpolate [param] with arg values (supports name=value and positional)
	positional := 0
	for _, arg := range node.Args {
		if k, v, ok := strings.Cut(arg, "="); ok {
			body = strings.ReplaceAll(body, "["+k+"]", v)
		} else if positional < len(method.Params) {
			body = strings.ReplaceAll(body, "["+method.Params[positional]+"]", arg)
			positional++
		}
	}

	// Append trailing text
	if node.Trailing != "" {
		body += "\n" + node.Trailing
	}

	return body
}
