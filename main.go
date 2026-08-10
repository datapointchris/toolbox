package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"

	"github.com/datapointchris/goselfupdate/autoupdate"
	"github.com/datapointchris/goselfupdate/cobracmd"
	"github.com/spf13/cobra"
)

// ldflags injectable — set by GoReleaser: -X main.buildVersion={{.Version}}
var buildVersion string

func getVersion() string {
	if buildVersion != "" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}

// main.go - CLI entry point using cobra framework
// Cobra is like Python's click or argparse, but more powerful
// It handles subcommands, flags, help text, etc.

// This is a package-level variable (like a global in Python)
// We load the registry once and share it across commands
var registry *Registry

// rootCmd is the base command (just "toolbox")
var rootCmd = &cobra.Command{
	Use:     "toolbox",
	Short:   "Discover and explore your dotfiles tools",
	Version: getVersion(),
	Long: `Toolbox - Dotfiles Tool Discovery System

Discover and learn about the 98 tools in your dotfiles.
Search by name, category, or tags. Browse interactively with gum.`,
	// A lone unrecognized argument is a search query, so `toolbox git` works.
	// Cobra's default root validator rejects it as an unknown command, which is
	// why the shortcut needs an explicit Args — and having one is what lets it
	// live in RunE rather than ahead of Execute. It used to run in main against
	// a hand-written list of every command name, and anything missing from that
	// list was silently searched for instead: `toolbox completion` printed
	// search results rather than shell completions.
	Args: rootArgs,
	RunE: runRoot,
}

func rootArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return cobracmd.UsageError(fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath()))
	}
	return nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	if err := requireRegistryPreRun(cmd, args); err != nil {
		return err
	}
	query := args[0]
	DisplaySearchResults(SearchTools(registry, query), query)
	return nil
}

// listCmd shows all tools grouped by category
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all tools grouped by category",
	PreRunE: requireRegistryPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		DisplayListByCategory(registry)
	},
}

// showCmd displays detailed info about a specific tool
var showCmd = &cobra.Command{
	Use:     "show <tool>",
	Short:   "Show detailed information about a tool",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireRegistryPreRun,
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
	Use:     "search <query>",
	Short:   "Search tools by description, tags, or name",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireRegistryPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		results := SearchTools(registry, query)
		DisplaySearchResults(results, query)
	},
}

// categoriesCmd shows interactive category browser
var categoriesCmd = &cobra.Command{
	Use:     "categories",
	Short:   "Browse tools by category (interactive)",
	PreRunE: requireRegistryPreRun,
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

// funcsCmd lists shell functions parsed from dotfiles shell source files
var funcsCmd = &cobra.Command{
	Use:   "funcs [filter]",
	Short: "List shell functions from dotfiles",
	Long: `List shell functions defined in your dotfiles shell source files.

Parses functions.sh and $PLATFORM.sh from $SHELL_DIR (~/.local/shell/).
Functions are identified by the #@name / #--> description marker convention.

Optionally filter by name or description (case-insensitive).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}

		funcs, err := LoadShellFunctions()
		if err != nil {
			fmt.Printf("%s %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

		filtered := FilterFunctions(funcs, filter)
		DisplayShellFunctions(filtered, filter)
	},
}

// aliasesCmd lists shell aliases parsed from dotfiles shell source files
var aliasesCmd = &cobra.Command{
	Use:   "aliases [filter]",
	Short: "List shell aliases from dotfiles",
	Long: `List shell aliases defined in your dotfiles shell source files.

Parses aliases.sh and $PLATFORM.sh from $SHELL_DIR (~/.local/shell/).
The comment line immediately above each alias is used as its description.

Optionally filter by name, command, or description (case-insensitive).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}

		aliases, err := LoadShellAliases()
		if err != nil {
			fmt.Printf("%s %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

		filtered := FilterAliases(aliases, filter)
		DisplayShellAliases(filtered, filter)
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

// requireRegistryPreRun loads the registry on demand.
// Attach as PreRunE to commands that need the registry.
// Commands like update, funcs, and aliases skip this so they work
// even when the registry file is missing.
func requireRegistryPreRun(cmd *cobra.Command, args []string) error {
	if registry != nil {
		return nil
	}
	path := GetRegistryPath()
	var err error
	registry, err = LoadRegistry(path)
	if err != nil {
		return fmt.Errorf("%v\nExpected registry at: %s", err, path)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(categoriesCmd)
	rootCmd.AddCommand(remindCmd)
	rootCmd.AddCommand(funcsCmd)
	rootCmd.AddCommand(aliasesCmd)
}

// main is the program entry point
// In Python, this would be: if __name__ == "__main__":
func main() {
	autoConfig := autoupdate.Config{Update: updateConfig()}
	if err := cobracmd.Execute(context.Background(), rootCmd, autoConfig); err != nil {
		// 2 says the command line was wrong rather than the run, which is the
		// only failure a caller should retry with different arguments.
		if errors.Is(err, cobracmd.ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
