# Distribution Scripts

CLI tools distribution pipeline scripts.

## Overview

This directory contains scripts for building, releasing, and publishing CLI tools defined in `features/`. These scripts form the distribution pipeline that produces cross-platform binaries and publishes them to package managers.

The scripts are divided into tool-specific release scripts, future content-specific release scripts, and shared utilities.

## Directory Structure

```
scripts/dist/
├── README.md                  # This file
├── tool/                      # Tool (tt) release pipeline scripts
│   ├── release.sh             # [Public] All-in-one: build + package + publish
│   └── internal/              # Internal step scripts (called by release.sh)
│       ├── build.sh           # Compiles Go binaries for target platforms
│       ├── package.sh         # Archives binaries and packages installers
│       ├── publish.sh         # Publishes to GitHub Releases and package managers
│       └── win/               # Windows release assets
│           ├── install.ps1    # Installer for Windows users
│           ├── uninstall.ps1  # Uninstaller for Windows users
│           └── README.md      # Japanese README for the Windows release package
│
├── content/                   # Content release pipeline scripts
│   └── release.sh             # [Public] All-in-one: build + regenerate catalog + git push
│
└── shared/                    # Shared utilities
    └── _lib.sh                # Common bash functions and environment settings
```

## Release Workflow (Tool)

### Quick Release (All-in-one)

Build, package, and publish `tt` in a single command:

```bash
# Patch release (default: increment patch version +v0.0.1)
./scripts/dist/tool/release.sh tt

# Specific version
./scripts/dist/tool/release.sh tt v2.0.0

# Increment minor version
./scripts/dist/tool/release.sh tt +v0.1.0
```

### Step-by-Step Release (Internal)

#### 1. Build
Build a CLI tool from feature source:
```bash
./scripts/dist/tool/internal/build.sh tt
```

#### 2. Package
Create release archives (tar.gz/zip) and generate checksums. For Windows targets, this automatically packages the Windows installer (`install.ps1`, `uninstall.ps1`):
```bash
./scripts/dist/tool/internal/package.sh tt v1.0.0
```
Artifacts are written to `dist/tt/v1.0.0/`.

#### 3. Publish
Publish the release to distribution channels:
```bash
./scripts/dist/tool/internal/publish.sh tt v1.0.0
```
This publishes to:
- GitHub Releases
- Homebrew tap (from `tools/installers/homebrew/`)
- Scoop bucket (from `tools/installers/scoop/`)

---

## Release Workflow (Content)

### Catalog Release

Build, regenerate catalog data using `templatizer`, and commit/push updates to main branch in a single command:

```bash
./scripts/dist/content/release.sh
```

---

## Developer Environment Setup

For developer setup and dev-environment launching, use the scripts located in **`scripts/dev/`**:
- `./scripts/dev/bootstrap.sh` — Initial setup for new developers
- `./scripts/dev/install-tools.sh` — Build and install development tools locally to `bin/`
- `./scripts/dev/dev.sh` — Launch development environment wrapper (`tt up`)
