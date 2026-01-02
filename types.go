package main

// types.go - Data structures for the toolbox registry
// This file defines the shape of our YAML data and how we work with it in Go

// Example represents a single usage example for a tool
// In Python, this would be a dataclass or a dict with specific keys
type Example struct {
	Cmd  string `yaml:"cmd"`  // The command to run
	Desc string `yaml:"desc"` // What the command does
}

// Tool represents a single tool entry in the registry
// The yaml:"fieldname" tags tell Go how to map YAML keys to struct fields
// This is similar to Python's dataclass with field annotations, but more explicit
type Tool struct {
	Category     string   `yaml:"category"`
	Description  string   `yaml:"description"`
	InstalledVia string   `yaml:"installed_via"`
	Usage        string   `yaml:"usage"`
	WhyUse       string   `yaml:"why_use"`      // YAML snake_case maps to Go PascalCase
	Examples     []Example `yaml:"examples"`    // Slice (like Python list) of Example structs
	SeeAlso      []string `yaml:"see_also"`     // Slice of strings
	Tags         []string `yaml:"tags"`
	DocsURL      string   `yaml:"docs_url"`
}

// Registry is a map from tool name (string) to Tool struct
// In Python: dict[str, Tool]
// The key difference: Go maps are NOT ordered (like Python's dict before 3.7)
type Registry struct {
	Tools map[string]Tool `yaml:"tools"`
}

// ByCategory groups tools by their category
// Returns a map where keys are category names, values are slices of tool names
// This is a method on Registry (receiver = r *Registry)
// In Python: def by_category(self) -> dict[str, list[str]]
func (r *Registry) ByCategory() map[string][]string {
	// Make creates a new map with string keys and []string values
	// In Python: result = {}
	result := make(map[string][]string)

	// Range is Go's foreach loop
	// In Python: for name, tool in self.tools.items()
	for name, tool := range r.Tools {
		// Append to the slice for this category
		// Go automatically handles slice growth (like Python lists)
		result[tool.Category] = append(result[tool.Category], name)
	}

	return result
}

// AllCategories returns all unique category names, sorted
// In Python: def all_categories(self) -> list[str]
func (r *Registry) AllCategories() []string {
	categories := make([]string, 0) // Create empty slice with 0 length
	seen := make(map[string]bool)   // Set-like map for deduplication

	for _, tool := range r.Tools {
		// Check if we've seen this category before
		// In Python: if category not in seen
		if !seen[tool.Category] {
			categories = append(categories, tool.Category)
			seen[tool.Category] = true
		}
	}

	return categories
}
