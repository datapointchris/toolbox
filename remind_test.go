package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// captureStdout runs f with os.Stdout redirected to a pipe and returns what it
// printed. renderReminder and DisplayToolDetails write straight to stdout, so
// this is how we assert on their output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy: %v", err)
	}
	return buf.String()
}

// TestRenderReminderToolShowsFullDetail locks in the fix: a tool reminder renders
// the same full field set the menu shows (via DisplayToolDetails), not a subset.
func TestRenderReminderToolShowsFullDetail(t *testing.T) {
	tool := Tool{
		Category:     "search",
		Description:  "ripgrep-like search",
		InstalledVia: "brew",
		Usage:        "mytool [flags] <pattern>",
		WhyUse:       "fast recursive search",
		Notes:        "respects .gitignore by default",
		Examples:     []Example{{Cmd: "mytool foo", Desc: "search for foo"}},
		SeeAlso:      []string{"grep", "ag"},
		Tags:         []string{"cli", "search"},
		DocsURL:      "https://example.com/mytool",
	}
	out := captureStdout(t, func() {
		renderReminder(remindTarget{name: "mytool", kind: "tool", tool: tool}, false)
	})

	// Fields the old stripped card omitted must now all be present.
	for _, want := range []string{
		"mytool", "reminder",
		"Description:", "ripgrep-like search",
		"Category:", "search",
		"Usage:", "mytool [flags] <pattern>",
		"Notes:", "respects .gitignore by default",
		"See also:", "grep",
		"Tags:", "cli",
		"Documentation:", "https://example.com/mytool",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("tool reminder missing %q\n---\n%s", want, out)
		}
	}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// TestRenderReminderToolBriefStaysShort locks the bound the shell-startup nudge
// depends on: --brief must not reach the full detail view however rich the tool,
// and no line may wrap. Fields are deliberately longer than briefWidth so the
// clip is exercised rather than merely not triggered.
func TestRenderReminderToolBriefStaysShort(t *testing.T) {
	tool := Tool{
		Category:     "search",
		Description:  strings.Repeat("a very wordy description ", 6),
		InstalledVia: "brew",
		Usage:        "mytool [flags] <pattern>",
		WhyUse:       "fast recursive search",
		Notes:        "respects .gitignore by default",
		Examples:     []Example{{Cmd: "mytool foo " + strings.Repeat("--flag ", 20), Desc: "search for foo"}},
		SeeAlso:      []string{"grep", "ag"},
		Tags:         []string{"cli", "search"},
		DocsURL:      "https://example.com/mytool",
	}
	out := captureStdout(t, func() {
		renderReminder(remindTarget{name: "mytool", kind: "tool", tool: tool}, true)
	})

	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if n := len([]rune(stripANSI(line))); n > briefWidth {
			t.Errorf("brief line is %d cols, want <= %d: %q", n, briefWidth, line)
		}
	}
	if lines := len(strings.Split(strings.TrimRight(out, "\n"), "\n")); lines > 3 {
		t.Errorf("brief tool reminder is %d lines, want <= 3\n---\n%s", lines, out)
	}
	for _, want := range []string{"reminder", "mytool", "a very wordy description", "mytool foo"} {
		if !strings.Contains(out, want) {
			t.Errorf("brief tool reminder missing %q\n---\n%s", want, out)
		}
	}
	// The detail view's field labels are what make it long; none may appear.
	for _, unwanted := range []string{"Usage:", "Notes:", "See also:", "Documentation:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("brief tool reminder leaked detail field %q\n---\n%s", unwanted, out)
		}
	}
}

// TestRenderReminderBriefBoundsEveryKind is the contract the startup nudge rests
// on. Every field is longer than briefWidth and the function body is many lines,
// so a kind that forgets to clip fails here rather than in a shell one morning.
func TestRenderReminderBriefBoundsEveryKind(t *testing.T) {
	long := strings.Repeat("wordy ", 40)
	al := ShellAlias{Name: "al", Description: long, Command: long}
	kinds := map[string]remindTarget{
		"tool":     {name: "tl", kind: "tool", tool: Tool{Description: long, Examples: []Example{{Cmd: long}}}},
		"function": {name: "fn", kind: "function", fn: ShellFunction{Description: long, Body: strings.Repeat("x\n", 30)}},
		"alias":    {name: "al", kind: "alias", al: al},
		"gitalias": {name: "al", kind: "gitalias", al: al},
		"forgit":   {name: "al", kind: "forgit", al: al},
	}

	for kind, target := range kinds {
		t.Run(kind, func(t *testing.T) {
			out := captureStdout(t, func() { renderReminder(target, true) })
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) > 3 {
				t.Errorf("%s brief card is %d lines, want <= 3\n---\n%s", kind, len(lines), out)
			}
			for _, line := range lines {
				if n := len([]rune(stripANSI(line))); n > briefWidth {
					t.Errorf("%s brief line is %d cols, want <= %d: %q", kind, n, briefWidth, line)
				}
			}
		})
	}
}

