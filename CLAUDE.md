# Toolbox

Dotfiles tool discovery CLI written in Go.

## Releasing

Releases are automated via GoReleaser + GitHub Actions.

1. `git tag vX.Y.Z && git push origin vX.Y.Z`
2. GitHub Actions builds binaries for linux/darwin x amd64/arm64
3. Binaries appear on the GitHub releases page

The `toolbox update` command pulls the latest release binary via `go-selfupdate`. No Go toolchain needed on target machines.

**Versioning**: `fix:` commits get a patch bump, `feat:` commits get a minor bump. The version is injected at build time via ldflags (`-X main.buildVersion`).

## Registry

The tool registry lives at `~/dev/tools.yml` (override with `$TOOLBOX_REGISTRY`). It is Syncthing-synced data, not config. Commands that don't need the registry (`update`, `funcs`, `aliases`, `--help`) work without it.
