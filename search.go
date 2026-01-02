package main

import (
	"sort"
	"strings"
)

// search.go - Search and filter functions for tools
// All functions here are pure (no side effects) - easy to test!

// SearchResult represents a tool that matched a search query
type SearchResult struct {
	Name        string
	Tool        Tool
}

// SearchTools finds all tools matching a query (case-insensitive)
// Searches in: description, tags, why_use, and tool name itself
// Returns slice of results sorted alphabetically by name
func SearchTools(registry *Registry, query string) []SearchResult {
	// Convert query to lowercase for case-insensitive search
	// In Python: query = query.lower()
	query = strings.ToLower(query)

	// results is a slice (dynamic array)
	// In Python: results = []
	results := make([]SearchResult, 0)

	for name, tool := range registry.Tools {
		if matchesTool(name, tool, query) {
			results = append(results, SearchResult{
				Name: name,
				Tool: tool,
			})
		}
	}

	// Sort results alphabetically by name
	// sort.Slice uses a custom comparison function (like Python's sorted with key=)
	// In Python: results.sort(key=lambda r: r.name)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// matchesTool checks if a tool matches the search query
// Helper function - not exported (lowercase first letter = private)
// In Python: def _matches_tool(name, tool, query) -> bool
func matchesTool(name string, tool Tool, query string) bool {
	// Build a searchable string from all relevant fields
	// strings.Builder is efficient for concatenating many strings
	// In Python, you'd use: searchable = " ".join([...])
	var searchable strings.Builder

	searchable.WriteString(strings.ToLower(name))
	searchable.WriteString(" ")
	searchable.WriteString(strings.ToLower(tool.Description))
	searchable.WriteString(" ")
	searchable.WriteString(strings.ToLower(tool.WhyUse))

	// Join all tags into the searchable string
	// In Python: " ".join(tool.tags)
	for _, tag := range tool.Tags {
		searchable.WriteString(" ")
		searchable.WriteString(strings.ToLower(tag))
	}

	// Check if query appears anywhere in the searchable text
	// In Python: query in searchable
	return strings.Contains(searchable.String(), query)
}

// GetToolsByCategory returns all tools in a category, sorted alphabetically
func GetToolsByCategory(registry *Registry, category string) []SearchResult {
	results := make([]SearchResult, 0)

	for name, tool := range registry.Tools {
		if tool.Category == category {
			results = append(results, SearchResult{
				Name: name,
				Tool: tool,
			})
		}
	}

	// Sort alphabetically
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// GetAllToolsSorted returns all tools sorted alphabetically by name
func GetAllToolsSorted(registry *Registry) []SearchResult {
	results := make([]SearchResult, 0, len(registry.Tools))

	for name, tool := range registry.Tools {
		results = append(results, SearchResult{
			Name: name,
			Tool: tool,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// GetCategoriesSorted returns all category names sorted alphabetically
func GetCategoriesSorted(registry *Registry) []string {
	categories := registry.AllCategories()

	// Sort in place
	// In Python: categories.sort()
	sort.Strings(categories)

	return categories
}
