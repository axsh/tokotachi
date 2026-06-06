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
- **`tt` CLI** — a development environment orchestrator written in Go
- **Template catalog** — version-pinned feature templates for reproducible scaffolding
- **Agent workflows** — structured AI-assisted specification, planning, and implementation processes
- **Multi-platform support** — Linux, macOS, and Windows with multiple editor/container combinations

## Repository Structure

`tt` is built with `tt` itself.

```
tokotachi/
├── features/              # All feature implementations
│   ├── tt/                # Development environment orchestrator (Go)
│   └── integration-test/  # Integration test suite (Python)
├── shared/                # Shared resources (libs, schemas, testdata)
├── tests/                 # Project-level test suites
│   └── tt/                # tt integration tests (Go)
├── scripts/               # Build and test automation
│   ├── dist/              # Distribution pipeline (tool/content release)
│   ├── dev/               # Developer environment setup utilities
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

### tt — Development Environment Orchestrator

The core feature of this repository. `tt` is a CLI tool that manages feature-level development environments across different **OS × Editor × Container** combinations.

#### Quick Start

```bash
# 1. Generate a new feature from a template
tt scaffold feature axsh-go-standard
# ... (answer prompts) ...

# 2. Start working — creates worktree, starts container, opens editor
tt open bug-fix-branch myprog

# 3. Done — stops containers and deletes worktree
tt close bug-fix-branch myprog
```

**`tt open`** is a syntax sugar that runs `create → up → editor` in sequence. If the container is already running, the `up` step is automatically skipped.

**`tt close`** is a syntax sugar that runs `down → delete` in sequence with a safety confirmation prompt for uncommitted changes.

#### Editor Selection

The editor is resolved in the following priority order:

1. `--editor` flag (e.g. `tt open mybranch tt --editor code`)
2. `TT_EDITOR` environment variable
3. Default configured value in settings (falls back to **cursor** if unset)

You can dynamically customize editor commands and arguments via the [editor.yaml](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/pkg/editor/config.go#L118) configuration file, supporting custom environments such as `code` (VSCode), `cursor`, `ag` (Antigravity IDE), and `claude` (Claude Code).

#### Primitive Commands

These are the building-block commands that perform a single operation each:

```bash
# Worktree management
tt create <branch>                           # Create a branch and worktree
tt delete <branch>                           # Delete worktree and branch
tt delete <branch> --force                   # Force delete even if branch not merged
tt delete <branch> --depth 5 --yes           # Recursive nested worktree deletion

# Container management
tt up <branch> <feature>                     # Start the development container
tt up <branch> <feature> --ssh               # Start with SSH mode
tt up <branch> <feature> --rebuild           # Rebuild the container image
tt down <branch> <feature>                   # Stop and remove the container

# Editor management
tt editor <branch> [feature]                 # Open the editor for a branch
tt editor <branch> [feature] --editor cursor # Specify editor (code|cursor|ag|claude)
tt editor <branch> [feature] --attach        # DevContainer attach to running container
```

#### Syntax Sugar Commands

These combine multiple primitive commands into a single operation:

```bash
# open = create → up → editor (all-in-one start)
tt open <branch> [feature]                   # Create worktree, start container, and open editor
tt open <branch> [feature] --editor code     # Specify editor to use

# close = down → delete (all-in-one teardown)
tt close <branch> [feature]                  # Stop containers and delete worktree
tt close <branch> [feature] --force          # Force close even if branch not merged
tt close <branch> [feature] --depth 5 --yes  # Recursive close with auto-confirm
```

#### Utility Commands

```bash
# Container interaction
tt status <branch> [feature]                 # Show worktree and container status
tt shell <branch> <feature>                  # Open a shell in the container
tt exec <branch> <feature> -- go test ./...  # Execute a command in the container

