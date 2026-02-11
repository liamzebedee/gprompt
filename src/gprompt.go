package main

import (
	"fmt"
	"os"
	"path/filepath"

	"p2p/compiler"
	"p2p/parser"
	"p2p/registry"
	"p2p/runtime"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gprompt <file.p>\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Parse the input file
	nodes, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Create registry and auto-load stdlib
	reg := registry.New()
	loadStdlib(reg, filename)

	// Process nodes: register methods, handle imports, collect execution nodes
	fileDir := filepath.Dir(filename)
	var execNodes []parser.Node

	for _, node := range nodes {
		switch node.Type {
		case parser.NodeMethodDef:
			reg.Register(node.Name, node.Params, node.Body)
		case parser.NodeImport:
			importPath := resolveImport(node.ImportPath, fileDir)
			importNodes, err := parser.Parse(importPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "import error (%s): %v\n", node.ImportPath, err)
				os.Exit(1)
			}
			for _, n := range importNodes {
				if n.Type == parser.NodeMethodDef {
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
		default:
			execNodes = append(execNodes, node)
		}
	}

	// Compile invocations into prompt steps
	steps := compiler.Compile(execNodes, reg)
	if len(steps) == 0 {
		return
	}

	// Execute
	if err := runtime.Execute(steps); err != nil {
		fmt.Fprintf(os.Stderr, "\nruntime error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
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
}

func resolveImport(importPath string, baseDir string) string {
	if filepath.IsAbs(importPath) {
		return importPath
	}
	return filepath.Join(baseDir, importPath)
}
