package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type MethodDefinition struct {
	Name       string
	Parameters []string
	Body       string
	LineNum    int
}

type MethodInvocation struct {
	Name         string
	Arguments    []string
	TrailingText string
	LineNum      int
}

type Program struct {
	Methods     map[string]*MethodDefinition
	Invocations []*MethodInvocation
}

// Parse reads a .p file and returns the parsed program
func Parse(filename string) (*Program, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	program := &Program{
		Methods:     make(map[string]*MethodDefinition),
		Invocations: make([]*MethodInvocation, 0),
	}

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			i++
			continue
		}

		// Check if it's a method definition (ends with ':')
		if isMethodDefinition(line) {
			methodDef, nextLine, err := parseMethodDefinition(lines, i)
			if err != nil {
				return nil, err
			}
			program.Methods[methodDef.Name] = methodDef
			i = nextLine
			continue
		}

		// Check if it's a method invocation (starts with '@')
		if strings.HasPrefix(strings.TrimSpace(line), "@") {
			invocation, err := parseMethodInvocation(line, i+1)
			if err != nil {
				return nil, err
			}
			program.Invocations = append(program.Invocations, invocation)
			i++
			continue
		}

		i++
	}

	return program, nil
}

// isMethodDefinition checks if a line is a method definition
func isMethodDefinition(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "@")
}

// parseMethodDefinition parses a method definition and its body
func parseMethodDefinition(lines []string, startIdx int) (*MethodDefinition, int, error) {
	line := lines[startIdx]
	trimmed := strings.TrimSpace(line)

	// Extract method name and parameters
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx == -1 {
		return nil, 0, fmt.Errorf("malformed method definition at line %d", startIdx+1)
	}

	nameAndParams := strings.TrimSpace(trimmed[:colonIdx])
	name, params := extractMethodNameAndParams(nameAndParams)

	// Collect body (tab-indented lines following the definition)
	var bodyLines []string
	nextIdx := startIdx + 1

	for nextIdx < len(lines) {
		currentLine := lines[nextIdx]

		// Stop if line is empty or comment
		if strings.TrimSpace(currentLine) == "" || strings.HasPrefix(strings.TrimSpace(currentLine), "#") {
			nextIdx++
			continue
		}

		// Stop if line doesn't start with tab
		if !strings.HasPrefix(currentLine, "\t") {
			break
		}

		// Add to body (remove leading tab)
		bodyLines = append(bodyLines, strings.TrimPrefix(currentLine, "\t"))
		nextIdx++
	}

	body := strings.Join(bodyLines, "\n")

	return &MethodDefinition{
		Name:       name,
		Parameters: params,
		Body:       body,
		LineNum:    startIdx + 1,
	}, nextIdx, nil
}

// extractMethodNameAndParams extracts name and parameters from "methodName(param1, param2)"
func extractMethodNameAndParams(nameAndParams string) (string, []string) {
	parenIdx := strings.Index(nameAndParams, "(")
	if parenIdx == -1 {
		// No parameters
		return nameAndParams, []string{}
	}

	name := strings.TrimSpace(nameAndParams[:parenIdx])
	paramsStr := nameAndParams[parenIdx+1:]
	closeParen := strings.LastIndex(paramsStr, ")")
	if closeParen != -1 {
		paramsStr = paramsStr[:closeParen]
	}

	var params []string
	for _, p := range strings.Split(paramsStr, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			params = append(params, trimmed)
		}
	}

	return name, params
}

// parseMethodInvocation parses a method invocation like "@method-name(arg1, arg2) trailing text"
func parseMethodInvocation(line string, lineNum int) (*MethodInvocation, error) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "@") {
		return nil, fmt.Errorf("invalid invocation at line %d: must start with @", lineNum)
	}

	// Remove @ prefix
	trimmed = trimmed[1:]

	// Find method name and arguments
	var methodName string
	var arguments []string
	var trailingText string

	// Find the opening parenthesis
	parenIdx := strings.Index(trimmed, "(")
	if parenIdx == -1 {
		// No arguments, just method name and trailing text
		parts := strings.SplitN(trimmed, " ", 2)
		methodName = parts[0]
		if len(parts) > 1 {
			trailingText = parts[1]
		}
	} else {
		methodName = strings.TrimSpace(trimmed[:parenIdx])

		// Find the closing parenthesis
		closeParenIdx := strings.Index(trimmed, ")")
		if closeParenIdx == -1 {
			return nil, fmt.Errorf("unmatched parenthesis at line %d", lineNum)
		}

		argsStr := trimmed[parenIdx+1 : closeParenIdx]
		for _, arg := range strings.Split(argsStr, ",") {
			trimmedArg := strings.TrimSpace(arg)
			if trimmedArg != "" {
				arguments = append(arguments, trimmedArg)
			}
		}

		// Trailing text is everything after the closing parenthesis
		if closeParenIdx+1 < len(trimmed) {
			trailingText = strings.TrimSpace(trimmed[closeParenIdx+1:])
		}
	}

	return &MethodInvocation{
		Name:         methodName,
		Arguments:    arguments,
		TrailingText: trailingText,
		LineNum:      lineNum,
	}, nil
}
