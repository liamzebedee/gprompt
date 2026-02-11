package main

import (
	"fmt"
	"os"

	"p2p/compiler"
	"p2p/parser"
	"p2p/registry"
	"p2p/runtime"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gprompt <file.p>\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	// 1. Parse input file
	program, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// 2. Load stdlib into registry
	reg := registry.NewRegistry()
	if err := reg.LoadStdlib(); err != nil {
		fmt.Fprintf(os.Stderr, "Registry error: %v\n", err)
		os.Exit(1)
	}

	// 2b. Register custom methods from input file
	for _, method := range program.Methods {
		reg.Register(method)
	}

	// 3. Compile to execution plan
	comp := compiler.NewCompiler(reg)
	plan, err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compile error: %v\n", err)
		os.Exit(1)
	}

	// 4. Execute
	rt := runtime.NewRuntime()
	if err := rt.Execute(plan); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
