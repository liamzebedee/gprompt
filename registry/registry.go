package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"p2p/parser"
)

// Registry manages method definitions
type Registry struct {
	methods map[string]*parser.MethodDefinition
}

// NewRegistry creates a new empty registry
func NewRegistry() *Registry {
	return &Registry{
		methods: make(map[string]*parser.MethodDefinition),
	}
}

// LoadStdlib loads the standard library from various possible locations
func (r *Registry) LoadStdlib() error {
	possiblePaths := []string{
		"./stdlib.p",
		"/home/liam/Music/p2p/stdlib.p",
	}

	// Also try to find relative to the executable
	exePath, err := os.Executable()
	if err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(filepath.Dir(exePath), "stdlib.p"))
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return r.Load(path)
		}
	}

	// stdlib.p is optional, so don't error if not found
	return nil
}

// Load parses a file and registers all methods from it
func (r *Registry) Load(filename string) error {
	program, err := parser.Parse(filename)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	for name, method := range program.Methods {
		r.methods[name] = method
	}

	return nil
}

// Get retrieves a method from the registry
func (r *Registry) Get(name string) (*parser.MethodDefinition, error) {
	if method, exists := r.methods[name]; exists {
		return method, nil
	}
	return nil, fmt.Errorf("method '%s' not found", name)
}

// Register adds a method to the registry
func (r *Registry) Register(method *parser.MethodDefinition) {
	r.methods[method.Name] = method
}
