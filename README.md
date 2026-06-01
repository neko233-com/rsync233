# rsync233

`rsync233` is a Go implementation of practical cross-platform rsync workflows.
It is built with Go 1.26 and runs on Windows, Linux, and macOS.

Supported release targets. In Go target names, `amd64` is the x86_64 build:

- `windows/amd64`
- `windows/arm64`
- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

## Features

- Local-to-local sync on every Go-supported desktop/server platform.
- SSH/SFTP endpoints for cross-machine sync without shelling out to rsync.
- rsync-style source trailing slash behavior.
- Archive/link mode preserves symbolic links as links.
- `-p`/`--perms` preserves permission bits; `--no-perms` disables permission preservation in archive mode.
- `--chmod` applies octal permission adjustments such as `F=644,D=755`.
- `-t`/`--times` preserves modification times; `--no-times` disables time preservation in archive mode.
- Incremental copy by file size and modification time, with optional SHA-256 checksum comparison.
- `--delete`, `--delete-before`, `--delete-after`, `--delete-during`,
  `--delete-excluded`, `--dry-run`, `--check`, `--include`,
  `--exclude`, `--include-from`, `--exclude-from`, and basic `-f` filter rules.
- Common rsync transfer controls: `--ignore-times`, `--size-only`,
  `--ignore-existing`, `--existing`, `--update`, `--ignore-missing-args`,
  `--min-size`, and `--max-size`.
- Backup controls: `--backup`, `--backup-dir`, and `--suffix` preserve overwritten
  or deleted destination files.
- Importable Go API through `internal/rsync233` for repo-local extension.

## Install

### One-Click Install

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.sh | sh -s -- v0.1.0
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.ps1 | iex
```

Windows CMD:

```cmd
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.ps1 | iex"
```

Install a specific version on Windows:

```powershell
irm https://raw.githubusercontent.com/neko233-com/rsync233/main/scripts/install.ps1 -OutFile $env:TEMP\rsync233-install.ps1
& $env:TEMP\rsync233-install.ps1 v0.1.0
```

### From Source

```powershell
go install github.com/neko233-com/rsync233/cmd/rsync233@latest
```

### Self Update

```powershell
rsync233 version
rsync233 update --check
rsync233 update -y
rsync233 update --version v0.1.0
```

## Usage

```powershell
rsync233 [options] SOURCE DEST
```

Examples:

```powershell
rsync233 -a .\public\ .\dist
rsync233 -a --delete --exclude "*.tmp" .\public\ .\dist
rsync233 -a --delete-before .\public\ .\dist
rsync233 -a --delete --delete-excluded --exclude "cache/" .\public\ .\dist
rsync233 -a --backup --suffix ".bak" .\public\ .\dist
rsync233 -a --delete --backup-dir .\rsync-backups .\public\ .\dist
rsync233 -a --chmod "F=644,D=755" .\public\ .\dist
rsync233 -r -p .\scripts\ .\bin
rsync233 -r -t .\assets\ .\published-assets
rsync233 -a --include "*.html" --exclude "*.tmp" .\public\ .\dist
rsync233 -a --exclude-from .\rsync-excludes.txt .\public\ .\dist
rsync233 -a -f "+ keep.txt" -f "- *.txt" .\public\ .\dist
rsync233 -a --min-size 1K --max-size 100M .\media\ .\backup-media
rsync233 -a --update --size-only .\public\ .\dist
rsync233 --ignore-missing-args .\maybe-missing\ .\dist
rsync233 -a --dry-run .\public\ ssh://deploy@example.com/var/www/
rsync233 -a --check -c .\public\ deploy@example.com:/var/www/
```

Source trailing slash matches rsync behavior:

- `rsync233 src/ dst` copies the contents of `src` into `dst`.
- `rsync233 src dst` copies `src` itself into `dst/src`.

Remote endpoints require SSH public-key authentication and a valid
`~/.ssh/known_hosts` entry. Supported remote forms are:

- `ssh://user@example.com:22/path`
- `sftp://user@example.com/path`
- `user@example.com:/path`

## Scripts

- `run.cmd` builds and runs the CLI with forwarded arguments.
- `test.cmd` runs formatting, dependency tidy checks, tests, a local build, and the full platform matrix build check.
- `check-platform-matrix.cmd` verifies Windows/Linux/macOS x amd64/arm64 builds.
- `verify-actions.cmd` validates the GitHub Actions workflows with Node.js 24 LTS tooling.
- `deploy.cmd` creates all supported release binaries under `dist/`, using release asset names compatible with the installers.
- `scripts/deploy.ps1 [version]` updates `version.txt`, builds artifacts, commits release metadata, tags `v*`, and pushes the tag to trigger GitHub Release.
- `git-push.cmd` runs `test.cmd`, commits pending changes, and pushes `main`.
- `scripts/install.sh` and `scripts/install.ps1` install release binaries on macOS/Linux and Windows.

## GitHub Actions

The repository follows the same CI/release shape as `neko233-com/unicli`:

- `.github/workflows/ci.yml` runs on pushes and pull requests to `main`.
- `.github/workflows/release.yml` runs on `v*` tags, builds release binaries, uploads artifacts, generates checksums, and creates a GitHub Release.
- Both workflows set up Node.js 24 LTS and Go 1.26.
- Both workflows verify formatting, `go mod tidy`, `go vet`, race-enabled tests, local build, workflow structure, and the six-platform build matrix.
- Release builds inject the tag into `rsync233 version`; `rsync233 update` downloads matching release assets from GitHub.

## Exit Codes

- `0`: success, or `--check` found no differences.
- `1`: runtime or argument error.
- `2`: `--check` found differences.
