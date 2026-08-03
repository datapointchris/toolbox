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

// briefWidth caps each line of a --brief card. A fixed width rather than the
// real terminal size: reading that needs golang.org/x/term, and one constant
// narrow enough for any usable pane is not worth a dependency. Without it a
// long example wraps and the card silently costs the extra line --brief exists
// to save.
const briefWidth = 72

// remindBrief bounds the card to a few lines. Set by the shell-startup nudge
// (`menu review`'s register runs `toolbox remind --brief`), where the full
// detail view would bury the rest of the nudge.
var remindBrief bool

var remindCmd = &cobra.Command{
	Use:   "remind",
	Short: "Surface a forgotten tool, function, or alias",
	Long: `Cycles through everything you own — registry tools, shell functions,
shell aliases, git aliases, and forgit shortcuts — showing the one reminded
least recently that you have not used in the last 90 days. Tracks reminder
history under the XDG state dir. Requires EXTENDED_HISTORY
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

		// The brief card is one line inside a larger nudge, so it skips the
		// breathing room a standalone card wants.
		if !remindBrief {
			fmt.Println()
		}
		renderReminder(target, remindBrief)
		if !remindBrief {
			fmt.Println()
		}

		reminders[target.name] = time.Now().Format("2006-01-02")
		saveReminders(remindersPath, reminders)
	},
}

func init() {
	remindCmd.Flags().BoolVar(&remindBrief, "brief", false, "Render a few lines instead of the full detail view")
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
	if aliases, err := LoadShellAliases(); err == nil {
		for _, alias := range aliases {
			cands = append(cands, remindTarget{name: alias.Name, kind: "alias", al: alias})
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
//
// Brief is a hard bound, not a hint: at most three lines, none wider than
// briefWidth, for every kind. Two branches are otherwise unbounded — a tool
// renders the whole detail view, and a function prints its entire body — and
// either can swamp the shell-startup nudge this card sits inside.
func renderReminder(t remindTarget, brief bool) {
	// The title line is the same shape for every kind; only the label differs.
	title := func(name, kind string) {
		fmt.Printf("  %s — %s  %s\n", colorYellow("reminder"), colorCyan(name), colorGreen("("+kind+")"))
	}
	detail := func(text string, indent int) {
		if text == "" {
			return
		}
		if brief {
			text = clipBrief(text, indent)
		}
		fmt.Printf("  %s\n", text)
	}
	arrow := func(text string) {
		if brief {
			text = clipBrief(text, 4)
		}
		fmt.Printf("  %s %s\n", colorYellow("↳"), text)
	}

	switch t.kind {
	case "function":
		title(t.name, "shell function")
		detail(t.fn.Description, 2)
		// The body is the refresher, and it is however long the function is —
		// so it is exactly what brief drops. `toolbox remind` on demand shows it.
		if t.fn.Body != "" && !brief {
			fmt.Println()
			for _, line := range strings.Split(t.fn.Body, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	case "alias":
		title(t.name, "alias")
		detail(t.al.Description, 2)
		arrow(t.al.Command)
	case "gitalias":
		title("git "+t.al.Name, "git alias")
		arrow("git " + t.al.Command)
	case "forgit":
		title(t.al.Name, "forgit")
		arrow(fmt.Sprintf("interactive git %s, via fzf", t.al.Command))
	default: // tool
		if brief {
			title(t.name, "tool")
			detail(t.tool.Description, 2)
			if len(t.tool.Examples) > 0 {
				arrow(t.tool.Examples[0].Cmd)
			}
			return
		}
		// Show the full detail — the same view `toolbox show` and the interactive
		// menu render — rather than a stripped-down card, so a reminder is a proper
		// refresher. DisplayToolDetails prints its own boxed header with the name.
		fmt.Printf("  %s — you haven't reached for %s lately:\n", colorYellow("reminder"), colorCyan(t.name))
		fmt.Println()
		DisplayToolDetails(t.name, t.tool)
	}
}

// clipBrief shortens s to fit briefWidth alongside indent columns of prefix.
// Measured in runes, not bytes, so a byte slice can't split a multi-byte
// character and print a replacement glyph.
func clipBrief(s string, indent int) string {
	runes := []rune(s)
	room := briefWidth - indent
	if len(runes) <= room || room < 2 {
		return s
	}
	return string(runes[:room-1]) + "…"
}

// getRemindersPath returns the path to the reminder-history file. Checks
// TOOLBOX_REMINDERS first, then falls back to the XDG state dir.
// The last-reminded dates are mutable state, written as the round-robin
// advances, so they live under the XDG state dir. They still follow you across
// machines (so the rotation doesn't restart on each) — but that replication is
// the sync layer's job, not something this path encodes.
func getRemindersPath(_ string) string {
	if path := os.Getenv("TOOLBOX_REMINDERS"); path != "" {
		return path
	}
	return filepath.Join(xdgStateHome(), "toolbox", "reminders.json")
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
