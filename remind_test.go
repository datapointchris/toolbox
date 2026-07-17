package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
