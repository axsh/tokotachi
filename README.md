# Tokotachi - 常立

"The Immutable Foundation for AI Orchestration."

A monorepo for **AI-assisted development workflow automation**, built around a modular feature architecture with template-based scaffolding, containerized development environments, and agent-driven workflows.

## Origin

Named after Kuni-no-Tokotachi-no-Kami, the first deity in Japanese mythology to appear and solidify the chaotic, drifting earth into a firm "foundation."

Just as the deity established the ground for all existence, Tokotachi eliminates the chaos of modern AI development environments. It provides a rock-solid, parallel execution platform where multiple AI agents and processes can thrive with absolute stability.

## Overview

**Tokotachi** provides a structured foundation for managing multiple development features in a single repository. Each feature is an isolated, self-contained component with its own source code, tests, dependencies, and containerized development environment.

The project is designed to work seamlessly with AI coding agents, enabling a specification → implementation plan → execution pipeline where both humans and AI collaborate through well-defined workflows.

### Key Highlights

- **Feature-based monorepo** — isolated modules under `features/`, each independently buildable and testable
- **`devctl` CLI** — a matrix-driven development environment orchestrator written in Go
- **Template catalog** — version-pinned feature templates for reproducible scaffolding
- **Agent workflows** — structured AI-assisted specification, planning, and implementation processes
- **Multi-platform support** — Linux, macOS, and Windows with multiple editor/container combinations

## Repository Structure

```
tokotachi/
├── features/              # All feature implementations
│   ├── devctl/            # Development environment orchestrator (Go)
│   └── integration-test/  # Integration test suite (Python)
├── catalog/               # Template catalog configuration
├── environments/          # Shared environment configs (Docker Compose)
├── shared/                # Shared resources (libs, schemas, testdata)
├── tests/                 # Project-level test suites
│   └── integration-test/  # Integration tests (Go)
├── scripts/               # Build and test automation
│   ├── dist/              # Distribution pipeline (build, release, publish)
│   ├── process/           # build.sh, integration_test.sh
│   └── utils/             # Utility scripts
├── prompts/               # AI workflow specifications and rules
│   ├── phases/            # Feature specs and implementation plans
│   └── rules/             # Coding, testing, and planning rules
├── .agent/                # AI agent configuration
│   ├── workflows/         # Workflow definitions
│   └── rules/             # Agent-specific rules
├── bin/                   # Build output (gitignored)
└── work/                  # Git worktrees (gitignored)
```

## Features

### devctl — Development Environment Orchestrator

The core feature of this repository. `devctl` is a CLI tool that manages feature-level development environments across different **OS × Editor × Container** combinations.

```bash
devctl up <feature> [branch]              # Start a development container
devctl up <feature> [branch] --editor cursor  # Start + open editor
devctl down <feature> [branch]            # Stop and remove the container
devctl open <feature> [branch] --editor code  # Open editor for a feature
devctl status <feature> [branch]          # Show feature status
devctl shell <feature> [branch]           # Open a shell in the container
devctl exec <feature> [branch] -- go test ./...  # Execute a command
devctl close <feature> [branch]           # Full teardown (container + worktree + branch)
devctl list <feature>                     # List branches for a feature
devctl pr <feature> [branch]              # Create a GitHub Pull Request
```

#### Supported Environments

| OS | Editors | Container Modes |
|---|---|---|
| Linux, macOS, Windows | VSCode, Cursor, Antigravity, Claude Code | `none`, `docker-local`, `docker-ssh`, `devcontainer` |

#### Architecture

```
detect → resolve → plan → execute → report
```

The processing pipeline detects the environment, resolves configuration, builds an execution plan, runs actions, and generates a report.

See [`features/devctl/README.md`](features/devctl/README.md) for full documentation.

## Installation

### Pre-built Binaries (Recommended)

Download the latest release from [GitHub Releases](https://github.com/axsh/tokotachi/releases).

#### Linux / macOS

```bash
# Download (replace OS and ARCH as needed: linux/darwin, amd64/arm64)
curl -LO https://github.com/axsh/tokotachi/releases/latest/download/devctl_linux_amd64.tar.gz

# Extract
tar xzf devctl_linux_amd64.tar.gz

# Move to PATH
sudo mv devctl /usr/local/bin/
```

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/axsh/tokotachi/releases/latest/download/devctl_darwin_arm64.tar.gz
tar xzf devctl_darwin_arm64.tar.gz
sudo mv devctl /usr/local/bin/
```

#### Windows

1. Download `devctl_windows_amd64.zip` from [Releases](https://github.com/axsh/tokotachi/releases)
2. Extract the zip file
3. Move `devctl.exe` to a directory in your `PATH`

#### Verify Installation

```bash
devctl --help
```

### Build from Source

Requires **Go 1.24+**, **Docker**, **Git**, and **Bash** (Git Bash on Windows).

```bash
# Clone the repository
git clone https://github.com/axsh/tokotachi.git
cd tokotachi

# Bootstrap: build and install all tools
./scripts/dist/bootstrap-tools

# Or build individually
./scripts/dist/build devctl
./scripts/dist/install-tools devctl
```

The compiled binary is output to `bin/devctl`.

### Run Tests

```bash
# Full build + unit tests
./scripts/process/build.sh

# Integration tests
./scripts/process/integration_test.sh
```

## Development Workflow

This project uses an **AI-assisted development workflow** with structured phases:

```
  Idea → Specification → Implementation Plan → Execution → Verification
```

### Workflow Phases

1. **Specification** — Capture requirements in `prompts/phases/` using [`create-specification`](.agent/workflows/create-specification.md)
2. **Implementation Plan** — Generate detailed plans using [`create-implementation-plan`](.agent/workflows/create-implementation-plan.md)
3. **Execution** — Implement code and tests using [`execute-implementation-plan`](.agent/workflows/execute-implementation-plan.md)
4. **Verification** — Build and test using [`build-pipeline`](.agent/workflows/build-pipeline.md)

Each phase includes a **human review checkpoint** before progressing to the next.

### Configuration

#### Project-level (`.devrc.yaml`)

```yaml
project_name: myproject
default_editor: cursor
default_container_mode: docker-local
```

#### Feature-level (`feature.yaml`)

```yaml
dev:
  editor_default: code
  container_mode: devcontainer
```

## Contributing

### Creating a New Feature

New features are generated from templates defined in the `catalog/` directory and placed under `features/`:

```bash
featurectl new my-feature --template go-service
```

### Collaboration Model

Multiple developers or AI agents can work on different features simultaneously using:

- **Git worktrees** — isolated working directories per feature/branch
- **Dev Containers** — consistent, reproducible development environments
- **Independent environments** — per-feature isolation

### Rules and Standards

- [Coding Rules](prompts/rules/coding-rules.md)
- [Testing Rules](prompts/rules/testing-rules.md)
- [Planning Rules](prompts/rules/planning-rules.md)
- [Vibe Coding Standard](prompts/rules/Vibe-Coding-Standard.md)

## Tech Stack

| Component | Technology |
|---|---|
| Core CLI | Go 1.24, Cobra |
| Testing | Go `testing`, `testify`; Python (integration) |
| Configuration | YAML (`gopkg.in/yaml.v3`) |
| Containers | Docker, Dev Containers |
| Infrastructure | Docker Compose |
| CI Scripts | Bash |

## License

This project is private.
