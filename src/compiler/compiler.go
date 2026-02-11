package compiler

import (
	"fmt"
	"strings"

	"p2p/src/parser"
	"p2p/src/registry"
)

// CompiledPrompt represents a resolved prompt ready for execution
type CompiledPrompt struct {
	Prompt  string
	LineNum int
}

// ExecutionPlan is an ordered list of prompts to execute
type ExecutionPlan struct {
	Prompts []*CompiledPrompt
}

// Compiler compiles programs to execution plans
type Compiler struct {
	registry *registry.Registry
}

// NewCompiler creates a new compiler with a registry
func NewCompiler(reg *registry.Registry) *Compiler {
	return &Compiler{
		registry: reg,
	}
}

// Compile converts a program to an execution plan
func (c *Compiler) Compile(program *parser.Program) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		Prompts: make([]*CompiledPrompt, 0),
	}

	for _, inv := range program.Invocations {
		compiled, err := c.resolveInvocation(inv)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", inv.LineNum, err)
		}
		plan.Prompts = append(plan.Prompts, compiled)
	}

	return plan, nil
}

// resolveInvocation resolves a method invocation
func (c *Compiler) resolveInvocation(inv *parser.MethodInvocation) (*CompiledPrompt, error) {
	method, err := c.registry.Get(inv.Name)
	if err != nil {
		return nil, err
	}

	// Check parameter count
	if len(inv.Arguments) != len(method.Params) {
		return nil, fmt.Errorf("method '%s' expects %d arguments, got %d",
			inv.Name, len(method.Params), len(inv.Arguments))
	}

	// Build parameter map
	params := make(map[string]string)
	for i, param := range method.Params {
		params[param] = inv.Arguments[i]
	}

	// Interpolate body
	prompt := c.interpolate(method.Body, params)

	// Append trailing text
	if inv.TrailingText != "" {
		prompt += "\n" + inv.TrailingText
	}

	return &CompiledPrompt{
		Prompt:  prompt,
		LineNum: inv.LineNum,
	}, nil
}

// interpolate replaces [param] placeholders with argument values
func (c *Compiler) interpolate(body string, params map[string]string) string {
	result := body
	for param, value := range params {
		placeholder := "[" + param + "]"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
