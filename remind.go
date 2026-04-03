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

		name, tool := pickRemindTarget(registry, recentlyUsed, reminders)
		if name == "" {
			return
		}

		fmt.Println()
		fmt.Printf("  %s — %s\n", colorYellow("reminder"), colorCyan(name))
		fmt.Printf("  %s\n", tool.Description)
		fmt.Println()

		reminders[name] = time.Now().Format("2006-01-02")
		saveReminders(remindersPath, reminders)
	},
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
	re := regexp.MustCompile(`^: (\d+):\d+;(\S+)`)

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

// pickRemindTarget returns the tool reminded least recently that isn't recently used.
func pickRemindTarget(reg *Registry, recentlyUsed map[string]bool, reminders remindersMap) (string, Tool) {
	names := make([]string, 0, len(reg.Tools))
	for name := range reg.Tools {
		names = append(names, name)
	}
	sort.Strings(names)

	bestName := ""
	bestDate := "9999-99-99"

	for _, name := range names {
		if recentlyUsed[name] {
			continue
		}
		last, ok := reminders[name]
		if !ok {
			last = "0000-00-00"
		}
		if strings.Compare(last, bestDate) < 0 {
			bestDate = last
			bestName = name
		}
	}

	if bestName == "" {
		return "", Tool{}
	}
	return bestName, reg.Tools[bestName]
}