# Project management
tt list                                      # List all worktree branches
tt list [branch]                             # List features for a specific branch
tt list --json --path --update --full        # Output options
tt pr <branch> [feature]                     # Create a GitHub Pull Request
tt doctor                                    # Check repository health and config
tt doctor --fix --json                       # Auto-fix with JSON output
tt scaffold [category] [name]                # Generate project structure from templates
tt scaffold --list                           # List available templates
tt scaffold --rollback                       # Rollback last scaffold operation
```

#### Global Flags

```bash
--verbose      # Show debug logs
--dry-run      # Show planned actions without executing
--report FILE  # Write execution report to Markdown file
--env          # Show environment variables in report
```

## Installation

### Pre-built Binaries (Recommended)

Download the latest release from [GitHub Releases](https://github.com/axsh/tokotachi/releases).

#### Linux / macOS

```bash
# Download (replace OS and ARCH as needed: linux/darwin, amd64/arm64)
curl -LO https://github.com/axsh/tokotachi/releases/latest/download/tt_linux_amd64.tar.gz

# Extract
tar xzf tt_linux_amd64.tar.gz

# Move to PATH
sudo mv tt /usr/local/bin/
```

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/axsh/tokotachi/releases/latest/download/tt_darwin_arm64.tar.gz
tar xzf tt_darwin_arm64.tar.gz
sudo mv tt /usr/local/bin/
```

#### Windows

**Option A: Installer Script (Recommended)**

Installs to `%LOCALAPPDATA%\Axsh\Tokotachi\bin` and configures user PATH. No admin privileges required.

```powershell
# Clone and install
git clone https://github.com/axsh/tokotachi.git
cd tokotachi
powershell -ExecutionPolicy Bypass -File .\scripts\dist\tool\internal\win\install.ps1
```

Open a new terminal and verify with `tt --help`.

To uninstall:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dist\tool\internal\win\uninstall.ps1
```

**Option B: Manual Install**

1. Download `tt_windows_amd64.zip` from [Releases](https://github.com/axsh/tokotachi/releases)
2. Extract the zip file
3. Move `tt.exe` to a directory in your `PATH`

---

## Build and Developer Tools

To build from source, you need **Go 1.24+**, **Docker**, **Git**, and **Bash** (Git Bash on Windows).

### Bootstrapping Development Environment
To build and install all CLI developer tools defined under the `features/` directory:

```bash
./scripts/dist/bootstrap-tools.sh
```

### Building and Installing Tools Individually
You can build and install a specific tool (such as `tt`) using individual pipeline scripts:

```bash
# Build the binary (outputs to bin/tt)
./scripts/dist/build.sh tt

# Install the built tool to your user PATH
./scripts/dist/install-tools.sh tt
```

---

## Running Tests

Test automation scripts are located in the [scripts/process/](file:///c:/Users/yamya/myprog/tokotachi/work/fix-antigravity-ide/scripts/process) directory.

### 1. Build & Unit Tests
To compile all modules and run Go package unit tests (excluding the `tests/` directory):

```bash
./scripts/process/build.sh
```

- If this script fails (Exit Code != 0), do not proceed to integration tests. Fix all compilation and unit test errors first.

### 2. Integration & E2E Tests
To run integration tests in the `tests/` directory (supporting Go test suites and Python test setups):

```bash
# Run all integration tests
./scripts/process/integration_test.sh

# Run tests under a specific category (e.g., tests/tt/)
./scripts/process/integration_test.sh --categories "tt"

# Run a specific test case name (passed to Go's -run filter)
./scripts/process/integration_test.sh --categories "tt" --specify "TestEditor_CustomEditorDynamicLaunch"

# Resume from the last failed test category
./scripts/process/integration_test.sh --resume
```

---

## Release and GitHub Distribution

Release and publishing pipelines are automated via scripts located in the [scripts/dist/](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist) directory.

### 1. Tool Release (tt)

Automates building, packaging, and publishing the `tt` CLI tool. Note that publishing requires GitHub CLI (`gh`) credentials.

#### All-in-One Quick Release (Recommended)
This runs the full build, packages the binaries, creates a GitHub Release, and publishes update manifests to Scoop/Homebrew:

```bash
# A. Release by incrementing patch version (e.g., v2.0.0 -> v2.0.1)
./scripts/dist/tool/release.sh tt

# B. Release with a specific version name
./scripts/dist/tool/release.sh tt v2.1.0

