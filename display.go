package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// display.go - Functions for displaying tool information with colors
// These use ANSI color codes - same as Python's colorama or rich

// ANSI color codes
// In Python, you might use colorama.Fore.BLUE, etc.
// Using bright variants (90-97) for better visibility
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[91m"    // Bright red
	ansiGreen  = "\033[92m"    // Bright green
	ansiYellow = "\033[93m"    // Bright yellow
	ansiBlue   = "\033[94m"    // Bright blue
	ansiCyan   = "\033[96m"    // Bright cyan
	ansiMagenta = "\033[95m"   // Bright magenta
	ansiBold   = "\033[1m"
)

// Color helper functions - wrap text in color codes
// In Python: def color_blue(text: str) -> str
func colorBlue(text string) string {
	return ansiBlue + text + ansiReset
}

func colorGreen(text string) string {
	return ansiGreen + text + ansiReset
}

func colorYellow(text string) string {
	return ansiYellow + text + ansiReset
}

func colorCyan(text string) string {
	return ansiCyan + text + ansiReset
}

func colorRed(text string) string {
	return ansiRed + text + ansiReset
}

func colorBold(text string) string {
	return ansiBold + text + ansiReset
}

func colorMagenta(text string) string {
	return ansiMagenta + text + ansiReset
}

// DisplayToolDetails shows detailed information about a single tool
// In Python: def display_tool_details(name: str, tool: Tool) -> None
func DisplayToolDetails(name string, tool Tool) {
	// Print with formatting (%s = string placeholder)
	// In Python: print(f"...")
	fmt.Println(colorMagenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Println(colorMagenta(" " + name))
	fmt.Println(colorMagenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Println()

	// Description
	fmt.Println(colorYellow("Description:"))
	fmt.Printf("  %s\n", tool.Description)
	fmt.Println()

	// Why use (if present)
	if tool.WhyUse != "" {
		fmt.Println(colorYellow("Why Use:"))
		fmt.Printf("  %s\n", tool.WhyUse)
		fmt.Println()
	}

	// Category and install method
	fmt.Printf("%s %s\n", colorYellow("Category:"), tool.Category)
	fmt.Printf("%s %s\n", colorYellow("Installed via:"), tool.InstalledVia)
	fmt.Println()

	// Usage
	fmt.Println(colorYellow("Usage:"))
	fmt.Printf("  %s\n", tool.Usage)
	fmt.Println()

	// Examples
	if len(tool.Examples) > 0 { // len() works on slices, maps, strings
		fmt.Println(colorYellow("Examples:"))
		for _, example := range tool.Examples {
			fmt.Printf("  $ %s\n", example.Cmd)
			fmt.Printf("    %s\n", example.Desc)
		}
		fmt.Println()
	}

	// See also
	if len(tool.SeeAlso) > 0 {
		// strings.Join is like Python's ", ".join(list)
		fmt.Printf("%s %s\n", colorYellow("See also:"), strings.Join(tool.SeeAlso, ", "))
		fmt.Println()
	}

	// Tags
	if len(tool.Tags) > 0 {
		fmt.Printf("%s %s\n", colorYellow("Tags:"), strings.Join(tool.Tags, ", "))
		fmt.Println()
	}

	// Documentation URL
	if tool.DocsURL != "" {
		fmt.Printf("%s %s\n", colorYellow("Documentation:"), tool.DocsURL)
		fmt.Println()
	}

	// Check if command exists in PATH
	// exec.LookPath is like Python's shutil.which()
	if _, err := exec.LookPath(name); err == nil {
		fmt.Printf("%s Installed in PATH\n", colorGreen("✓"))
	} else {
		fmt.Printf("%s Shell function (source from dotfiles)\n", colorYellow("⚠"))
	}

	fmt.Println(colorMagenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
}

// DisplaySearchResults shows search results in a compact list
func DisplaySearchResults(results []SearchResult, query string) {
	if len(results) == 0 {
		fmt.Printf("%s No tools found matching '%s'\n", colorRed("✗ Error:"), query)
		fmt.Println()
		fmt.Println(colorYellow("💡 TIP:") + " Try searching for:")
		fmt.Printf("  - Category: %s\n", colorCyan("toolbox search file-management"))
		fmt.Printf("  - Tag: %s\n", colorCyan("toolbox search git"))
		fmt.Printf("  - Feature: %s\n", colorCyan("toolbox search syntax"))
		return
	}

	fmt.Printf("%s %s\n", colorMagenta("Search Results for:"), query)
	fmt.Println()

	for _, result := range results {
		// %-25s means left-align in 25 character field
		// In Python: f"{name:<25}"
		fmt.Printf("  %-25s %s %s\n",
			colorCyan(result.Name),
			colorYellow("["+result.Tool.Category+"]"),
			result.Tool.Description)
	}
}

// DisplayListByCategory shows all tools grouped by category
func DisplayListByCategory(registry *Registry) {
	categories := GetCategoriesSorted(registry)

	fmt.Println(colorMagenta("Installed Tools") + fmt.Sprintf(" (%d total)", len(registry.Tools)))
	fmt.Println()

	for _, category := range categories {
		tools := GetToolsByCategory(registry, category)

		// Print category header
		fmt.Println(colorYellow(category) + fmt.Sprintf(" (%d tools)", len(tools)))

		// Print tools in this category
		for _, result := range tools {
			fmt.Printf("  %-25s %s\n", colorCyan(result.Name), result.Tool.Description)
		}
		fmt.Println()
	}

	fmt.Println(colorYellow("💡 TIP:") + " Use " + colorCyan("toolbox show <name>") + " to see detailed info and examples")
}

// DisplayCategories shows just the category names with counts
func DisplayCategories(registry *Registry) {
	categories := GetCategoriesSorted(registry)
	categoryMap := registry.ByCategory()

	fmt.Println(colorMagenta("Tool Categories"))
	fmt.Println()

	for _, category := range categories {
		count := len(categoryMap[category])
		fmt.Printf("  %-25s %d tools\n", colorCyan(category), count)
	}

	fmt.Println()
	fmt.Println(colorYellow("💡 TIP:") + " Use " + colorCyan("toolbox list") + " to see tools by category")
}
