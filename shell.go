package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// shell.go - Parse shell functions and aliases from dotfiles shell source files
//
// Shell files live at $SHELL_DIR (default ~/.local/shell/) as symlinks into dotfiles.
// Three files are sourced on startup:
//   - functions.sh  — cross-platform shell functions
//   - aliases.sh    — cross-platform aliases
//   - $PLATFORM.sh  — platform-specific functions + aliases (macos, arch, wsl)
//
// Function doc convention: adjacent comment pair before the function body
//   #@funcname
//   #--> Human-readable description or usage hint
//
// Alias format: optional single-line comment above the alias line
//   # Description of what it does
//   alias name='command'

// ShellFunction holds a parsed function name and its description.
type ShellFunction struct {
	Name        string
	Description string
	Source      string // which file it came from (basename)
}

// ShellAlias holds a parsed alias name, the command it expands to, and an optional description.
type ShellAlias struct {
	Name        string
	Command     string
	Description string
	Source      string // which file it came from (basename)
}

// getShellDir returns the shell source directory, honoring $SHELL_DIR.
func getShellDir() string {
	if dir := os.Getenv("SHELL_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("cannot determine home directory: %v", err))
	}
	return filepath.Join(home, ".local", "shell")
}

// getPlatform returns the current platform identifier from $PLATFORM.
func getPlatform() string {
	return os.Getenv("PLATFORM")
}

// shellSourceFiles returns the ordered list of shell files to parse.
// functions.sh and aliases.sh are always included; $PLATFORM.sh is added
// when $PLATFORM is set and the file exists.
func shellSourceFiles(shellDir string) (funcFiles, aliasFiles []string) {
	functionsFile := filepath.Join(shellDir, "functions.sh")
	aliasesFile := filepath.Join(shellDir, "aliases.sh")

	if fileExists(functionsFile) {
		funcFiles = append(funcFiles, functionsFile)
	}
	if fileExists(aliasesFile) {
		aliasFiles = append(aliasFiles, aliasesFile)
	}

	platform := getPlatform()
	if platform != "" {
		platformFile := filepath.Join(shellDir, platform+".sh")
		if fileExists(platformFile) {
			// Platform file contains both functions and aliases
			funcFiles = append(funcFiles, platformFile)
			aliasFiles = append(aliasFiles, platformFile)
		}
	}

	return funcFiles, aliasFiles
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ParseFunctions scans a shell file for the #@name / #--> description pattern.
// Both markers must appear on consecutive lines (the name line immediately
// before the description line) — matching how the shell awk one-liner works.
func ParseFunctions(path string) ([]ShellFunction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	source := filepath.Base(path)
	var funcs []ShellFunction
	var pendingName string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#@") {
			pendingName = strings.TrimSpace(line[2:])
			continue
		}

		if strings.HasPrefix(line, "#-->") && pendingName != "" {
			desc := strings.TrimSpace(line[4:])
			funcs = append(funcs, ShellFunction{
				Name:        pendingName,
				Description: desc,
				Source:      source,
			})
			pendingName = ""
			continue
		}

		// Any other line resets the pending name — the markers must be adjacent
		pendingName = ""
	}

	return funcs, scanner.Err()
}

// ParseAliases scans a shell file for alias lines, capturing the preceding
// comment line (if any) as the description.
//
// Recognized formats:
//
//	alias name='command'
//	alias name="command"
//	alias name=command     (no quotes)
func ParseAliases(path string) ([]ShellAlias, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	source := filepath.Base(path)
	var aliases []ShellAlias
	var lastComment string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "# ") {
			lastComment = strings.TrimSpace(line[2:])
			continue
		}

		if strings.HasPrefix(line, "alias ") {
			body := strings.TrimPrefix(line, "alias ")
			name, cmd := splitAlias(body)
			if name != "" {
				aliases = append(aliases, ShellAlias{
					Name:        name,
					Command:     cmd,
					Description: lastComment,
					Source:      source,
				})
			}
			lastComment = ""
			continue
		}

		// Blank line or non-comment/non-alias resets the comment buffer
		if line == "" || !strings.HasPrefix(line, "#") {
			lastComment = ""
		}
	}

	return aliases, scanner.Err()
}

// splitAlias splits "name='cmd'" into ("name", "cmd").
// Handles single-quoted, double-quoted, and unquoted values.
func splitAlias(body string) (name, cmd string) {
	eq := strings.IndexByte(body, '=')
	if eq < 0 {
		return "", ""
	}
	name = strings.TrimSpace(body[:eq])
	raw := body[eq+1:]

	// Strip surrounding quotes
	if len(raw) >= 2 {
		if (raw[0] == '\'' && raw[len(raw)-1] == '\'') ||
			(raw[0] == '"' && raw[len(raw)-1] == '"') {
			raw = raw[1 : len(raw)-1]
		}
	}
	return name, raw
}

// LoadShellFunctions returns all parsed functions from the active shell files.
func LoadShellFunctions() ([]ShellFunction, error) {
	shellDir := getShellDir()
	funcFiles, _ := shellSourceFiles(shellDir)

	var all []ShellFunction
	for _, path := range funcFiles {
		funcs, err := ParseFunctions(path)
		if err != nil {
			return nil, err
		}
		all = append(all, funcs...)
	}
	return all, nil
}

// LoadShellAliases returns all parsed aliases from the active shell files.
func LoadShellAliases() ([]ShellAlias, error) {
	shellDir := getShellDir()
	_, aliasFiles := shellSourceFiles(shellDir)

	var all []ShellAlias
	for _, path := range aliasFiles {
		aliases, err := ParseAliases(path)
		if err != nil {
			return nil, err
		}
		all = append(all, aliases...)
	}
	return all, nil
}

// LoadGitAliases returns git aliases from `git config --get-regexp ^alias\.`.
// Git resolves every config file, so this is source-agnostic. A missing git or
// an absence of aliases is non-fatal — the caller just gets an empty slice.
func LoadGitAliases() ([]ShellAlias, error) {
	out, err := exec.Command("git", "config", "--get-regexp", `^alias\.`).Output()
	if err != nil {
		return nil, err
	}

	var aliases []ShellAlias
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		key, cmd, found := strings.Cut(scanner.Text(), " ")
		if !found {
			continue
		}
		name := strings.TrimPrefix(key, "alias.")
		if name != "" {
			aliases = append(aliases, ShellAlias{Name: name, Command: cmd, Source: "gitconfig"})
		}
	}
	return aliases, scanner.Err()
}

// FilterFunctions returns functions whose name or description contains the filter string (case-insensitive).
func FilterFunctions(funcs []ShellFunction, filter string) []ShellFunction {
	if filter == "" {
		return funcs
	}
	lower := strings.ToLower(filter)
	var out []ShellFunction
	for _, f := range funcs {
		if strings.Contains(strings.ToLower(f.Name), lower) ||
			strings.Contains(strings.ToLower(f.Description), lower) {
			out = append(out, f)
		}
	}
	return out
}

// FilterAliases returns aliases whose name, command, or description contains the filter string (case-insensitive).
func FilterAliases(aliases []ShellAlias, filter string) []ShellAlias {
	if filter == "" {
		return aliases
	}
	lower := strings.ToLower(filter)
	var out []ShellAlias
	for _, a := range aliases {
		if strings.Contains(strings.ToLower(a.Name), lower) ||
			strings.Contains(strings.ToLower(a.Command), lower) ||
			strings.Contains(strings.ToLower(a.Description), lower) {
			out = append(out, a)
		}
	}
	return out
}
