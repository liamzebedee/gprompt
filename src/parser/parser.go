package parser

import (
	"fmt"
	"os"
	"strings"
)

type NodeType int

const (
	NodeMethodDef NodeType = iota
	NodeInvocation
	NodeImport
	NodePlainText
)

type Node struct {
	Type       NodeType
	Name       string   // method name (def/invocation)
	Params     []string // param names (def)
	Body       string   // body text (def)
	Args       []string // arg values (invocation)
	Trailing   string   // trailing text (invocation)
	ImportPath string   // file path (import)
	Text       string   // content (plain text)
}

func Parse(filename string) ([]Node, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseString(string(content))
}

func ParseString(content string) ([]Node, error) {
	lines := strings.Split(content, "\n")

	var nodes []Node
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}

		// Method definition: unindented line ending with ':'
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "@") && strings.HasSuffix(trimmed, ":") {
			name, params := parseMethodHeader(trimmed)
			var bodyLines []string
			i++
			for i < len(lines) {
				if lines[i] != "" && lines[i] != trimmed && lines[i][0] == ' ' {
					return nil, fmt.Errorf("line %d: use tabs for indentation, not spaces", i+1)
				}
				if strings.HasPrefix(lines[i], "\t") {
					bodyLines = append(bodyLines, strings.TrimPrefix(lines[i], "\t"))
					i++
				} else if strings.TrimSpace(lines[i]) == "" {
					// Blank line in body: include if next non-blank line is indented
					j := i + 1
					for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
						j++
					}
					if j < len(lines) && strings.HasPrefix(lines[j], "\t") {
						bodyLines = append(bodyLines, "")
						i++
					} else {
						break
					}
				} else {
					break
				}
			}
			nodes = append(nodes, Node{
				Type:   NodeMethodDef,
				Name:   name,
				Params: params,
				Body:   strings.Join(bodyLines, "\n"),
			})
			continue
		}

		// Parse line for @ expressions (can appear anywhere in the line)
		nodes = append(nodes, parseLine(trimmed)...)
		i++
	}

	return nodes, nil
}

// parseLine splits a line into plain text, imports, and invocation nodes.
// Handles inline @ expressions like "some text @method(args) more text".
func parseLine(line string) []Node {
	var nodes []Node
	for {
		atIdx := strings.Index(line, "@")
		if atIdx == -1 {
			// No more @ — rest is plain text
			if t := strings.TrimSpace(line); t != "" {
				nodes = append(nodes, Node{Type: NodePlainText, Text: t})
			}
			break
		}

		// Text before the @
		if atIdx > 0 {
			if t := strings.TrimSpace(line[:atIdx]); t != "" {
				nodes = append(nodes, Node{Type: NodePlainText, Text: t})
			}
		}

		rest := line[atIdx+1:]
		if len(rest) == 0 {
			break
		}

		// Import: word ends with .p and no parens
		fields := strings.Fields(rest)
		firstWord := fields[0]
		if strings.HasSuffix(firstWord, ".p") && !strings.Contains(firstWord, "(") {
			nodes = append(nodes, Node{Type: NodeImport, ImportPath: firstWord})
			line = strings.TrimSpace(rest[len(firstWord):])
			continue
		}

		// Invocation with parens: consume name(...)
		if parenIdx := strings.Index(rest, "("); parenIdx != -1 {
			// Check there's no space before the paren (it's part of the method name)
			nameCandidate := rest[:parenIdx]
			if !strings.Contains(nameCandidate, " ") {
				closeIdx := strings.Index(rest, ")")
				if closeIdx != -1 {
					name := nameCandidate
					argStr := rest[parenIdx+1 : closeIdx]
					var args []string
					if argStr != "" {
						args = strings.Split(argStr, ",")
						for i := range args {
							args[i] = strings.TrimSpace(args[i])
						}
					}
					nodes = append(nodes, Node{
						Type: NodeInvocation,
						Name: name,
						Args: args,
					})
					line = strings.TrimSpace(rest[closeIdx+1:])
					continue
				}
			}
		}

		// Invocation without parens: @word consumes rest of line as trailing
		parts := strings.SplitN(rest, " ", 2)
		name := parts[0]
		trailing := ""
		if len(parts) > 1 {
			trailing = parts[1]
		}
		nodes = append(nodes, Node{
			Type:     NodeInvocation,
			Name:     name,
			Trailing: trailing,
		})
		break // trailing consumed rest of line
	}
	return nodes
}

func parseMethodHeader(line string) (string, []string) {
	line = strings.TrimSuffix(line, ":")

	if idx := strings.Index(line, "("); idx != -1 {
		name := line[:idx]
		closeIdx := strings.Index(line, ")")
		if closeIdx == -1 {
			return line, nil
		}
		paramStr := line[idx+1 : closeIdx]
		if paramStr == "" {
			return name, nil
		}
		params := strings.Split(paramStr, ",")
		for i := range params {
			params[i] = strings.TrimSpace(params[i])
		}
		return name, params
	}

	return line, nil
}

func parseInvocation(rest string) (string, []string, string) {
	// With parenthesized args: method(arg1, arg2) trailing
	if idx := strings.Index(rest, "("); idx != -1 {
		name := rest[:idx]
		closeIdx := strings.Index(rest, ")")
		if closeIdx == -1 {
			return rest, nil, ""
		}
		argStr := rest[idx+1 : closeIdx]
		var args []string
		if argStr != "" {
			args = strings.Split(argStr, ",")
			for i := range args {
				args[i] = strings.TrimSpace(args[i])
			}
		}
		trailing := strings.TrimSpace(rest[closeIdx+1:])
		return name, args, trailing
	}

	// Without parens: method trailing text
	parts := strings.SplitN(rest, " ", 2)
	name := parts[0]
	trailing := ""
	if len(parts) > 1 {
		trailing = parts[1]
	}
	return name, nil, trailing
}

