package compiler

import (
	"strings"

	"p2p/parser"
	"p2p/pipeline"
	"p2p/registry"
)

type PlanKind int

const (
	PlanPrompt   PlanKind = iota // single prompt (existing behavior)
	PlanPipeline                 // multi-step pipeline
)

type Plan struct {
	Kind     PlanKind
	Prompt   string             // for PlanPrompt
	Pipeline *pipeline.Pipeline // for PlanPipeline
	Args     map[string]string  // bound args for pipeline
}

// Compile takes execution nodes and a registry, returns a Plan.
// If the nodes contain a single invocation of a pipeline method, returns PlanPipeline.
// Otherwise returns PlanPrompt with the compiled prompt string.
func Compile(nodes []parser.Node, reg *registry.Registry) *Plan {
	// Check for single pipeline invocation
	if len(nodes) == 1 && nodes[0].Type == parser.NodeInvocation {
		method := reg.Get(nodes[0].Name)
		if method != nil && method.IsPipeline {
			args := bindArgs(method, nodes[0].Args)
			return &Plan{
				Kind:     PlanPipeline,
				Pipeline: method.Pipeline,
				Args:     args,
			}
		}
	}

	// Default: compile to single prompt string
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

	return &Plan{
		Kind:   PlanPrompt,
		Prompt: out.String(),
	}
}

// bindArgs maps invocation args to method param names.
func bindArgs(method *registry.Method, args []string) map[string]string {
	bound := make(map[string]string)
	positional := 0
	for _, arg := range args {
		if k, v, ok := strings.Cut(arg, "="); ok {
			bound[k] = v
		} else if positional < len(method.Params) {
			bound[method.Params[positional]] = arg
			positional++
		}
	}
	return bound
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
