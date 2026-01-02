# toolbox

Dotfiles tool discovery system written in Go.

## Features

- **Search**: Case-insensitive search across tool names, descriptions, tags, and usage
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
```

## Commands

- `toolbox` - Show help
- `toolbox list` - List all tools grouped by category (alphabetically)
- `toolbox show <tool>` - Show detailed information about a tool
- `toolbox search <query>` - Search tools (case-insensitive)
- `toolbox categories` - Interactive category picker with gum
- `toolbox <query>` - Shortcut for search

## Building and Installing

Toolbox uses Task for building and installing (same pattern as sess):

```bash
# Build the binary (creates apps/common/toolbox/toolbox)
cd apps/common/toolbox
task build

# Build and install to ~/go/bin
task install
```

**Important**: The built binary `apps/common/toolbox/toolbox` is a build artifact (gitignored). The actual installation copies it to `~/go/bin/toolbox` (standard Go location).

### Build vs Install

- **Build**: Creates `apps/common/toolbox/toolbox` (local, gitignored)
- **Install**: Copies to `~/go/bin/toolbox` (standard Go binary location)

This follows dotfiles best practice:

- Source code lives in dotfiles repo
- Build artifacts are gitignored
- Installation happens outside the repo
- No symlinks for binaries (separation of concerns)

## Testing

```bash
go test -v
```

## Registry

Tools are defined in `~/.config/toolbox/registry.yml` (or `$DOTFILES_REGISTRY`)

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
- `search_test.go` - Tests

## Comments

This code includes extensive comments for learning Go, including:

- Comparisons with Python
- Explanations of Go idioms
- Gotchas and best practices
