# devctl — Development Environment Orchestrator

A matrix-driven CLI tool that manages feature-level development environments across different OS, editor, and container combinations.

## Overview

`devctl` standardizes the workflow for starting, connecting to, and tearing down development environments. It abstracts away differences between operating systems, editors, and container modes through a unified subcommand interface.

## Usage

```bash
# Start a container for a feature
devctl up <feature> [branch]

# Start container + open editor
devctl up <feature> [branch] --editor cursor

# Open editor (local worktree)
devctl open <feature> [branch] --editor code

# Reconnect editor to running container (DevContainer attach)
devctl open <feature> [branch] --editor code --attach

# Stop and remove the container
devctl down <feature> [branch]

# Show feature status
devctl status <feature> [branch]

# Open a shell in the container
devctl shell <feature> [branch]

# Execute a command in the container
devctl exec <feature> [branch] -- go test ./...

# Dry-run with verbose logging and report
devctl up <feature> [branch] --editor cursor --dry-run --verbose --report report.md
```

**Note**: `[branch]` is optional. When omitted, the feature name is used as the branch name.

### Subcommands

| Subcommand | Description |
|---|---|
| `up` | Start the development container (+worktree auto-creation) |
| `down` | Stop and remove the container |
| `open` | Open the editor |
| `status` | Show feature status |
| `shell` | Open a shell in the container |
| `exec` | Execute a command in the container |
| `pr` | Create a GitHub Pull Request |
| `close` | Close (down + worktree remove + branch delete) |
| `list` | List branches for a feature |

### Subcommand Flags

| Flag | Applicable To | Description |
|---|---|---|
| `--editor <name>` | `up`, `open` | Editor to use: `code`, `cursor`, `ag`, `claude` |
| `--ssh` | `up` | Enable SSH mode |
| `--rebuild` | `up` | Rebuild the container image |
| `--no-build` | `up` | Skip image build |
| `--attach` | `open` | Attempt DevContainer attach to running container |
| `--force` | `close` | Force delete even if branch is not merged |

### Global Flags (All Subcommands)

| Flag | Description |
|---|---|
| `--verbose` | Show debug-level logs |
| `--dry-run` | Show planned actions without executing |
| `--report <file>` | Write execution report as Markdown file |

### Command Logging

All external commands (docker, git, gh, editors) are logged before execution:
- Normal: `[CMD] docker run -d --name myproj-devctl ...`
- Dry-run: `[DRY-RUN] docker run -d --name myproj-devctl ...`

### Execution Report

After each run, devctl outputs an execution summary including:
- Date, Feature, Branch
- Environment variables (set values / defaults)
- Detected environment (OS, Editor, ContainerMode)
- Steps executed with results
- Overall result (SUCCESS / FAILED)

Use `--report <file>` to save as Markdown.

## Environment Variables

### Editor Resolution

| Variable | Description | Default |
|---|---|---|
| `DEVCTL_EDITOR` | Override editor selection (same values as `--editor`) | — |

### Command Overrides

All external commands can be overridden via environment variables. This is useful when the command is installed in a non-standard path or has a different name.

| Variable | Description | Default |
|---|---|---|
| `DEVCTL_CMD_CODE` | Path to VSCode CLI | `code` |
| `DEVCTL_CMD_CURSOR` | Path to Cursor CLI | `cursor` |
| `DEVCTL_CMD_AG` | Path to Antigravity CLI | `antigravity` |
| `DEVCTL_CMD_CLAUDE` | Path to Claude Code CLI | `claude` |
| `DEVCTL_CMD_GIT` | Path to Git CLI | `git` |
| `DEVCTL_CMD_GH` | Path to GitHub CLI | `gh` |

### Editor Resolution Priority

1. `--editor` flag (CLI)
2. `DEVCTL_EDITOR` environment variable
3. `feature.yaml` (`editor_default` field)
4. `.devrc.yaml` (`default_editor` field)
5. Default: `cursor`

