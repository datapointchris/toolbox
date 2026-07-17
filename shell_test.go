package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shell_test.go - Tests for shell source parsing (functions, aliases, forgit).
// The parsers consume real dotfiles shell files, so these tests write small
// fixture files to a temp dir and assert on the parsed structs. This locks down
// the marker-adjacency and comment-buffer rules that are easy to break silently.

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return path
}

// TestParseFunctions checks the #@name / #--> desc marker pair, including the
// rule that the two markers must be on consecutive lines — any line between
// them resets the pending name.
func TestParseFunctions(t *testing.T) {
	content := `#@foo
#--> Does foo
function foo() { :; }

#@bar
some code line
#--> orphaned description

#@baz
#--> Baz it
`
	funcs, err := ParseFunctions(writeTempFile(t, "functions.sh", content))
	if err != nil {
		t.Fatalf("ParseFunctions: %v", err)
	}

	// bar is dropped: a non-marker line separates #@bar from its #--> line.
	if len(funcs) != 2 {
		t.Fatalf("expected 2 functions, got %d: %+v", len(funcs), funcs)
	}
	if funcs[0].Name != "foo" || funcs[0].Description != "Does foo" {
		t.Errorf("func 0 = %+v, want foo/Does foo", funcs[0])
	}
	if funcs[1].Name != "baz" || funcs[1].Description != "Baz it" {
		t.Errorf("func 1 = %+v, want baz/Baz it", funcs[1])
	}
	if funcs[0].Source != "functions.sh" {
		t.Errorf("source = %q, want functions.sh", funcs[0].Source)
	}
}

// TestParseFunctionsBody checks that the definition is captured from the
// `name() {` line through the closing `}`, and that a following annotation ends
// the previous body cleanly.
func TestParseFunctionsBody(t *testing.T) {
	content := `#@mkcd
#--> Make a dir and cd into it
mkcd() {
  mkdir -p "$1" && cd "$1"
}

#@greet
#--> Say hi
greet() {
  echo "hi $1"
}
`
	funcs, err := ParseFunctions(writeTempFile(t, "functions.sh", content))
	if err != nil {
		t.Fatalf("ParseFunctions: %v", err)
	}
	if len(funcs) != 2 {
		t.Fatalf("expected 2 functions, got %d: %+v", len(funcs), funcs)
	}
	wantBody := "mkcd() {\n  mkdir -p \"$1\" && cd \"$1\"\n}"
	if funcs[0].Body != wantBody {
		t.Errorf("mkcd body = %q, want %q", funcs[0].Body, wantBody)
	}
	if funcs[1].Name != "greet" || !strings.Contains(funcs[1].Body, `echo "hi $1"`) {
		t.Errorf("greet = %+v, want body containing the echo line", funcs[1])
	}
}

// TestParseAliases checks comment-as-description capture and the reset rules:
// a blank line or an intervening alias clears the pending comment.
func TestParseAliases(t *testing.T) {
	content := `# Says hello
alias hi='echo hello'

alias bare=ls

# stale comment

alias nodesc="echo hi"
alias eq='a=b'
`
	aliases, err := ParseAliases(writeTempFile(t, "aliases.sh", content))
	if err != nil {
		t.Fatalf("ParseAliases: %v", err)
	}

	want := []ShellAlias{
		{Name: "hi", Command: "echo hello", Description: "Says hello"},
		{Name: "bare", Command: "ls", Description: ""},
		// blank line after "# stale comment" clears the buffer
		{Name: "nodesc", Command: "echo hi", Description: ""},
		// only the first '=' splits, so the command keeps its own '='
		{Name: "eq", Command: "a=b", Description: ""},
	}
	if len(aliases) != len(want) {
		t.Fatalf("expected %d aliases, got %d: %+v", len(want), len(aliases), aliases)
	}
	for i, w := range want {
		got := aliases[i]
		if got.Name != w.Name || got.Command != w.Command || got.Description != w.Description {
			t.Errorf("alias %d = {%q %q %q}, want {%q %q %q}",
				i, got.Name, got.Command, got.Description, w.Name, w.Command, w.Description)
		}
	}
}

