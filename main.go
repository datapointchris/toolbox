package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// main.go - CLI entry point using cobra framework
// Cobra is like Python's click or argparse, but more powerful
// It handles subcommands, flags, help text, etc.

var (
	// This is a package-level variable (like a global in Python)
	// We load the registry once and share it across commands
	registry *Registry
)

// rootCmd is the base command (just "toolbox")
var rootCmd = &cobra.Command{
	Use:   "toolbox",
	Short: "Discover and explore your dotfiles tools",
	Long: `Toolbox - Dotfiles Tool Discovery System

Discover and learn about the 98 tools in your dotfiles.
Search by name, category, or tags. Browse interactively with gum.`,
	// Run is called when command is executed without subcommands
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help() // Show help by default
	},
}

// listCmd shows all tools grouped by category
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tools grouped by category",
	Run: func(cmd *cobra.Command, args []string) {
		DisplayListByCategory(registry)
	},
}

// showCmd displays detailed info about a specific tool
var showCmd = &cobra.Command{
	Use:   "show <tool>",
	Short: "Show detailed information about a tool",
	Args:  cobra.ExactArgs(1), // Require exactly 1 argument
	Run: func(cmd *cobra.Command, args []string) {
		toolName := args[0]

		tool, exists := registry.GetTool(toolName)
		if !exists {
			fmt.Printf("%s Tool '%s' not found in registry\n", colorRed("Error:"), toolName)
			fmt.Println()
			fmt.Println("Use " + colorCyan("toolbox list") + " to see all available tools")
			os.Exit(1)
		}

		DisplayToolDetails(toolName, tool)
	},
}

// searchCmd searches for tools by query
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tools by description, tags, or name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		results := SearchTools(registry, query)
		DisplaySearchResults(results, query)
	},
}

// categoriesCmd shows interactive category browser
var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Browse tools by category (interactive)",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if gum is installed
		// In Python: if not shutil.which("gum")
		if err := checkGumInstalled(); err != nil {
			fmt.Printf("%s gum is not installed. Use: brew install gum\n", colorRed("Error:"))
			os.Exit(1)
		}

		if err := InteractiveCategoryBrowse(registry); err != nil {
			fmt.Printf("%s %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}
	},
}

// checkGumInstalled verifies gum is available in PATH
func checkGumInstalled() error {
	// exec.LookPath is like Python's shutil.which()
	if _, err := exec.LookPath("gum"); err != nil {
		return fmt.Errorf("gum not found in PATH")
	}
	return nil
}

// init runs before main() - used for setup
// In Python, this would be module-level code
func init() {
	// Load registry
	path := GetRegistryPath()

	var err error
	registry, err = LoadRegistry(path)
	if err != nil {
		// Fatal error - can't continue without registry
		fmt.Fprintf(os.Stderr, "%s %v\n", colorRed("Error:"), err)
		fmt.Fprintf(os.Stderr, "Expected registry at: %s\n", path)
		os.Exit(1)
	}

	// Add subcommands to root
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(categoriesCmd)
}

// main is the program entry point
// In Python, this would be: if __name__ == "__main__":
func main() {
	// Check if there's a single argument that isn't a known command
	// Treat it as a search query (like "toolbox git")
	if len(os.Args) == 2 {
		arg := os.Args[1]

		// Check if it's a known command
		// In Python: if arg not in ["list", "show", "search", "categories", "help"]
		knownCommands := []string{"list", "show", "search", "categories", "help", "--help", "-h"}
		isKnownCommand := false
		for _, cmd := range knownCommands {
			if arg == cmd {
				isKnownCommand = true
				break
			}
		}

		// If not a known command, treat as search query
		if !isKnownCommand {
			results := SearchTools(registry, arg)
			DisplaySearchResults(results, arg)
			return
		}
	}

	// Execute the root command
	// This handles all subcommands, flags, help, etc.
	// In Python with click: if __name__ == "__main__": cli()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