## Supported Environments

### Operating Systems

| OS | Status |
|---|---|
| Linux | ✅ Supported |
| macOS | ✅ Supported |
| Windows | ✅ Supported |

### Editors / Agents

| Editor | CLI Command | Description |
|---|---|---|
| VSCode | `code` | Dev Container attach supported |
| Cursor | `cursor` | Dev Container attach supported |
| Antigravity | `antigravity` | Local worktree only |
| Claude Code | `claude` | CLI/agent, launched with worktree as cwd |

## Feature Support Matrix

### Container Modes

| Mode | Description |
|---|---|
| `none` | No container; editor opens local worktree |
| `docker-local` | Docker container with bind-mounted worktree |
| `docker-ssh` | Docker container with SSH access |
| `devcontainer` | Dev Container integration (VSCode/Cursor only) |

### OS × Editor Compatibility

| OS | Editor | Dev Container | SSH | Local Open | Launch New Window |
|---|---|---|---|---|---|
| **Linux** | VSCode | ✅ L1 | ✅ | ✅ | ✅ |
| | Cursor | ✅ L1 | ✅ | ✅ | ✅ |
| | Antigravity | ❌ L4 | — | ✅ | — |
| | Claude Code | ❌ L4 | — | ✅ | — |
| **macOS** | VSCode | ✅ L2 | ✅ | ✅ | ✅ |
| | Cursor | ✅ L2 | ✅ | ✅ | ✅ |
| | Antigravity | ❌ L4 | — | ✅ | — |
| | Claude Code | ❌ L4 | — | ✅ | — |
| **Windows** | VSCode | ✅ L2 | ✅ | ✅ | ✅ |
| | Cursor | ✅ L2 | ✅ | ✅ | ✅ |
| | Antigravity | ❌ L4 | — | ✅ | — |
| | Claude Code | ❌ L4 | — | ✅ | — |

### Compatibility Levels

| Level | Name | Behavior |
|---|---|---|
| L1 | Full Support | Normal execution |
| L2 | Best Effort | Attempts operation, falls back on failure |
| L3 | Fallback Only | Skips to fallback directly |
| L4 | Unsupported | Warns or no-op |

## Configuration Files

### Global Config (`.devrc.yaml`)

```yaml
project_name: myproject
default_editor: cursor
default_container_mode: docker-local
```

### Feature Config (`feature.yaml`)

Located in `work/<feature>/feature.yaml` or `features/<feature>/feature.yaml`:

```yaml
dev:
  editor_default: code
  container_mode: devcontainer
```

## Build

```bash
# Build binary to bin/devctl
./scripts/process/build.sh

# Or build directly
cd features/devctl && go build -o ../../bin/devctl .
```

## Architecture

```
features/devctl/
├── main.go                          # Entrypoint
├── cmd/
│   ├── root.go                      # Cobra root + global flags
│   ├── common.go                    # AppContext, shared init logic
│   ├── up.go                        # up subcommand
│   ├── down.go                      # down subcommand
│   ├── open.go                      # open subcommand (--attach)
│   ├── status.go                    # status subcommand
│   ├── shell.go                     # shell subcommand
│   └── exec_cmd.go                  # exec subcommand
├── internal/
│   ├── cmdexec/                     # Unified external command execution
│   ├── report/                      # Execution report generation
│   ├── log/                         # Leveled logger
│   ├── detect/                      # OS and editor detection
│   ├── matrix/                      # OS × Editor capability matrix
│   ├── resolve/                     # Worktree, container, config, devcontainer
│   ├── editor/                      # Editor launch (VSCode, Cursor, AG, Claude)
│   ├── plan/                        # Execution plan builder
│   └── action/                      # Container actions (up, down, status, shell, exec, open)
├── Dockerfile                       # Dev container for devctl itself
└── .devcontainer/
    └── devcontainer.json
```

### Processing Pipeline

```
detect → resolve → plan → execute → report
```
