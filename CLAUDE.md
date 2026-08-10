# Toolbox

Dotfiles tool discovery CLI written in Go.

## Releasing

Releases are fully automated on push to `main` — do NOT tag manually. The
`Release` workflow (`.github/workflows/release.yml`) runs `go-semantic-release`,
which reads the conventional commits since the last release, decides the next
version, creates the tag and GitHub release, and prepends to `CHANGELOG.md`.
GoReleaser then builds binaries for linux/darwin × amd64/arm64 and attaches them.

Just push conventional commits to `main`; the version and tag follow. A manual
`git tag vX.Y.Z` preempts semantic-release — it sees the commit as already
tagged, emits no version, and GoReleaser is skipped, so no binaries are built.
If that happens, delete the stray tag (local and remote) and re-run the workflow.

The `toolbox update` command pulls the latest release binary via `goselfupdate` — the fleet's own library at `github.com/datapointchris/goselfupdate`, not the third-party `go-selfupdate`, which carries `GO-2026-5932`. No Go toolchain needed on target machines.

**Versioning**: `fix:` commits get a patch bump, `feat:` commits get a minor bump. The version is injected at build time via ldflags (`-X main.buildVersion`).

`.semrelrc` overrides the analyzer's release rules so a breaking marker cannot
cut a major: `feat!:` bumps a minor, and `chore(release-major)` is the only
subject that majors. A major strands every installed binary, because
`goselfupdate` refuses a lower version and reports "already up to date"
forever — recovery is a reinstall on each machine, not an update.

**Never write `BREAKING CHANGE` in a commit message, including the body.** The
analyzer matches that string anywhere in the raw message, before and
regardless of the configured rules, so it majors even a `fix:` commit and no
config can switch it off. A commit that needs to discuss the trailer has to
name it some other way. This is unrecoverable without a rewrite of `main`,
which branch protection refuses: the offending commit keeps re-cutting the
major on every push until a tag above it takes it out of range.

## Registry

The tool registry is authored data, not config — owned by dotfiles and symlinked under the XDG data dir so it is version-controlled and reaches every machine. toolbox reads `$TOOLBOX_REGISTRY` if set, otherwise the XDG data path; reminder state defaults under the XDG state dir (`$TOOLBOX_REMINDERS` overrides). Commands that don't need the registry (`update`, `funcs`, `aliases`, `--help`) work without it.
