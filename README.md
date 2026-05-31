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
- Incremental copy by file size and modification time, with optional SHA-256 checksum comparison.
- `--delete`, `--dry-run`, `--check`, and repeatable `--exclude` rules.
- Importable Go API through `internal/rsync233` for repo-local extension.

## Install

```powershell
go install github.com/neko233-com/rsync233/cmd/rsync233@latest
```

## Usage

```powershell
rsync233 [options] SOURCE DEST
```

Examples:

```powershell
rsync233 .\public\ .\dist
rsync233 --delete --exclude "*.tmp" .\public\ .\dist
rsync233 --dry-run .\public\ ssh://deploy@example.com/var/www/
rsync233 --check -c .\public\ deploy@example.com:/var/www/
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
- `deploy.cmd` creates all supported release binaries under `dist/`.
- `git-push.cmd` runs `test.cmd`, commits pending changes, and pushes `main`.

## Exit Codes

- `0`: success, or `--check` found no differences.
- `1`: runtime or argument error.
- `2`: `--check` found differences.
