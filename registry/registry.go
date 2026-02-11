package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"p2p/parser"
)

type Registry struct {
	methods map[string]*parser.MethodDefinition
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		methods: make(map[string]*parser.MethodDefinition),
	}
}

// LoadStdlib loads the standard library from stdlib.p
func (r *Registry) LoadStdlib() error {
	// Try multiple paths
	paths := []string{
		"./stdlib.p",
		"/home/liam/Music/p2p/stdlib.p",
	}

	// Also try relative to the executable
	if exePath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), "stdlib.p"))
	}

	var lastErr error
	for _, path := range paths {
		if err := r.Load(path); err == nil {
			return nil // Successfully loaded
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("could not load stdlib.p from any path: %w", lastErr)
}

// Load parses a file and registers all methods from it
func (r *Registry) Load(filename string) error {
	program, err := parser.Parse(filename)
	if err != nil {
		return err
	}

	for name, methodDef := range program.Methods {
		r.methods[name] = methodDef
	}

	return nil
}

// Get retrieves a method definition by name
func (r *Registry) Get(name string) (*parser.MethodDefinition, error) {
	method, exists := r.methods[name]
	if !exists {
		return nil, fmt.Errorf("method not found: @%s", name)
	}
	return method, nil
}

// Register adds a method definition to the registry
func (r *Registry) Register(method *parser.MethodDefinition) {
	r.methods[method.Name] = method
}