// TestRenderReminderFunctionShowsBody confirms a function reminder now includes
// the captured definition, not just the one-line description.
func TestRenderReminderFunctionShowsBody(t *testing.T) {
	fn := ShellFunction{
		Name:        "mkcd",
		Description: "make a dir and cd into it",
		Body:        "mkcd() {\n  mkdir -p \"$1\" && cd \"$1\"\n}",
	}
	out := captureStdout(t, func() {
		renderReminder(remindTarget{name: "mkcd", kind: "function", fn: fn}, false)
	})
	for _, want := range []string{"mkcd", "make a dir and cd into it", "mkdir -p", "cd \"$1\""} {
		if !strings.Contains(out, want) {
			t.Errorf("function reminder missing %q\n---\n%s", want, out)
		}
	}
}

// remind_test.go - Tests for the two behaviorally subtle pieces of remind:
// parsing zsh EXTENDED_HISTORY (including the `git <alias>` second-word case)
// and selecting the least-recently-reminded, not-recently-used candidate.

// TestParseRecentlyUsed verifies the 90-day cutoff and the git-alias second-word
// rule: `git co` must mark the alias `co` as used, not just `git`, or a git alias
// would never be seen as used and would resurface forever.
func TestParseRecentlyUsed(t *testing.T) {
	now := time.Now().Unix()
	recent := now - 10*86400 // within the 90-day window
	old := now - 200*86400   // well outside it

	content := fmt.Sprintf(`: %d:0;rg pattern
: %d:0;ancienttool bar
: %d:0;git co feature-branch
not a history line
: %d:5;fd --type f
`, recent, old, recent, recent)

	histFile := filepath.Join(t.TempDir(), "history")
	if err := os.WriteFile(histFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}
	t.Setenv("HISTFILE", histFile)

	used := parseRecentlyUsed("/nonexistent-home")

	if !used["rg"] {
		t.Error("expected recent `rg` to be marked used")
	}
	if !used["fd"] {
		t.Error("expected recent `fd` to be marked used")
	}
	if used["ancienttool"] {
		t.Error("expected 200-day-old `ancienttool` to be outside the cutoff")
	}
	// The core git-alias case:
	if !used["git"] {
		t.Error("expected `git` to be marked used")
	}
	if !used["co"] {
		t.Error("expected `git co` to also mark the alias `co` used")
	}
}

// TestParseRecentlyUsedMissingFile confirms a missing HISTFILE is non-fatal —
// it just yields an empty set rather than erroring.
func TestParseRecentlyUsedMissingFile(t *testing.T) {
	t.Setenv("HISTFILE", filepath.Join(t.TempDir(), "does-not-exist"))
	used := parseRecentlyUsed("/nonexistent-home")
	if len(used) != 0 {
		t.Errorf("expected empty set for missing history, got %v", used)
	}
}

func TestPickRemindTarget(t *testing.T) {
	cands := []remindTarget{
		{name: "alpha", kind: "tool"},
		{name: "bravo", kind: "tool"},
		{name: "charlie", kind: "tool"},
	}

	t.Run("never-reminded beats previously-reminded", func(t *testing.T) {
		// alpha was reminded recently; bravo/charlie never have been (→ 0000-00-00).
		reminders := remindersMap{"alpha": "2026-07-01"}
		got, ok := pickRemindTarget(cands, map[string]bool{}, reminders)
		if !ok {
			t.Fatal("expected a target")
		}
		// bravo and charlie tie at 0000-00-00; name sort breaks the tie → bravo.
		if got.name != "bravo" {
			t.Errorf("got %q, want bravo", got.name)
		}
	})

	t.Run("recently-used is skipped", func(t *testing.T) {
		// All never reminded; alpha would win by name but is recently used.
		got, ok := pickRemindTarget(cands, map[string]bool{"alpha": true}, remindersMap{})
		if !ok {
			t.Fatal("expected a target")
		}
		if got.name != "bravo" {
			t.Errorf("got %q, want bravo (alpha skipped as used)", got.name)
		}
	})

	t.Run("oldest reminder date wins", func(t *testing.T) {
		reminders := remindersMap{
			"alpha":   "2026-06-01",
			"bravo":   "2026-05-01", // oldest → should win
			"charlie": "2026-07-01",
		}
		got, ok := pickRemindTarget(cands, map[string]bool{}, reminders)
		if !ok {
			t.Fatal("expected a target")
		}
		if got.name != "bravo" {
			t.Errorf("got %q, want bravo (oldest reminder)", got.name)
		}
	})

	t.Run("all recently used yields nothing", func(t *testing.T) {
		used := map[string]bool{"alpha": true, "bravo": true, "charlie": true}
		if _, ok := pickRemindTarget(cands, used, remindersMap{}); ok {
			t.Error("expected no target when every candidate is recently used")
		}
	})

	t.Run("empty candidate set yields nothing", func(t *testing.T) {
		if _, ok := pickRemindTarget(nil, map[string]bool{}, remindersMap{}); ok {
			t.Error("expected no target for empty candidates")
		}
	})
}
