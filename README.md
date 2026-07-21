# toolbox

Dotfiles tool discovery system written in Go.

## Features

- **Search**: Case-insensitive search across tool names, descriptions, tags, why-use, and notes
- **Browse**: Interactive category → tool browser using gum
- **Display**: Beautiful colored output with detailed tool information
- **Fast**: Instant search and filtering
- **Tested**: Comprehensive test coverage

## Usage

```bash
# Show help
toolbox

# Search for tools (shortcut)
toolbox git

# Explicit search
toolbox search git

# Show tool details
toolbox show ripgrep

# List all tools by category
toolbox list

# Interactive category browser (requires gum)
toolbox categories

# List shell functions (from dotfiles shell source files)
toolbox funcs
toolbox funcs git

# List shell aliases
toolbox aliases
toolbox aliases docker
```

## Commands

- `toolbox` - Show help
- `toolbox list` - List all tools grouped by category (alphabetically)
- `toolbox show <tool>` - Show detailed information about a tool
- `toolbox search <query>` - Search tools (case-insensitive)
- `toolbox categories` - Interactive category picker with gum
- `toolbox check` - Audit registry against installed tools
- `toolbox remind` - Surface a forgotten tool, function, alias, git alias, or forgit shortcut (neglect-weighted, 90-day recency)
- `toolbox funcs [filter]` - List shell functions parsed from dotfiles shell files
- `toolbox aliases [filter]` - List shell aliases parsed from dotfiles shell files
- `toolbox update` - Update toolbox to the latest release
- `toolbox <query>` - Shortcut for search

## Building and Installing

```bash
# Build the binary
task build

# Build and install to ~/go/bin
task install
```

The built binary (`toolbox`) is a build artifact (gitignored). `go install` also works and puts the binary in `$GOPATH/bin`.

## Testing

```bash
go test -v
```

## Registry

Tools are defined in a registry owned by dotfiles and symlinked under the XDG data dir (override with `$TOOLBOX_REGISTRY`)

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [yaml.v3](https://github.com/go-yaml/yaml) - YAML parser
- [gum](https://github.com/charmbracelet/gum) - Optional, for interactive menus

## Code Structure

- `main.go` - CLI entry point and commands
- `types.go` - Data structures
- `registry.go` - YAML loading
- `search.go` - Search and filter functions
- `display.go` - Output formatting
- `interactive.go` - Gum integration
- `shell.go` - Shell function and alias parsing from `~/.local/shell/`
- `check.go` - Registry audit against installed tools
- `remind.go` - Least-recently-reminded tool surfacing
- `update.go` - Self-update from GitHub releases
- `search_test.go` - Search and matching tests
- `shell_test.go` - Shell function/alias parsing tests
- `remind_test.go` - History parsing and reminder-selection tests

## Comments

This code includes extensive comments for learning Go, including:

- Comparisons with Python
- Explanations of Go idioms
- Gotchas and best practices
