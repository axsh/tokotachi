# Distribution Scripts

CLI tools distribution pipeline scripts.

## Overview

This directory contains scripts for building, releasing, and publishing
CLI tools defined in `features/`. These scripts form the distribution
pipeline that produces cross-platform binaries and publishes them to
package managers.

## Prerequisites

- Go 1.21+
- Python 3 (YAML parsing)
- GitHub CLI `gh` (for `publish.sh` and `github-upload.sh`)

## Scripts

| Script | Description | Usage |
|--------|-------------|-------|
| `_lib.sh` | Common library (sourced by all scripts) | — |
| `build.sh` | Build CLI tools from features | `./scripts/dist/build.sh <tool-id>` |
| `release.sh` | Create release artifacts | `./scripts/dist/release.sh <tool-id> <version>` |
| `publish.sh` | Publish to GitHub Releases | `./scripts/dist/publish.sh <tool-id> <version>` |
| `github-upload.sh` | All-in-one: build + release + publish | `./scripts/dist/github-upload.sh <tool-id> [version]` |
| `dev.sh` | Launch development environments | `./scripts/dist/dev.sh <feature-name>` |
| `install-tools.sh` | Install developer tools locally | `./scripts/dist/install-tools.sh [--all \| <tool-id>...]` |
| `bootstrap-tools.sh` | Initial setup for new developers | `./scripts/dist/bootstrap-tools.sh` |
| `install.ps1` | Install tt to user-local directory (Windows) | `powershell -ExecutionPolicy Bypass -File .\scripts\dist\install.ps1` |
| `uninstall.ps1` | Uninstall tt (Windows) | `powershell -ExecutionPolicy Bypass -File .\scripts\dist\uninstall.ps1` |

## Release Workflow

### Quick Release (All-in-one)

Build, release, and publish in a single command:

```bash
# Patch release (default: +v0.0.1)
./scripts/dist/github-upload.sh tt

# Specific version
./scripts/dist/github-upload.sh tt v2.0.0

# Increment version
./scripts/dist/github-upload.sh tt +v0.1.0
```

### Step-by-Step Release

#### 1. Build

Build a CLI tool from its feature source:

```bash
./scripts/dist/build.sh tt
```

This reads `tools/manifests/tt.yaml` to determine build targets,
then compiles the Go binary for all specified platforms.

#### 2. Release

Create release artifacts (archives + checksums):

```bash
./scripts/dist/release.sh tt v1.0.0
```

Artifacts are written to `dist/tt/v1.0.0/`.

#### 3. Publish

Publish the release to distribution channels:

```bash
./scripts/dist/publish.sh tt v1.0.0
```

This publishes to:
- GitHub Releases
- Homebrew tap (from `tools/installers/homebrew/`)
- Scoop bucket (from `tools/installers/scoop/`)

## Artifact Flow

```
features/
     ↓
tools/manifests/
     ↓
scripts/dist/build.sh
     ↓
dist/
     ↓
scripts/dist/release.sh
     ↓
packaging/
     ↓
scripts/dist/publish.sh
     ↓
Homebrew / Scoop / GitHub Releases

── Or all-in-one ──
scripts/dist/github-upload.sh → build.sh → release.sh → publish.sh
```

## Related Files

- `tools/manifests/` — Tool distribution metadata
- `packaging/` — Build packaging configuration (GoReleaser, archives, checksums)
- `releases/` — Release history and channel definitions
- `dist/` — Build artifacts (gitignored)
