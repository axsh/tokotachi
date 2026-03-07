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
- GitHub CLI `gh` (for `publish` only)

## Scripts

| Script | Description | Usage |
|--------|-------------|-------|
| `_lib.sh` | Common library (sourced by all scripts) | — |
| `build` | Build CLI tools from features | `./scripts/dist/build <tool-id>` |
| `release` | Create release artifacts | `./scripts/dist/release <tool-id> <version>` |
| `publish` | Publish to GitHub Releases | `./scripts/dist/publish <tool-id> <version>` |
| `dev` | Launch development environments | `./scripts/dist/dev <feature-name>` |
| `install-tools` | Install developer tools locally | `./scripts/dist/install-tools [--all \| <tool-id>...]` |
| `bootstrap-tools` | Initial setup for new developers | `./scripts/dist/bootstrap-tools` |

## Release Workflow

### 1. Build

Build a CLI tool from its feature source:

```bash
./scripts/dist/build devctl
```

This reads `tools/manifests/devctl.yaml` to determine build targets,
then compiles the Go binary for all specified platforms.

### 2. Release

Create release artifacts (archives + checksums):

```bash
./scripts/dist/release devctl v1.0.0
```

Artifacts are written to `dist/devctl/v1.0.0/`.

### 3. Publish

Publish the release to distribution channels:

```bash
./scripts/dist/publish devctl v1.0.0
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
scripts/dist/build
     ↓
dist/
     ↓
scripts/dist/release
     ↓
packaging/
     ↓
scripts/dist/publish
     ↓
Homebrew / Scoop / GitHub Releases
```

## Related Files

- `tools/manifests/` — Tool distribution metadata
- `packaging/` — Build packaging configuration (GoReleaser, archives, checksums)
- `releases/` — Release history and channel definitions
- `dist/` — Build artifacts (gitignored)
