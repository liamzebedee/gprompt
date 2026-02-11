package compiler

import (
	"strings"

	"p2p/parser"
	"p2p/pipeline"
	"p2p/registry"
)

type PlanKind int

const (
	PlanPrompt   PlanKind = iota
	PlanPipeline
)

type Plan struct {
	Kind     PlanKind
	Prompt   string
	Pipeline *pipeline.Pipeline
	Args     map[string]string
	Preamble string
}

func Compile(nodes []parser.Node, reg *registry.Registry) *Plan {
	// Split nodes: find pipeline invocation, everything else is preamble
	var pipeNode *parser.Node
	var inlinePipe *pipeline.Pipeline
	var rest []parser.Node

	for i := range nodes {
		if nodes[i].Type == parser.NodeInvocation {
			// @loop(method) or @map(ref, method) as inline pipeline syntax
			if pipeNode == nil && (nodes[i].Name == "loop" || nodes[i].Name == "map") {
				syn := nodes[i].Name + "(" + strings.Join(nodes[i].Args, ", ") + ")"
				if p, err := pipeline.Parse(syn); err == nil {
					inlinePipe = p
					continue
				}
			}
			m := reg.Get(nodes[i].Name)
			if m != nil && m.IsPipeline && pipeNode == nil {
				pipeNode = &nodes[i]
				continue
			}
		}
		rest = append(rest, nodes[i])
	}

	compiled := expandNodes(rest, reg)

	if inlinePipe != nil {
		return &Plan{
			Kind:     PlanPipeline,
			Pipeline: inlinePipe,
			Args:     make(map[string]string),
			Preamble: compiled,
		}
	}

	if pipeNode != nil {
		m := reg.Get(pipeNode.Name)
		return &Plan{
			Kind:     PlanPipeline,
			Pipeline: m.Pipeline,
			Args:     bindArgs(m, pipeNode.Args),
			Preamble: compiled,
		}
	}

	return &Plan{Kind: PlanPrompt, Prompt: compiled}
}

func expandNodes(nodes []parser.Node, reg *registry.Registry) string {
	var parts []string
	for _, node := range nodes {
		switch node.Type {
		case parser.NodeInvocation:
			parts = append(parts, expand(node, reg))
		case parser.NodePlainText:
			parts = append(parts, node.Text)
		}
	}
	return strings.Join(parts, "\n")
}

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
	positional := 0
	for _, arg := range node.Args {
		if k, v, ok := strings.Cut(arg, "="); ok {
			body = strings.ReplaceAll(body, "["+k+"]", v)
		} else if positional < len(method.Params) {
			body = strings.ReplaceAll(body, "["+method.Params[positional]+"]", arg)
			positional++
		}
	}

	if node.Trailing != "" {
		body += "\n" + node.Trailing
	}

	return body
}
