package main

import (
	"fmt"
	"os"

	"p2p/src/compiler"
	"p2p/src/parser"
	"p2p/src/registry"
	"p2p/src/runtime"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: gprompt <file.p>\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Step 1: Parse input file
	program, err := parser.Parse(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Load stdlib
	reg := registry.NewRegistry()
	_ = reg.LoadStdlib() // stdlib is optional

	// Step 3: Register custom methods from input file
	for _, method := range program.Methods {
		reg.Register(method)
	}

	// Step 4: Compile program
	comp := compiler.NewCompiler(reg)
	plan, err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Execute plan
	rt := runtime.NewRuntime()
	if err := rt.Execute(plan); err != nil {
		fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
		os.Exit(1)
	}
}
