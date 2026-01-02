package main

import (
	"strings"
	"testing"
)

// search_test.go - Example tests for search functionality
// Go has built-in testing - no pytest needed!
// Run with: go test

// TestSearchTools verifies case-insensitive search works correctly
// Test function names must start with "Test"
// In Python with pytest: def test_search_tools():
func TestSearchTools(t *testing.T) {
	// t is the test context - like pytest's implicit fixtures
	// t.Error, t.Fatal, etc. are used instead of assert

	// Create a small test registry
	registry := &Registry{
		Tools: map[string]Tool{
			"git-delta": {
				Category:    "version-control",
				Description: "Syntax-highlighting pager for git",
				Tags:        []string{"git", "diff", "pager"},
			},
			"ripgrep": {
				Category:    "search",
				Description: "Fast text search tool",
				Tags:        []string{"search", "grep"},
			},
			"bat": {
				Category:    "file-viewer",
				Description: "Cat with syntax highlighting",
				Tags:        []string{"syntax", "cat"},
			},
		},
	}

	// Test case-insensitive search
	results := SearchTools(registry, "GIT") // uppercase
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'GIT', got %d", len(results))
	}
	if len(results) > 0 && results[0].Name != "git-delta" {
		t.Errorf("Expected 'git-delta', got '%s'", results[0].Name)
	}

	// Test search by tag
	results = SearchTools(registry, "syntax")
	if len(results) != 2 { // Both git-delta and bat have syntax-related content
		t.Errorf("Expected 2 results for 'syntax', got %d", len(results))
	}

	// Test no results
	results = SearchTools(registry, "nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nonexistent', got %d", len(results))
	}
}

// TestGetToolsByCategory verifies category filtering
func TestGetToolsByCategory(t *testing.T) {
	registry := &Registry{
		Tools: map[string]Tool{
			"git": {Category: "version-control"},
			"lazygit": {Category: "version-control"},
			"ripgrep": {Category: "search"},
		},
	}

	results := GetToolsByCategory(registry, "version-control")
	if len(results) != 2 {
		t.Errorf("Expected 2 tools in version-control, got %d", len(results))
	}

	// Verify alphabetical sorting
	if len(results) == 2 {
		if results[0].Name != "git" || results[1].Name != "lazygit" {
			t.Errorf("Results not sorted alphabetically: %v", results)
		}
	}
}

// Example of table-driven tests - very common in Go
// In Python, you'd use pytest.mark.parametrize
func TestMatchesTool(t *testing.T) {
	tool := Tool{
		Description: "Fast search tool",
		Tags:        []string{"search", "grep"},
		WhyUse:      "Better than grep",
	}

	// Slice of test cases
	// In Python: @pytest.mark.parametrize("name,query,expected", [...])
	tests := []struct{
		name     string // test case name
		query    string
		expected bool   // should it match?
	}{
		{"exact match", "search", true},
		{"case insensitive", "SEARCH", true},
		{"tag match", "grep", true},
		{"partial match", "fast", true},
		{"no match", "database", false},
	}

	// Run each test case
	for _, tt := range tests {
		// t.Run creates a subtest - appears in output as "TestMatchesTool/exact_match"
		t.Run(tt.name, func(t *testing.T) {
			// matchesTool expects query to already be lowercased
			// (SearchTools does this before calling matchesTool)
			query := strings.ToLower(tt.query)
			result := matchesTool("ripgrep", tool, query)
			if result != tt.expected {
				t.Errorf("query '%s': expected %v, got %v", tt.query, tt.expected, result)
			}
		})
	}
}
