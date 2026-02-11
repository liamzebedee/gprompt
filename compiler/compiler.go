package compiler

import (
	"fmt"
	"strings"

	"p2p/parser"
	"p2p/registry"
)

type CompiledPrompt struct {
	Text    string
	LineNum int
}

type ExecutionPlan struct {
	Prompts []*CompiledPrompt
}

type Compiler struct {
	registry *registry.Registry
}

// NewCompiler creates a new compiler with a registry
func NewCompiler(reg *registry.Registry) *Compiler {
	return &Compiler{
		registry: reg,
	}
}

// Compile takes a program and returns an execution plan
func (c *Compiler) Compile(program *parser.Program) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		Prompts: make([]*CompiledPrompt, 0),
	}

	for _, invocation := range program.Invocations {
		prompt, err := c.resolveInvocation(invocation)
		if err != nil {
			return nil, fmt.Errorf("error at line %d: %w", invocation.LineNum, err)
		}

		plan.Prompts = append(plan.Prompts, &CompiledPrompt{
			Text:    prompt,
			LineNum: invocation.LineNum,
		})
	}

	return plan, nil
}

// resolveInvocation resolves a method invocation to a prompt string
func (c *Compiler) resolveInvocation(inv *parser.MethodInvocation) (string, error) {
	// Get method definition from registry
	methodDef, err := c.registry.Get(inv.Name)
	if err != nil {
		return "", err
	}

	// Build parameter map
	paramMap := make(map[string]string)
	if len(methodDef.Parameters) != len(inv.Arguments) {
		return "", fmt.Errorf("method @%s expects %d arguments, got %d",
			inv.Name, len(methodDef.Parameters), len(inv.Arguments))
	}

	for i, paramName := range methodDef.Parameters {
		paramMap[paramName] = inv.Arguments[i]
	}

	// Interpolate body
	interpolated := c.interpolate(methodDef.Body, paramMap)

	// Append trailing text
	var result strings.Builder
	result.WriteString(interpolated)
	if inv.TrailingText != "" {
		if interpolated != "" {
			result.WriteString("\n")
		}
		result.WriteString(inv.TrailingText)
	}

	return result.String(), nil
}

// interpolate replaces [param] placeholders with values
func (c *Compiler) interpolate(body string, params map[string]string) string {
	result := body
	for paramName, paramValue := range params {
		placeholder := "[" + paramName + "]"
		result = strings.ReplaceAll(result, placeholder, paramValue)
	}
	return result
}