// TestSplitAlias covers the quote-stripping and first-'=' split.
func TestSplitAlias(t *testing.T) {
	tests := []struct {
		body     string
		wantName string
		wantCmd  string
	}{
		{"n='cmd'", "n", "cmd"},
		{`n="cmd"`, "n", "cmd"},
		{"n=cmd", "n", "cmd"},
		{"n='a=b'", "n", "a=b"},
		{"noequals", "", ""},
		{"gs='git status'", "gs", "git status"},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			name, cmd := splitAlias(tt.body)
			if name != tt.wantName || cmd != tt.wantCmd {
				t.Errorf("splitAlias(%q) = (%q, %q), want (%q, %q)",
					tt.body, name, cmd, tt.wantName, tt.wantCmd)
			}
		})
	}
}

func TestFilterFunctions(t *testing.T) {
	funcs := []ShellFunction{
		{Name: "gclone", Description: "Clone a git repo"},
		{Name: "mkcd", Description: "Make dir and cd into it"},
	}
	// name match
	if got := FilterFunctions(funcs, "gclone"); len(got) != 1 || got[0].Name != "gclone" {
		t.Errorf("name filter: got %+v", got)
	}
	// description match, case-insensitive
	if got := FilterFunctions(funcs, "GIT"); len(got) != 1 || got[0].Name != "gclone" {
		t.Errorf("desc filter: got %+v", got)
	}
	// empty filter returns all
	if got := FilterFunctions(funcs, ""); len(got) != 2 {
		t.Errorf("empty filter should return all, got %d", len(got))
	}
	// no match
	if got := FilterFunctions(funcs, "zzz"); len(got) != 0 {
		t.Errorf("no-match filter should be empty, got %+v", got)
	}
}

func TestFilterAliases(t *testing.T) {
	aliases := []ShellAlias{
		{Name: "ll", Command: "eza --git -l", Description: "Long listing"},
		{Name: "gs", Command: "git status", Description: "Status"},
	}
	// command match — the reason FilterAliases searches Command, not just name
	if got := FilterAliases(aliases, "eza"); len(got) != 1 || got[0].Name != "ll" {
		t.Errorf("command filter: got %+v", got)
	}
	// name match
	if got := FilterAliases(aliases, "gs"); len(got) != 1 || got[0].Name != "gs" {
		t.Errorf("name filter: got %+v", got)
	}
	// description match, case-insensitive
	if got := FilterAliases(aliases, "LONG"); len(got) != 1 || got[0].Name != "ll" {
		t.Errorf("desc filter: got %+v", got)
	}
}

// TestLoadForgitAliases exercises the plugin-default regex against a fixture,
// honoring $ZSH_PLUGINS_DIR so no real forgit install is required.
func TestLoadForgitAliases(t *testing.T) {
	pluginsDir := t.TempDir()
	forgitDir := filepath.Join(pluginsDir, "forgit")
	if err := os.MkdirAll(forgitDir, 0o755); err != nil {
		t.Fatalf("mkdir forgit: %v", err)
	}
	content := `# forgit defaults
forgit_log="${forgit_log:-glo}"
forgit_checkout_file="${forgit_checkout_file:-gcf}"
unrelated line
`
	if err := os.WriteFile(filepath.Join(forgitDir, "forgit.plugin.zsh"), []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	t.Setenv("ZSH_PLUGINS_DIR", pluginsDir)

	aliases, err := LoadForgitAliases()
	if err != nil {
		t.Fatalf("LoadForgitAliases: %v", err)
	}
	if len(aliases) != 2 {
		t.Fatalf("expected 2 forgit aliases, got %d: %+v", len(aliases), aliases)
	}
	// Name is the shortcut default; Command is the action with underscores spaced.
	if aliases[0].Name != "glo" || aliases[0].Command != "log" {
		t.Errorf("alias 0 = %+v, want glo/log", aliases[0])
	}
	if aliases[1].Name != "gcf" || aliases[1].Command != "checkout file" {
		t.Errorf("alias 1 = %+v, want gcf/'checkout file'", aliases[1])
	}
}
