package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/datapointchris/goselfupdate/autoupdate"
)

// runLine drives the real tree through run, the path main takes, with the
// version check suppressed. goclikit.Execute resolves the command from
// os.Args, so the line goes there rather than through SetArgs. What cobra
// prints itself lands in the same buffer as what run prints.
func runLine(t *testing.T, args ...string) (int, string) {
	t.Helper()
	original := os.Args
	os.Args = append([]string{"toolbox"}, args...)
	// Execute resolves the command from the root's parsed words where it has
	// any, and a line an earlier test ran on the root leaves its words there.
	if err := rootCmd.Flags().Parse(nil); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	t.Cleanup(func() {
		os.Args = original
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	code := run(&output, autoupdate.Config{Suppress: true})
	return code, output.String()
}

// `update` silences cobra and prints its own result, so a mistyped flag on it
// reached the terminal only if run printed it.
func TestAFlagMistakeOnUpdateIsPrintedOnce(t *testing.T) {
	code, stderr := runLine(t, "update", "--check", "--bogus")

	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if got := strings.Count(stderr, "unknown flag: --bogus"); got != 1 {
		t.Errorf("printed the error %d times, want once:\n%s", got, stderr)
	}
}

// The error carries its own pointer to --help, so the usage screen cobra would
// print after it is the same help said twice.
func TestAnUnknownCommandIsAnsweredWithoutTheUsageScreen(t *testing.T) {
	code, stderr := runLine(t, "bogus", "extra")

	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if got := strings.Count(stderr, `unknown command "bogus" for "toolbox"`); got != 1 {
		t.Errorf("refused the word %d times, want once:\n%s", got, stderr)
	}
	if strings.Contains(stderr, "Available Commands:") {
		t.Errorf("the usage screen followed the error:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Run 'toolbox --help' for usage.") {
		t.Errorf("the error does not point at --help:\n%s", stderr)
	}
}

// Cobra runs a command that declares no Args with whatever words follow it, so
// `toolbox list git` would print every tool as though it had filtered them.
func TestAWordOnACommandTakingNoneIsRefused(t *testing.T) {
	for _, command := range []string{"list", "categories", "remind"} {
		t.Run(command, func(t *testing.T) {
			code, stderr := runLine(t, command, "bogus")

			if code != 2 {
				t.Errorf("exit = %d, want 2", code)
			}
			if want := `unknown command "bogus" for "toolbox ` + command + `"`; !strings.Contains(stderr, want) {
				t.Errorf("the word was not refused:\n%s", stderr)
			}
		})
	}
}

// Cobra returns a wrong argument count as a plain error, which exits 1 as
// though the command had run and failed. A bare command is answered with its
// help and one given too many words with the refusal, so each has a line.
func TestAWrongArgumentCountExitsTwo(t *testing.T) {
	for _, line := range [][]string{{"show"}, {"funcs", "one", "two"}} {
		t.Run(strings.Join(line, " "), func(t *testing.T) {
			code, stderr := runLine(t, line...)

			if code != 2 {
				t.Errorf("exit = %d, want 2:\n%s", code, stderr)
			}
		})
	}
}

// Two words make the first a command, and a slip from one is answered with it.
func TestAMistypedCommandNamesTheNearOne(t *testing.T) {
	code, stderr := runLine(t, "lisy", "extra")

	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "\tlist") {
		t.Errorf("the near command was not named:\n%s", stderr)
	}
}
