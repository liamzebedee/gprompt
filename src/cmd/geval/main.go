package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"p2p/debug"
	"p2p/parser"
	"p2p/registry"
	"p2p/sexp"
)

//go:embed stdlib.p
var stdlibSource string

func main() {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-d" {
			debug.Enabled = true
			args = append(args[:i], args[i+1:]...)
			i--
		}
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: geval [-d] <file.p> [identifier]\n")
		os.Exit(1)
	}

	filename := args[0]
	filter := ""
	if len(args) >= 2 {
		filter = args[1]
	}

	debug.Log("parsing %s", filename)

	nodes, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	debug.Log("parsed %d nodes", len(nodes))

	reg := registry.New()
	loadStdlib(reg, filename)

	// Process nodes: register methods, resolve imports, collect all nodes for emission
	fileDir := filepath.Dir(filename)
	var allNodes []parser.Node

	for _, node := range nodes {
		switch node.Type {
		case parser.NodeMethodDef:
			reg.Register(node.Name, node.Params, node.Body)
			allNodes = append(allNodes, node)
		case parser.NodeImport:
			importPath := resolveImport(node.ImportPath, fileDir)
			importNodes, err := parser.Parse(importPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "import error (%s): %v\n", node.ImportPath, err)
				os.Exit(1)
			}
			allNodes = append(allNodes, node)
			for _, n := range importNodes {
				if n.Type == parser.NodeMethodDef {
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
		default:
			allNodes = append(allNodes, node)
		}
	}

	output := sexp.EmitProgram(allNodes, reg, filter)
	fmt.Print(output)
}

func loadStdlib(reg *registry.Registry, inputFile string) {
	inputDir := filepath.Dir(inputFile)
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	paths := []string{
		filepath.Join(inputDir, "stdlib.p"),
		"stdlib.p",
		filepath.Join(exeDir, "stdlib.p"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			nodes, err := parser.Parse(p)
			if err != nil {
				continue
			}
			for _, n := range nodes {
				if n.Type == parser.NodeMethodDef {
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
			return
		}
	}

	// Fallback: embedded stdlib
	nodes, err := parser.ParseString(stdlibSource)
	if err != nil {
		return
	}
	for _, n := range nodes {
		if n.Type == parser.NodeMethodDef {
			reg.Register(n.Name, n.Params, n.Body)
		}
	}
}

func resolveImport(importPath string, baseDir string) string {
	if filepath.IsAbs(importPath) {
		return importPath
	}
	return filepath.Join(baseDir, importPath)
}
