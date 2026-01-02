package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// registry.go - Loading and accessing the tool registry YAML file

// LoadRegistry reads and parses the registry YAML file
// Returns (*Registry, error) - a pointer to Registry and an error
// Go uses explicit error returns instead of exceptions
// In Python, this would raise an exception on error
func LoadRegistry(path string) (*Registry, error) {
	// Read file - returns ([]byte, error)
	// := is short variable declaration: type is inferred
	// In Python: data = open(path, 'rb').read()
	data, err := os.ReadFile(path)
	if err != nil { // Go's error handling: explicit checks, no try/catch
		// Wrap error with context (like Python's "raise ... from e")
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	// Create a new Registry struct
	// & takes the address (pointer) - we return *Registry not Registry
	// Why? Efficiency - don't copy large structs, pass reference
	// In Python: everything is a reference by default
	var registry Registry

	// Unmarshal YAML into our struct
	// In Python: registry = yaml.safe_load(data)
	err = yaml.Unmarshal(data, &registry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry YAML: %w", err)
	}

	// Return pointer and nil error (nil = Python's None)
	return &registry, nil
}

// GetRegistryPath returns the path to the registry file
// Checks DOTFILES_REGISTRY env var first, falls back to default
// In Python: def get_registry_path() -> str
func GetRegistryPath() string {
	// os.Getenv returns empty string if not set (no KeyError like Python)
	if path := os.Getenv("DOTFILES_REGISTRY"); path != "" {
		return path
	}

	// Get home directory
	// In Python: pathlib.Path.home()
	home, err := os.UserHomeDir()
	if err != nil {
		// Panic is like raising an exception, but for unrecoverable errors
		// Use sparingly - prefer returning errors
		panic(fmt.Sprintf("cannot determine home directory: %v", err))
	}

	// Join path components
	// In Python: Path(home) / ".config" / "toolbox" / "registry.yml"
	return filepath.Join(home, ".config", "toolbox", "registry.yml")
}

// ToolExists checks if a tool exists in the registry
// This is a method on *Registry (pointer receiver)
// In Python: def tool_exists(self, name: str) -> bool
func (r *Registry) ToolExists(name string) bool {
	// Map lookup returns (value, exists)
	// The _ discards the value, we only care if key exists
	// In Python: name in self.tools
	_, exists := r.Tools[name]
	return exists
}

// GetTool returns a tool by name
// Returns the Tool and a boolean indicating if it was found
// In Python, you'd do: self.tools.get(name) and check for None
// Go prefers explicit "found" boolean to avoid nil pointer issues
func (r *Registry) GetTool(name string) (Tool, bool) {
	tool, exists := r.Tools[name]
	return tool, exists
}
