package parser

import (
	"bufio"
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
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

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
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "@") && strings.HasSuffix(trimmed, ":") {
			name, params := parseMethodHeader(trimmed)
			var bodyLines []string
			i++
			for i < len(lines) {
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

		// Lines starting with @
		if strings.HasPrefix(trimmed, "@") {
			rest := trimmed[1:]
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				i++
				continue
			}

			// Import: first word ends with .p and no parens
			firstWord := fields[0]
			if strings.HasSuffix(firstWord, ".p") && !strings.Contains(firstWord, "(") {
				nodes = append(nodes, Node{
					Type:       NodeImport,
					ImportPath: firstWord,
				})
				i++
				continue
			}

			// Invocation
			name, args, trailing := parseInvocation(rest)
			nodes = append(nodes, Node{
				Type:     NodeInvocation,
				Name:     name,
				Args:     args,
				Trailing: trailing,
			})
			i++
			continue
		}

		// Plain text
		nodes = append(nodes, Node{
			Type: NodePlainText,
			Text: trimmed,
		})
		i++
	}

	return nodes, nil
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
