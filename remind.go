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
	Short: "Surface a forgotten tool, function, or alias",
	Long: `Cycles through everything you own — registry tools, shell functions,
shell aliases, git aliases, and forgit shortcuts — showing the one reminded
least recently that you have not used in the last 90 days. Tracks reminder
history in ~/dev/toolbox-reminders.json. Requires EXTENDED_HISTORY
(setopt EXTENDED_HISTORY in .zshrc) for the 90-day recency check.`,
	PreRunE: requireRegistryPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s cannot determine home directory: %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

		remindersPath := getRemindersPath(home)
		if err := os.MkdirAll(filepath.Dir(remindersPath), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%s cannot create state dir: %v\n", colorRed("Error:"), err)
			os.Exit(1)
		}

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
	if forgits, err := LoadForgitAliases(); err == nil {
		for _, fa := range forgits {
			cands = append(cands, remindTarget{name: fa.Name, kind: "forgit", al: fa})
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
		if t.fn.Body != "" {
			fmt.Println()
			for _, line := range strings.Split(t.fn.Body, "\n") {
				fmt.Printf("    %s\n", line)
			}
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
	case "forgit":
		fmt.Printf("  %s — %s  %s\n", colorYellow("reminder"), colorCyan(t.al.Name), colorGreen("(forgit)"))
		fmt.Printf("  %s interactive git %s, via fzf\n", colorYellow("↳"), t.al.Command)
	default: // tool
		// Show the full detail — the same view `toolbox show` and the interactive
		// menu render — rather than a stripped-down card, so a reminder is a proper
		// refresher. DisplayToolDetails prints its own boxed header with the name.
		fmt.Printf("  %s — you haven't reached for %s lately:\n", colorYellow("reminder"), colorCyan(t.name))
		fmt.Println()
		DisplayToolDetails(t.name, t.tool)
	}
}

// getRemindersPath returns the path to the reminder-history file. Checks
// TOOLBOX_REMINDERS first, then falls back to ~/dev/toolbox-reminders.json.
// Like the registry (GetRegistryPath), the last-reminded dates are Syncthing-
// synced data, not machine-specific state: they follow you across machines so
// the round-robin doesn't restart on each one — hence ~/dev/, not ~/.local/state.
func getRemindersPath(home string) string {
	if path := os.Getenv("TOOLBOX_REMINDERS"); path != "" {
		return path
	}
	return filepath.Join(home, "dev", "toolbox-reminders.json")
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