# C. Release by incrementing minor version (e.g., +v0.1.0)
./scripts/dist/tool/release.sh tt +v0.1.0
```

#### Manual Step-by-Step Release Flow
You can trigger each step of the release pipeline individually for fine-grained control:

##### Step 1: Build cross-platform binaries
```bash
./scripts/dist/tool/internal/build.sh tt
```

##### Step 2: Packaging release artifacts
```bash
./scripts/dist/tool/internal/package.sh tt v2.0.0
```

> [!TIP]
> **Verify before publishing**:
> You can manually verify the packaged artifacts (e.g., check that the correct binaries and Windows installer scripts/README are included inside the zip/tar.gz files) under `dist/tt/v2.0.0/` before proceeding to the publish step.

##### Step 3: Publish to distribution channels
```bash
./scripts/dist/tool/internal/publish.sh tt v2.0.0
```

### 2. Content Release (Catalog Templates)

Automates building the templatizer, packaging catalog originals into scaffolds, and pushing updates to the current active branch of the remote repository:

```bash
./scripts/dist/content/release.sh
```

---

## 開発とリリースのワークフロー

本プロジェクトでは、AIアシスト開発ワークフローと、ブランチベースのリリースパイプラインを統合した開発プロセスを採用しています。

### 1. 開発ループ (AIアシスト)

ツールの実装変更やコンテンツ（カタログテンプレート）の更新を行う場合は、まず新しいフィーチャーブランチ（作業ブランチ）を作成し、以下の定義された開発フェーズに従って進めます。

```
  ブランチ作成 ──> 仕様書作成 ──> 実装計画作成 ──> 実装実行 ──> ビルド・検証
```

#### 各開発フェーズ
1. **仕様書作成 (Specification)** — 開発要件や仕様を `prompts/phases/` 配下に定義します。作成には [create-specification](.agent/workflows/create-specification.md) ワークフローを使用します。
2. **実装計画作成 (Implementation Plan)** — 具体的な実装手順や検証計画を作成します。作成には [create-implementation-plan](.agent/workflows/create-implementation-plan.md) ワークフローを使用します。
3. **実装実行 (Execution)** — コードやテスト、カタログの修正を行います。実行には [execute-implementation-plan](.agent/workflows/execute-implementation-plan.md) ワークフローを使用します。
4. **ビルド・検証 (Verification)** — 実装した内容が正しく動作するかテストを実行します。検証には [build-pipeline](.agent/workflows/build-pipeline.md) ワークフローを使用します。

*各フェーズの移行時には、必ず人間によるレビューと承認が必要です。自動で次のフェーズに進むことはありません。*

---

### 2. リリースワークフロー

開発ブランチでの検証が完了した後、変更対象のコンポーネントに応じてリリース作業を行います。

#### A. ツールリリース (tt CLIツール)
CLIツール `tt` のバイナリおよび配布用アーカイブをコンパイルし、GitHub Releasesに直接アップロードして公開します。
- リリース用スクリプト `./scripts/dist/tool/release.sh tt` を実行してリリースを公開します。
- このスクリプトを実行すると、現在のコードベースに基づいてビルド、パッケージング、および配布チャネルへの公開（GitHub Releasesへのアップロードなど）が実行されます。
- 通常は、開発ブランチから `main` ブランチへPRをマージした後に、`main` ブランチまたはリリース専用ブランチから実行します。

#### B. コンテンツリリース (カタログテンプレート)
カタログテンプレート（`catalog/scaffolds/` やインデックス、メタデータ）はGitリポジトリ内に直接格納されるため、リリースはリポジトリの更新コミットとして行われます。
1. **コンテンツリリースの実行** — 作業中のカレントブランチ上で `./scripts/dist/content/release.sh` を実行します。これにより、自動的に検証用バイナリのビルド、カタログの再生成と検証が行われ、カタログ更新のコミットが自動生成された後、作業中のリモートブランチへ直接 `push` されます。
2. **プルリクエスト (PR) の作成** — カレントブランチがリモートにプッシュされた後、GitHub上で `main` ブランチに対するプルリクエスト（PR）を作成します。
3. **マージとリリース完了** — PRがレビューされ `main` ブランチにマージされることで、リリースが完了します。マージされた瞬間から、すべての `tt` クライアントが最新のカタログテンプレートを利用可能になります。

## Contributing

### Creating a New Feature

New features are generated from templates using the `scaffold` command and placed under `features/`:

```bash
tt scaffold [category] [name]
```

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
