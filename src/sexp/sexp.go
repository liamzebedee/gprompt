package sexp

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"p2p/parser"
	"p2p/pipeline"
	"p2p/registry"
)

// EmitProgram emits a (program ...) S-expression from parsed nodes.
// If filter is non-empty, only the matching top-level definition is emitted.
func EmitProgram(nodes []parser.Node, reg *registry.Registry, filter string) string {
	var forms []string
	for _, node := range nodes {
		if filter != "" {
			if node.Type != parser.NodeMethodDef || node.Name != filter {
				continue
			}
		}
		sexpr := emitNode(node, reg)
		if sexpr == "" {
			continue
		}
		hash := shortcode(sexpr)
		forms = append(forms, fmt.Sprintf("; id=%s\n%s", hash, sexpr))
	}

	if filter != "" {
		if len(forms) == 0 {
			return ""
		}
		return forms[0] + "\n"
	}

	if len(forms) == 0 {
		return "(program)\n"
	}

	var lines []string
	lines = append(lines, "(program")
	for i, form := range forms {
		if i > 0 {
			lines = append(lines, "")
		}
		for _, fl := range strings.Split(form, "\n") {
			lines = append(lines, "  "+fl)
		}
	}
	// Close paren on last line
	last := len(lines) - 1
	lines[last] = lines[last] + ")"
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func emitNode(node parser.Node, reg *registry.Registry) string {
	switch node.Type {
	case parser.NodeMethodDef:
		return emitMethodDef(node, reg)
	case parser.NodeInvocation:
		return emitInvocation(node)
	case parser.NodeImport:
		return fmt.Sprintf("(import %q)", node.ImportPath)
	case parser.NodePlainText:
		return fmt.Sprintf("(text %q)", node.Text)
	}
	return ""
}

func emitMethodDef(node parser.Node, reg *registry.Registry) string {
	name := node.Name
	m := reg.Get(name)

	// Agent definition: agent- prefix
	if strings.HasPrefix(name, "agent-") {
		agentName := strings.TrimPrefix(name, "agent-")
		if m != nil && m.IsPipeline {
			return fmt.Sprintf("(defagent %q\n%s)", agentName, indent(emitPipeline(m.Pipeline), 2))
		}
		return fmt.Sprintf("(defagent %q\n%s)", agentName, indent(fmt.Sprintf("%q", node.Body), 2))
	}

	// Pipeline definition
	if m != nil && m.IsPipeline {
		params := formatParams(node.Params)
		return fmt.Sprintf("(defpipeline %s %s\n%s)", name, params, indent(emitPipeline(m.Pipeline), 2))
	}

	// Regular method
	params := formatParams(node.Params)
	return fmt.Sprintf("(defmethod %s %s\n%s)", name, params, indent(fmt.Sprintf("%q", node.Body), 2))
}

func emitInvocation(node parser.Node) string {
	parts := []string{"invoke", node.Name}

	if node.Trailing != "" {
		parts = append(parts, ":trailing", fmt.Sprintf("%q", node.Trailing))
	} else if len(node.Args) > 0 {
		for _, arg := range node.Args {
			if eqIdx := strings.Index(arg, "="); eqIdx != -1 {
				key := arg[:eqIdx]
				val := arg[eqIdx+1:]
				parts = append(parts, ":"+key, fmt.Sprintf("%q", val))
			} else {
				parts = append(parts, fmt.Sprintf("%q", arg))
			}
		}
	}

	return "(" + strings.Join(parts, " ") + ")"
}

func emitPipeline(p *pipeline.Pipeline) string {
	var parts []string
	parts = append(parts, "(pipeline")
	if p.InitialInput != "" {
		parts[0] = "(pipeline " + p.InitialInput
	}
	for _, step := range p.Steps {
		parts = append(parts, indent(emitStep(step), 2))
	}
	return strings.Join(parts, "\n") + ")"
}

func emitStep(s pipeline.Step) string {
	var action string
	switch s.Kind {
	case pipeline.StepSimple:
		action = fmt.Sprintf("(call %s)", s.Method)
	case pipeline.StepMap:
		action = fmt.Sprintf("(map %s %s)", s.MapRef, s.MapMethod)
	case pipeline.StepLoop:
		action = fmt.Sprintf("(loop %s)", s.LoopMethod)
	}
	return fmt.Sprintf("(step %q %s)", s.Label, action)
}

func formatParams(params []string) string {
	if len(params) == 0 {
		return "()"
	}
	return "(" + strings.Join(params, " ") + ")"
}

func indent(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func shortcode(sexpr string) string {
	h := sha256.Sum256([]byte(sexpr))
	return fmt.Sprintf("%x", h[:4])
}
