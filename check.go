package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Audit registry against installed tools and local tool dirs",
	Long: `Checks two things:
  1. Which registered tools are missing from PATH
  2. Which tools in ~/tools/ and ~/dotfiles/apps/common/ are not in the registry`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s cannot determine home directory: %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

		missing := checkMissingFromPath(registry)
		unregistered := checkUnregisteredTools(home, registry)

		printCheckResults(missing, unregistered)
	},
}

// checkMissingFromPath returns registry tool names not found in PATH.
// Skips dotfiles-installed tools — those are shell functions that don't
// exist as files on PATH, only in a live shell session.
func checkMissingFromPath(reg *Registry) []string {
	var missing []string
	for name, tool := range reg.Tools {
		if tool.InstalledVia == "dotfiles" {
			continue
		}
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

// checkUnregisteredTools scans ~/tools/*/  and ~/dotfiles/apps/common/ for executables
// that are not in the registry.
func checkUnregisteredTools(home string, reg *Registry) []string {
	var unregistered []string
	seen := make(map[string]bool)

	// ~/tools/<repo>/ — each repo dir name is the tool name
	toolsDir := filepath.Join(home, "tools")
	entries, err := os.ReadDir(toolsDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && !reg.ToolExists(e.Name()) && !seen[e.Name()] {
				unregistered = append(unregistered, e.Name())
				seen[e.Name()] = true
			}
		}
	}

	// ~/dotfiles/apps/common/ — each file is a potential tool
	appsDir := filepath.Join(home, "dotfiles", "apps", "common")
	files, err := os.ReadDir(appsDir)
	if err == nil {
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if !reg.ToolExists(name) && !seen[name] {
				unregistered = append(unregistered, name)
				seen[name] = true
			}
		}
	}

	sort.Strings(unregistered)
	return unregistered
}

func printCheckResults(missing, unregistered []string) {
	fmt.Println(colorMagenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Println(colorMagenta(" toolbox check"))
	fmt.Println(colorMagenta("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Println()

	// Registered but not found in PATH
	fmt.Printf("%s  Registered tools missing from PATH (%d)\n", colorYellow("▸"), len(missing))
	if len(missing) == 0 {
		fmt.Printf("  %s all registered tools found in PATH\n", colorGreen("✓"))
	} else {
		for _, name := range missing {
			tool, _ := registry.GetTool(name)
			fmt.Printf("  %s  %-25s %s\n", colorRed("✗"), name, colorYellow("["+tool.InstalledVia+"]"))
		}
	}
	fmt.Println()

	// On disk but not in registry
	fmt.Printf("%s  Tools on disk not in registry (%d)\n", colorYellow("▸"), len(unregistered))
	if len(unregistered) == 0 {
		fmt.Printf("  %s all local tools are registered\n", colorGreen("✓"))
	} else {
		for _, name := range unregistered {
			fmt.Printf("  %s  %s\n", colorCyan("?"), name)
		}
	}
	fmt.Println()
}
