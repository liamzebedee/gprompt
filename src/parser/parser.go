package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MethodDefinition represents a method defined in a .p file
type MethodDefinition struct {
	Name      string
	Params    []string
	Body      string
	LineNum   int
}

// MethodInvocation represents a method call like @method(args)
type MethodInvocation struct {
	Name         string
	Arguments    []string
	TrailingText string
	LineNum      int
}

// Program represents the parsed .p file
type Program struct {
	Methods     map[string]*MethodDefinition
	Invocations []*MethodInvocation
}

// Parse parses a .p file and returns the Program
func Parse(filename string) (*Program, error) {
	return parseWithBasePath(filename, filepath.Dir(filename))
}

// parseWithBasePath parses a file with a specific base path for imports
func parseWithBasePath(filename string, basePath string) (*Program, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	lines := strings.Split(string(content), "\n")
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

		// Check for file import
		if isFileImport(strings.TrimSpace(line)) {
			importPath := strings.TrimSpace(line)
			if strings.HasPrefix(importPath, "@") {
				importPath = importPath[1:]
			}
			// Resolve relative to current file's directory
			resolvedPath := filepath.Join(basePath, importPath)
			imported, err := parseWithBasePath(resolvedPath, filepath.Dir(resolvedPath))
			if err != nil {
				return nil, fmt.Errorf("failed to import %s: %w", importPath, err)
			}
			// Merge imported methods
			for name, method := range imported.Methods {
				program.Methods[name] = method
			}
			i++
			continue
		}

		// Check for method definition (name: or name(params):)
		if isMethodDefinition(strings.TrimSpace(line)) {
			method, nextIdx, err := parseMethodDefinition(lines, i)
			if err != nil {
				return nil, err
			}
			program.Methods[method.Name] = method
			i = nextIdx
			continue
		}

		// Check for method invocation
		if strings.HasPrefix(strings.TrimSpace(line), "@") {
			inv, nextIdx := parseMethodInvocation(lines, i)
			program.Invocations = append(program.Invocations, inv)
			i = nextIdx
			continue
		}

		i++
	}

	return program, nil
}

// isFileImport checks if a line is a file import like @stdlib.p
func isFileImport(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@") {
		return false
	}
	// File imports end with .p and have no parentheses
	if strings.HasSuffix(line, ".p") && !strings.Contains(line, "(") {
		return true
	}
	return false
}

// isMethodDefinition checks if a line is a method definition
func isMethodDefinition(line string) bool {
	line = strings.TrimSpace(line)
	// Method definition: name: or name(params):
	if strings.Contains(line, ":") && !strings.HasPrefix(line, "@") {
		return true
	}
	return false
}

// parseMethodDefinition parses a method definition block
func parseMethodDefinition(lines []string, startIdx int) (*MethodDefinition, int, error) {
	line := strings.TrimSpace(lines[startIdx])

	// Extract name and parameters
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return nil, startIdx + 1, fmt.Errorf("malformed method definition at line %d", startIdx+1)
	}

	signature := line[:colonIdx]
	name := signature
	var params []string

	if strings.Contains(signature, "(") {
		nameEnd := strings.Index(signature, "(")
		name = signature[:nameEnd]
		paramsStr := signature[nameEnd+1:]
		if strings.HasSuffix(paramsStr, ")") {
			paramsStr = paramsStr[:len(paramsStr)-1]
		}
		if strings.TrimSpace(paramsStr) != "" {
			params = strings.Split(paramsStr, ",")
			for i := range params {
				params[i] = strings.TrimSpace(params[i])
			}
		}
	}

	name = strings.TrimSpace(name)

	// Collect body (tab-indented lines)
	var body []string
	i := startIdx + 1
	for i < len(lines) {
		if lines[i] == "" {
			i++
			continue
		}
		if strings.HasPrefix(lines[i], "\t") {
			// Remove the leading tab
			body = append(body, lines[i][1:])
			i++
		} else if !strings.HasPrefix(lines[i], "\t") && strings.TrimSpace(lines[i]) != "" {
			// End of body
			break
		} else {
			i++
		}
	}

	method := &MethodDefinition{
		Name:    name,
		Params:  params,
		Body:    strings.Join(body, "\n"),
		LineNum: startIdx + 1,
	}

	return method, i, nil
}

// parseMethodInvocation parses a method invocation
func parseMethodInvocation(lines []string, startIdx int) (*MethodInvocation, int) {
	line := lines[startIdx]
	trimmed := strings.TrimSpace(line)

	// Remove leading @
	if strings.HasPrefix(trimmed, "@") {
		trimmed = trimmed[1:]
	}

	// Extract method name and arguments
	var name string
	var args []string
	var trailing string

	// Check for parentheses (method with arguments)
	if parenIdx := strings.Index(trimmed, "("); parenIdx != -1 {
		name = strings.TrimSpace(trimmed[:parenIdx])
		closeIdx := strings.Index(trimmed, ")")
		if closeIdx != -1 {
			argsStr := trimmed[parenIdx+1 : closeIdx]
			if strings.TrimSpace(argsStr) != "" {
				args = strings.Split(argsStr, ",")
				for i := range args {
					args[i] = strings.TrimSpace(args[i])
				}
			}
			trailing = strings.TrimSpace(trimmed[closeIdx+1:])
		}
	} else {
		// No parentheses - split on first space or take whole thing
		parts := strings.SplitN(trimmed, " ", 2)
		name = parts[0]
		if len(parts) > 1 {
			trailing = parts[1]
		}
	}

	inv := &MethodInvocation{
		Name:         name,
		Arguments:    args,
		TrailingText: trailing,
		LineNum:      startIdx + 1,
	}

	return inv, startIdx + 1
}
