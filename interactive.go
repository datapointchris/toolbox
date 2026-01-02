package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// interactive.go - Interactive menu functionality using gum
// This calls out to the `gum` CLI tool for beautiful terminal UIs

// SelectCategory shows an interactive category picker using gum
// Returns selected category name or empty string if cancelled
func SelectCategory(registry *Registry) (string, error) {
	categories := GetCategoriesSorted(registry)
	categoryMap := registry.ByCategory()

	// Build gum choose command
	// In Python: subprocess.run(['gum', 'choose', ...], capture_output=True)
	args := []string{"choose", "--header", "Select a category:"}

	// Add each category with tool count
	for _, cat := range categories {
		count := len(categoryMap[cat])
		args = append(args, fmt.Sprintf("%s (%d tools)", cat, count))
	}

	// Run gum choose
	cmd := exec.Command("gum", args...)
	cmd.Stderr = os.Stderr // Show gum errors

	// Run and capture output
	output, err := cmd.Output()
	if err != nil {
		// User cancelled (Ctrl+C) - not really an error
		return "", nil
	}

	// Parse selection - extract category name before " ("
	selection := strings.TrimSpace(string(output))
	parts := strings.Split(selection, " (")
	if len(parts) > 0 {
		return parts[0], nil
	}

	return "", nil
}

// SelectToolInCategory shows an interactive tool picker for a category
// Uses gum with preview showing tool details
func SelectToolInCategory(registry *Registry, category string) (string, error) {
	tools := GetToolsByCategory(registry, category)

	if len(tools) == 0 {
		fmt.Printf("%s No tools found in category: %s\n", colorRed("Error:"), category)
		return "", nil
	}

	// Build list of tool names for gum
	toolNames := make([]string, len(tools))
	for i, result := range tools {
		toolNames[i] = result.Name
	}

	// Build gum choose command with preview
	// The preview command will show tool details when hovering
	args := []string{
		"choose",
		"--header", fmt.Sprintf("Tools in %s:", category),
		"--height", "20",
	}
	args = append(args, toolNames...)

	cmd := exec.Command("gum", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return "", nil // User cancelled
	}

	return strings.TrimSpace(string(output)), nil
}

// InteractiveCategoryBrowse provides a two-level menu: category -> tool -> details
// This is the main interactive flow for browsing tools
func InteractiveCategoryBrowse(registry *Registry) error {
	// Step 1: Select category
	category, err := SelectCategory(registry)
	if err != nil {
		return err
	}
	if category == "" {
		return nil // User cancelled
	}

	// Step 2: Select tool in category
	toolName, err := SelectToolInCategory(registry, category)
	if err != nil {
		return err
	}
	if toolName == "" {
		return nil // User cancelled
	}

	// Step 3: Show tool details
	tool, exists := registry.GetTool(toolName)
	if !exists {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	DisplayToolDetails(toolName, tool)
	return nil
}
