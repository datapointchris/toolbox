package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		renderReminder(remindTarget{name: "mytool", kind: "tool", tool: tool})
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

// TestRenderReminderFunctionShowsBody confirms a function reminder now includes
// the captured definition, not just the one-line description.
func TestRenderReminderFunctionShowsBody(t *testing.T) {
	fn := ShellFunction{
		Name:        "mkcd",
		Description: "make a dir and cd into it",
		Body:        "mkcd() {\n  mkdir -p \"$1\" && cd \"$1\"\n}",
	}
	out := captureStdout(t, func() {
		renderReminder(remindTarget{name: "mkcd", kind: "function", fn: fn})
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
