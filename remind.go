package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const cutoffDays = 90

var remindCmd = &cobra.Command{
	Use:   "remind",
	Short: "Surface a forgotten tool from the registry",
	Long: `Cycles through all registry tools at shell startup, showing the one
reminded least recently that hasn't been used in the last 90 days.
Tracks reminder history in ~/.local/state/toolbox/reminders.json.
Requires EXTENDED_HISTORY (setopt EXTENDED_HISTORY in .zshrc) for the
90-day recency check.`,
	PreRunE: requireRegistryPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s cannot determine home directory: %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

		stateDir := getStateDir(home)
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%s cannot create state dir: %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}
		remindersPath := filepath.Join(stateDir, "reminders.json")

		recentlyUsed := parseRecentlyUsed(home)
		reminders := loadReminders(remindersPath)

		candidates := buildRemindCandidates(registry)
		target, ok := pickRemindTarget(candidates, recentlyUsed, reminders)
		if !ok {
			return
		}

		fmt.Println()
		renderReminder(target)
		fmt.Println()

		reminders[target.name] = time.Now().Format("2006-01-02")
		saveReminders(remindersPath, reminders)
	},
}

// remindTarget is one thing worth resurfacing — a registry tool, a shell
// function, or an alias. remind rotates over all three so a forgotten fzf
// function surfaces the same way a forgotten tool does.
type remindTarget struct {
	name string
	kind string // "tool", "function", "alias"
	tool Tool
	fn   ShellFunction
	al   ShellAlias
}

// buildRemindCandidates pools the registry tools with the annotated shell
// functions and aliases. Shell parsing failures are non-fatal — a missing
// functions.sh just means fewer candidates, not a broken reminder.
func buildRemindCandidates(reg *Registry) []remindTarget {
	var cands []remindTarget
	for name, tool := range reg.Tools {
		cands = append(cands, remindTarget{name: name, kind: "tool", tool: tool})
	}
	if fns, err := LoadShellFunctions(); err == nil {
		for _, fn := range fns {
			cands = append(cands, remindTarget{name: fn.Name, kind: "function", fn: fn})
		}
	}
	if als, err := LoadShellAliases(); err == nil {
		for _, al := range als {
			cands = append(cands, remindTarget{name: al.Name, kind: "alias", al: al})
		}
	}
	if gits, err := LoadGitAliases(); err == nil {
		for _, g := range gits {
			cands = append(cands, remindTarget{name: g.Name, kind: "gitalias", al: g})
		}
	}
	return cands
}

// renderReminder prints the card for whichever kind of thing was picked.
func renderReminder(t remindTarget) {
	switch t.kind {
	case "function":
		fmt.Printf("  %s — %s  %s\n", colorYellow("reminder"), colorCyan(t.name), colorGreen("(shell function)"))
		if t.fn.Description != "" {
			fmt.Printf("  %s\n", t.fn.Description)
		}
	case "alias":
		fmt.Printf("  %s — %s  %s\n", colorYellow("reminder"), colorCyan(t.name), colorGreen("(alias)"))
		if t.al.Description != "" {
			fmt.Printf("  %s\n", t.al.Description)
		}
		fmt.Printf("  %s %s\n", colorYellow("↳"), t.al.Command)
	case "gitalias":
		fmt.Printf("  %s — %s  %s\n", colorYellow("reminder"), colorCyan("git "+t.al.Name), colorGreen("(git alias)"))
		fmt.Printf("  %s git %s\n", colorYellow("↳"), t.al.Command)
	default: // tool
		fmt.Printf("  %s — %s\n", colorYellow("reminder"), colorCyan(t.name))
		fmt.Printf("  %s\n", t.tool.Description)
		if t.tool.WhyUse != "" {
			fmt.Printf("  %s %s\n", colorYellow("↳"), t.tool.WhyUse)
		}
		if len(t.tool.Examples) > 0 {
			fmt.Println()
			for _, ex := range t.tool.Examples {
				fmt.Printf("    %s %s\n", colorGreen("$"), ex.Cmd)
				fmt.Printf("      %s\n", ex.Desc)
			}
		}
	}
}

func getStateDir(home string) string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "toolbox")
	}
	return filepath.Join(home, ".local", "state", "toolbox")
}

// parseRecentlyUsed reads EXTENDED_HISTORY and returns tools used within cutoffDays.
// EXTENDED_HISTORY format: ": timestamp:elapsed;command args..."
func parseRecentlyUsed(home string) map[string]bool {
	used := make(map[string]bool)

	histFile := os.Getenv("HISTFILE")
	if histFile == "" {
		histFile = filepath.Join(home, ".local", "state", "zsh", "history")
	}

	f, err := os.Open(histFile)
	if err != nil {
		return used
	}
	defer func() { _ = f.Close() }()

	cutoff := time.Now().Unix() - cutoffDays*86400
	// Capture the command and an optional second word. The second word matters
	// for git aliases: they are invoked as `git co`, so without it a git alias is
	// never seen as used and would resurface forever.
	re := regexp.MustCompile(`^: (\d+):\d+;(\S+)(?:\s+(\S+))?`)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		m := re.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		ts, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || ts <= cutoff {
			continue
		}
		used[m[2]] = true
		// `git co` marks the git alias `co` as used, not just `git`.
		if m[2] == "git" && m[3] != "" {
			used[m[3]] = true
		}
	}
	return used
}

type remindersMap map[string]string

func loadReminders(path string) remindersMap {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(remindersMap)
	}
	var r remindersMap
	if err := json.Unmarshal(data, &r); err != nil {
		return make(remindersMap)
	}
	return r
}

func saveReminders(path string, r remindersMap) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	// Atomic rename so a crash mid-write doesn't corrupt the file
	_ = os.Rename(tmp, path)
}

// pickRemindTarget returns the candidate reminded least recently that isn't
// recently used, across tools, functions, and aliases alike. Candidates are
// sorted by name first so the choice is deterministic when reminder dates tie.
func pickRemindTarget(candidates []remindTarget, recentlyUsed map[string]bool, reminders remindersMap) (remindTarget, bool) {
	sorted := make([]remindTarget, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })

	best := remindTarget{}
	bestDate := "9999-99-99"
	found := false

	for _, c := range sorted {
		if recentlyUsed[c.name] {
			continue
		}
		last, ok := reminders[c.name]
		if !ok {
			last = "0000-00-00"
		}
		if strings.Compare(last, bestDate) < 0 {
			bestDate = last
			best = c
			found = true
		}
	}

	return best, found
}
