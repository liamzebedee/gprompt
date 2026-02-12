package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"p2p/compiler"
	"p2p/debug"
	"p2p/parser"
	"p2p/registry"
	"p2p/runtime"
	"p2p/stdlib"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer debug.Cleanup()

	args := os.Args[1:]
	var evalExpr string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-d":
			debug.Enabled = true
			args = append(args[:i], args[i+1:]...)
			i--
		case "-e":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "-e requires an expression\n")
				os.Exit(1)
			}
			evalExpr = args[i+1]
			args = append(args[:i], args[i+2:]...)
			i--
		}
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: gprompt [-d] [-e expr] <file.p>\n")
		os.Exit(1)
	}

	filename := args[0]

	debug.Log("parsing %s", filename)

	// Parse the input file
	nodes, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	debug.Log("parsed %d nodes", len(nodes))
	for i, n := range nodes {
		debug.Log("  node[%d] type=%d name=%q", i, n.Type, n.Name)
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
			debug.Log("register method %q params=%v", node.Name, node.Params)
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

	// If -e was given, parse it as the exec nodes instead
	if evalExpr != "" {
		debug.Log("eval: %s", evalExpr)
		evalNodes, err := parser.ParseString(evalExpr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "eval parse error: %v\n", err)
			os.Exit(1)
		}
		execNodes = nil
		for _, n := range evalNodes {
			if n.Type != parser.NodeMethodDef {
				execNodes = append(execNodes, n)
			}
		}
	}

	// Compile into a plan
	debug.Log("compiling %d exec nodes", len(execNodes))
	plan := compiler.Compile(execNodes, reg)

	switch plan.Kind {
	case compiler.PlanPrompt:
		if plan.Prompt == "" {
			return
		}
		debug.LogPrompt("COMPILED", 1, plan.Prompt)
		if err := runtime.Execute(ctx, plan.Prompt); err != nil {
			fmt.Fprintf(os.Stderr, "\nruntime error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()

	case compiler.PlanPipeline:
		debug.Log("executing pipeline with %d steps, args=%v", len(plan.Pipeline.Steps), plan.Args)
		if err := runtime.ExecutePipeline(ctx, plan.Pipeline, plan.Args, reg, plan.Preamble); err != nil {
			fmt.Fprintf(os.Stderr, "\npipeline error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}
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
		debug.Log("stdlib search: %s", p)
		if _, err := os.Stat(p); err == nil {
			debug.Log("stdlib found: %s", p)
			nodes, err := parser.Parse(p)
			if err != nil {
				debug.Log("stdlib parse error: %v", err)
				continue
			}
			for _, n := range nodes {
				if n.Type == parser.NodeMethodDef {
					debug.Log("stdlib method: %q", n.Name)
					reg.Register(n.Name, n.Params, n.Body)
				}
			}
			return
		}
	}

	// Fallback: embedded stdlib
	debug.Log("stdlib not found on disk, using embedded")
	nodes, err := parser.ParseString(stdlib.Source)
	if err != nil {
		debug.Log("embedded stdlib parse error: %v", err)
		return
	}
	for _, n := range nodes {
		if n.Type == parser.NodeMethodDef {
			debug.Log("stdlib method: %q", n.Name)
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
