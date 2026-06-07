# 000-DevScript-Part1

> **Source Specification**: [000-DevScript.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/000-DevScript.md)

## Goal Description

devctl の基盤実装を行う。Go モジュール初期化、型定義、マトリクスエンジン、設定ファイル読み込み、CLI 骨格、および基本アクション（`--up`, `--down`, `--status`, `--shell`, `--exec`）を実装する。

本パートでは以下を対象とする：
- Go プロジェクト骨格の構築（`features/devctl/`）
- 開発用コンテナ環境（`.devcontainer/`, `Dockerfile`）
- マトリクスエンジン（OS × Editor × Container mode → Capability 判定）
- 設定ファイル読み込み（`.devrc.yaml`, `feature.yaml`）
- CLI コマンド定義（cobra ベース）
- コンテナ操作アクション（up, down, status, shell, exec）
- `.gitignore` への `bin/` 追加

エディタ起動（`--open`）、SSH モード、devcontainer.json 解釈は Part 2 で扱う。

## User Review Required

> [!IMPORTANT]
> 本実装計画は Part 1 / Part 2 に分割しています。Part 1 はマトリクスエンジンと基本コンテナ操作を対象とし、Part 2 でエディタ連携・SSH・devcontainer 統合を扱います。この分割方針についてご確認ください。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: feature 単位の開発環境管理 | `internal/resolve/worktree.go` |
| R2: コンテナライフサイクル管理 (`--up`, `--down`, `--status`) | `internal/action/up.go`, `down.go`, `status.go` |
| R3: エディタ/エージェント起動 (`--open`, `--editor`) | **Part 2 で対応**（エディタ解決ロジックは Part 1 で `internal/detect/editor.go` に実装） |
| R4: エディタ別の接続方式 | **Part 2 で対応** |
| R5: コンテナ内操作 (`--shell`, `--exec`) | `internal/action/shell.go`, `exec.go` |
| R6: SSH モード | **Part 2 で対応** |
| R7: マトリクス駆動の分岐制御 | `internal/matrix/` パッケージ全体 |
| コンテナ識別（名前・イメージ名） | `internal/resolve/container.go` |
| 設定ファイル読み込み | `internal/resolve/config.go` |
| ログ仕様 | `internal/log/logger.go` |
| 終了コード・エラー処理 | `cmd/root.go` + 各アクション |
| `.devcontainer/` + `Dockerfile` (devctl 自身の開発環境) | `features/devctl/.devcontainer/`, `Dockerfile` |
| `bin/` を `.gitignore` に追加 | `.gitignore` |

## Proposed Changes

### プロジェクト基盤

#### [NEW] [go.mod](file:///c:/Users/yamya/myprog/escape/features/devctl/go.mod)
*   **Description**: Go モジュール初期化
*   **Technical Design**:
    ```go
    module github.com/escape-dev/devctl

    go 1.22
    ```
*   **Logic**:
    *   モジュール名は `github.com/escape-dev/devctl` とする（プロジェクトに合わせて調整可）
    *   依存: `cobra` (v2+), `viper` (v2+), `gopkg.in/yaml.v3`, `testify`

#### [NEW] [main.go](file:///c:/Users/yamya/myprog/escape/features/devctl/main.go)
*   **Description**: エントリポイント
*   **Technical Design**:
    ```go
    package main

    import (
        "os"
        "github.com/escape-dev/devctl/cmd"
    )

    func main() {
        if err := cmd.Execute(); err != nil {
            os.Exit(1)
        }
    }
    ```

---

### `.gitignore` 更新

#### [MODIFY] [.gitignore](file:///c:/Users/yamya/myprog/escape/.gitignore)
*   **Description**: `bin/` ディレクトリをバージョン管理対象外にする
*   **Logic**: ファイル末尾に以下を追加：
    ```
    # Build output
    bin/
    ```

---

### 開発用コンテナ

#### [NEW] [Dockerfile](file:///c:/Users/yamya/myprog/escape/features/devctl/Dockerfile)
*   **Description**: devctl 開発用 Dockerfile
*   **Technical Design**:
    ```dockerfile
    FROM golang:1.22-bookworm

    # Install Docker CLI (for testing container operations)
    RUN apt-get update && apt-get install -y \
        docker.io \
        git \
        && rm -rf /var/lib/apt/lists/*

    # Install Go tools
    RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

    WORKDIR /workspace
    ```

#### [NEW] [devcontainer.json](file:///c:/Users/yamya/myprog/escape/features/devctl/.devcontainer/devcontainer.json)
*   **Description**: VSCode / Cursor 用 Dev Container 定義
*   **Technical Design**:
    ```json
    {
      "name": "devctl-dev",
      "build": {
        "dockerfile": "../Dockerfile"
      },
      "workspaceFolder": "/workspace",
      "customizations": {
        "vscode": {
          "extensions": [
            "golang.go"
          ]
        }
      },
      "mounts": [
        "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
      ],
      "remoteUser": "root"
    }
    ```

---

### ログパッケージ (`internal/log/`)

#### [NEW] [logger_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/log/logger_test.go)
*   **Description**: ログ出力のテスト
*   **Technical Design**:
    ```go
    package log_test

    import (
        "bytes"
        "testing"

        "github.com/escape-dev/devctl/internal/log"
        "github.com/stretchr/testify/assert"
    )

    func TestLogger_LevelFiltering(t *testing.T) {
        tests := []struct {
            name     string
            verbose  bool
            logFunc  func(l *log.Logger)
            expected string
            notIn    string
        }{
            {
                name:     "info is always visible",
                verbose:  false,
                logFunc:  func(l *log.Logger) { l.Info("hello") },
                expected: "[INFO]",
            },
            {
                name:     "debug hidden when not verbose",
                verbose:  false,
                logFunc:  func(l *log.Logger) { l.Debug("detail") },
                notIn:    "[DEBUG]",
            },
            {
                name:     "debug visible when verbose",
                verbose:  true,
                logFunc:  func(l *log.Logger) { l.Debug("detail") },
                expected: "[DEBUG]",
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                var buf bytes.Buffer
                l := log.New(&buf, tt.verbose)
                tt.logFunc(l)
                output := buf.String()
                if tt.expected != "" {
                    assert.Contains(t, output, tt.expected)
                }
                if tt.notIn != "" {
                    assert.NotContains(t, output, tt.notIn)
                }
            })
        }
    }
    ```

#### [NEW] [logger.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/log/logger.go)
*   **Description**: INFO / WARN / ERROR / DEBUG レベルのログ出力
*   **Technical Design**:
    ```go
    package log

    import (
        "fmt"
        "io"
    )

    // Level represents log severity.
    type Level int

    const (
        LevelDebug Level = iota
        LevelInfo
        LevelWarn
        LevelError
    )

    // Logger provides leveled logging output.
    type Logger struct {
        out     io.Writer
        verbose bool
    }

    // New creates a Logger. If verbose is true, DEBUG messages are shown.
    func New(out io.Writer, verbose bool) *Logger {
        return &Logger{out: out, verbose: verbose}
    }

    func (l *Logger) log(level Level, prefix, format string, args ...any) {
        if level == LevelDebug && !l.verbose {
            return
        }
        msg := fmt.Sprintf(format, args...)
        fmt.Fprintf(l.out, "%s %s\n", prefix, msg)
    }

    func (l *Logger) Info(format string, args ...any)  { l.log(LevelInfo, "[INFO]", format, args...) }
    func (l *Logger) Warn(format string, args ...any)  { l.log(LevelWarn, "[WARN]", format, args...) }
    func (l *Logger) Error(format string, args ...any) { l.log(LevelError, "[ERROR]", format, args...) }
    func (l *Logger) Debug(format string, args ...any) { l.log(LevelDebug, "[DEBUG]", format, args...) }
    ```

---

### 検出パッケージ (`internal/detect/`)

#### [NEW] [os_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/detect/os_test.go)
*   **Description**: OS 検出のテスト
*   **Technical Design**:
    ```go
    package detect_test

    import (
        "runtime"
        "testing"

        "github.com/escape-dev/devctl/internal/detect"
        "github.com/stretchr/testify/assert"
    )

    func TestDetectOS(t *testing.T) {
        got := detect.CurrentOS()
        // Should return a valid OS value matching runtime.GOOS
        switch runtime.GOOS {
        case "linux":
            assert.Equal(t, detect.OSLinux, got)
        case "darwin":
            assert.Equal(t, detect.OSMacOS, got)
        case "windows":
            assert.Equal(t, detect.OSWindows, got)
        default:
            t.Fatalf("unexpected GOOS: %s", runtime.GOOS)
        }
    }
    ```

#### [NEW] [os.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/detect/os.go)
*   **Description**: 実行環境の OS を検出する
*   **Technical Design**:
    ```go
    package detect

    import "runtime"

    // OS represents the detected operating system.
    type OS string

    const (
        OSLinux   OS = "linux"
        OSMacOS   OS = "macos"
        OSWindows OS = "windows"
    )

    // CurrentOS returns the detected OS for the current platform.
    func CurrentOS() OS {
        switch runtime.GOOS {
        case "darwin":
            return OSMacOS
        case "windows":
            return OSWindows
        default:
            return OSLinux
        }
    }
    ```

#### [NEW] [editor_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/detect/editor_test.go)
*   **Description**: editor 解釈・解決のテスト
*   **Technical Design**:
    ```go
    package detect_test

    import (
        "testing"

        "github.com/escape-dev/devctl/internal/detect"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
    )

    func TestParseEditor(t *testing.T) {
        tests := []struct {
            name    string
            input   string
            want    detect.Editor
            wantErr bool
        }{
            {"code", "code", detect.EditorVSCode, false},
            {"cursor", "cursor", detect.EditorCursor, false},
            {"ag", "ag", detect.EditorAG, false},
            {"claude", "claude", detect.EditorClaude, false},
            {"vscode alias", "vscode", detect.EditorVSCode, false},
            {"invalid", "vim", detect.Editor(""), true},
            {"empty", "", detect.Editor(""), true},
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got, err := detect.ParseEditor(tt.input)
                if tt.wantErr {
                    require.Error(t, err)
                } else {
                    require.NoError(t, err)
                    assert.Equal(t, tt.want, got)
                }
            })
        }
    }

    func TestResolveEditor(t *testing.T) {
        // Resolution priority: CLI flag > env var > feature config > global config > default
        tests := []struct {
            name          string
            cliFlag       string
            envVar        string
            featureConfig string
            globalConfig  string
            want          detect.Editor
        }{
            {
                name:    "cli flag takes highest priority",
                cliFlag: "code",
                envVar:  "ag",
                want:    detect.EditorVSCode,
            },
            {
                name:   "env var overrides config",
                envVar: "ag",
                globalConfig: "cursor",
                want:   detect.EditorAG,
            },
            {
                name:          "feature config overrides global",
                featureConfig: "claude",
                globalConfig:  "cursor",
                want:          detect.EditorClaude,
            },
            {
                name:         "global config used as fallback",
                globalConfig: "code",
                want:         detect.EditorVSCode,
            },
            {
                name: "default is cursor when nothing set",
                want: detect.EditorCursor,
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got, err := detect.ResolveEditor(tt.cliFlag, tt.envVar, tt.featureConfig, tt.globalConfig)
                require.NoError(t, err)
                assert.Equal(t, tt.want, got)
            })
        }
    }

    func TestResolveEditor_InvalidValue(t *testing.T) {
        _, err := detect.ResolveEditor("vim", "", "", "")
        require.Error(t, err)
    }
    ```

#### [NEW] [editor.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/detect/editor.go)
*   **Description**: editor 文字列の解釈・解決ロジックと型定義
*   **Technical Design**:
    ```go
    package detect

    import "fmt"

    // Editor represents a supported editor or agent.
    type Editor string

    const (
        EditorVSCode Editor = "code"
        EditorCursor Editor = "cursor"
        EditorAG     Editor = "ag"
        EditorClaude Editor = "claude"
    )

    // EnvKeyEditor is the environment variable name for the default editor.
    const EnvKeyEditor = "DEVCTL_EDITOR"

    // ParseEditor parses a string into an Editor value.
    // Accepts aliases: "vscode" -> "code".
    func ParseEditor(s string) (Editor, error) {
        switch s {
        case "code", "vscode":
            return EditorVSCode, nil
        case "cursor":
            return EditorCursor, nil
        case "ag", "antigravity":
            return EditorAG, nil
        case "claude":
            return EditorClaude, nil
        default:
            return "", fmt.Errorf("unknown editor: %q (supported: code, cursor, ag, claude)", s)
        }
    }

    // ResolveEditor determines the editor using the following priority:
    //   1. CLI flag (cliFlag)
    //   2. Environment variable (envValue, from DEVCTL_EDITOR)
    //   3. Feature-level config (featureConfig)
    //   4. Global config (globalConfig, from .devrc.yaml)
    //   5. Default: "cursor"
    func ResolveEditor(cliFlag, envValue, featureConfig, globalConfig string) (Editor, error) {
        sources := []string{cliFlag, envValue, featureConfig, globalConfig}
        for _, src := range sources {
            if src != "" {
                return ParseEditor(src)
            }
        }
        return EditorCursor, nil // default
    }
    ```
*   **Logic**:
    *   優先順位: CLI フラグ `--editor` > 環境変数 `DEVCTL_EDITOR` > `feature.yaml` の `editor_default` > `.devrc.yaml` の `default_editor` > ハードコードデフォルト `cursor`
    *   呼び出し元（`cmd/root.go`）が `os.Getenv(detect.EnvKeyEditor)` を渡す

---

### マトリクスパッケージ (`internal/matrix/`)

#### [NEW] [types.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/matrix/types.go)
*   **Description**: マトリクス制御軸の型定義
*   **Technical Design**:
    ```go
    package matrix

    import "github.com/escape-dev/devctl/internal/detect"

    // ContainerMode represents how containers are used.
    type ContainerMode string

    const (
        ContainerNone         ContainerMode = "none"
        ContainerDevContainer ContainerMode = "devcontainer"
        ContainerDockerLocal  ContainerMode = "docker-local"
        ContainerDockerSSH    ContainerMode = "docker-ssh"
    )

    // Action represents the user-requested operation.
    type Action string

    const (
        ActionUp     Action = "up"
        ActionOpen   Action = "open"
        ActionUpOpen Action = "up_open"
        ActionDown   Action = "down"
        ActionShell  Action = "shell"
        ActionExec   Action = "exec"
        ActionStatus Action = "status"
    )

    // CompatLevel represents the support level for a combination.
    type CompatLevel int

    const (
        L1Supported  CompatLevel = iota // Full support
        L2BestEffort                     // Try, fallback on failure
        L3Fallback                       // Direct fallback
        L4Unsupported                    // Error or no-op
    )

    // String returns the human-readable level name.
    func (l CompatLevel) String() string {
        switch l {
        case L1Supported:
            return "L1:Supported"
        case L2BestEffort:
            return "L2:BestEffort"
        case L3Fallback:
            return "L3:Fallback"
        case L4Unsupported:
            return "L4:Unsupported"
        default:
            return "Unknown"
        }
    }

    // Context holds the resolved environment for matrix lookup.
    type Context struct {
        OS            detect.OS
        Editor        detect.Editor
        ContainerMode ContainerMode
        Action        Action
    }
    ```

#### [NEW] [capability.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/matrix/capability.go)
*   **Description**: Capability オブジェクト定義
*   **Technical Design**:
    ```go
    package matrix

    // Capability describes what features are available for a given combination.
    type Capability struct {
        CanOpenLocal              bool
        CanTryDevcontainerAttach  bool
        CanUseSSHMode             bool
        CanLaunchNewWindow        bool
        CanRunClaudeLocally       bool
        CanRunClaudeInContainer   bool
        RequiresBestEffort        bool
        LocalOpenLevel            CompatLevel
        DevcontainerOpenLevel     CompatLevel
        SSHLevel                  CompatLevel
    }
    ```

#### [NEW] [matrix_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/matrix/matrix_test.go)
*   **Description**: マトリクスルックアップのテスト
*   **Technical Design**:
    ```go
    package matrix_test

    import (
        "testing"

        "github.com/escape-dev/devctl/internal/detect"
        "github.com/escape-dev/devctl/internal/matrix"
        "github.com/stretchr/testify/assert"
    )

    func TestResolveCapability(t *testing.T) {
        tests := []struct {
            name                  string
            os                    detect.OS
            editor                detect.Editor
            wantLocalOpen         matrix.CompatLevel
            wantDevcontainerOpen  matrix.CompatLevel
        }{
            {
                name:                 "linux+vscode",
                os:                   detect.OSLinux,
                editor:               detect.EditorVSCode,
                wantLocalOpen:        matrix.L1Supported,
                wantDevcontainerOpen: matrix.L2BestEffort,
            },
            {
                name:                 "macos+ag",
                os:                   detect.OSMacOS,
                editor:               detect.EditorAG,
                wantLocalOpen:        matrix.L1Supported,
                wantDevcontainerOpen: matrix.L4Unsupported,
            },
            {
                name:                 "windows+claude",
                os:                   detect.OSWindows,
                editor:               detect.EditorClaude,
                wantLocalOpen:        matrix.L1Supported,
                wantDevcontainerOpen: matrix.L1Supported,
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                cap := matrix.ResolveCapability(tt.os, tt.editor)
                assert.Equal(t, tt.wantLocalOpen, cap.LocalOpenLevel)
                assert.Equal(t, tt.wantDevcontainerOpen, cap.DevcontainerOpenLevel)
            })
        }
    }

    func TestResolveCapability_AllCombinations(t *testing.T) {
        // Ensure every OS×Editor combination returns a valid capability
        oses := []detect.OS{detect.OSLinux, detect.OSMacOS, detect.OSWindows}
        editors := []detect.Editor{detect.EditorVSCode, detect.EditorCursor, detect.EditorAG, detect.EditorClaude}

        for _, os := range oses {
            for _, editor := range editors {
                t.Run(string(os)+"+"+string(editor), func(t *testing.T) {
                    cap := matrix.ResolveCapability(os, editor)
                    assert.True(t, cap.CanOpenLocal, "CanOpenLocal should always be true")
                })
            }
        }
    }
    ```

#### [NEW] [matrix.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/matrix/matrix.go)
*   **Description**: マトリクスルックアップエンジン
*   **Technical Design**:
    ```go
    package matrix

    import "github.com/escape-dev/devctl/internal/detect"

    // ResolveCapability returns the Capability for the given OS and Editor.
    // This is the central matrix lookup function.
    func ResolveCapability(os detect.OS, editor detect.Editor) Capability {
        key := matrixKey{os: os, editor: editor}
        if cap, ok := defaultMatrix[key]; ok {
            return cap
        }
        // Fallback: local open only
        return Capability{
            CanOpenLocal:          true,
            LocalOpenLevel:        L1Supported,
            DevcontainerOpenLevel: L4Unsupported,
            SSHLevel:              L4Unsupported,
        }
    }

    type matrixKey struct {
        os     detect.OS
        editor detect.Editor
    }

    // defaultMatrix encodes the specification's OS×Editor compatibility table.
    var defaultMatrix = map[matrixKey]Capability{
        // --- Linux ---
        {detect.OSLinux, detect.EditorVSCode}: {
            CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
        },
        {detect.OSLinux, detect.EditorCursor}: {
            CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
        },
        {detect.OSLinux, detect.EditorAG}: {
            CanOpenLocal: true, CanTryDevcontainerAttach: false, CanUseSSHMode: true,
            RequiresBestEffort: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L2BestEffort,
        },
        {detect.OSLinux, detect.EditorClaude}: {
            CanOpenLocal: true, CanRunClaudeLocally: true, CanUseSSHMode: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L1Supported,
        },
        // --- macOS ---
        {detect.OSMacOS, detect.EditorVSCode}: {
            CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
        },
        // ... (same pattern for macOS+Cursor, macOS+AG, macOS+Claude)
        // --- Windows ---
        {detect.OSWindows, detect.EditorVSCode}: {
            CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
            RequiresBestEffort: true,
            LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L2BestEffort,
        },
        // ... (all 12 combinations as per specification)
    }
    ```
*   **Logic**:
    *   仕様書の「OS × Editor の基本互換マトリクス」表を `defaultMatrix` マップにエンコードする
    *   全 12 組み合わせ (3 OS × 4 Editor) を網羅する
    *   将来的に `.devrc.yaml` の `compatibility` セクションでオーバーライド可能にする

---

### 解決パッケージ (`internal/resolve/`)

#### [NEW] [worktree_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/worktree_test.go)
*   **Description**: worktree 解決のテスト
*   **Technical Design**:
    ```go
    package resolve_test

    import (
        "os"
        "path/filepath"
        "testing"

        "github.com/escape-dev/devctl/internal/resolve"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
    )

    func TestResolveWorktree(t *testing.T) {
        // Create temp dir structure: <root>/work/test-feature/
        root := t.TempDir()
        featureDir := filepath.Join(root, "work", "test-feature")
        require.NoError(t, os.MkdirAll(featureDir, 0755))

        t.Run("existing worktree", func(t *testing.T) {
            path, err := resolve.Worktree(root, "test-feature")
            require.NoError(t, err)
            assert.Equal(t, featureDir, path)
        })

        t.Run("non-existing worktree", func(t *testing.T) {
            _, err := resolve.Worktree(root, "nonexistent")
            require.Error(t, err)
            assert.Contains(t, err.Error(), "not found")
        })
    }
    ```

#### [NEW] [worktree.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/worktree.go)
*   **Description**: feature 名から worktree パスを解決する
*   **Technical Design**:
    ```go
    package resolve

    import (
        "fmt"
        "os"
        "path/filepath"
    )

    // Worktree resolves the worktree path for the given feature.
    // Returns an error if the directory does not exist.
    func Worktree(repoRoot, feature string) (string, error) {
        path := filepath.Join(repoRoot, "work", feature)
        info, err := os.Stat(path)
        if err != nil {
            return "", fmt.Errorf("worktree for feature %q not found: %w", feature, err)
        }
        if !info.IsDir() {
            return "", fmt.Errorf("worktree path is not a directory: %s", path)
        }
        return path, nil
    }
    ```

#### [NEW] [container_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/container_test.go)
*   **Description**: コンテナ名・イメージ名解決のテスト
*   **Technical Design**:
    ```go
    package resolve_test

    import (
        "testing"

        "github.com/escape-dev/devctl/internal/resolve"
        "github.com/stretchr/testify/assert"
    )

    func TestContainerName(t *testing.T) {
        tests := []struct {
            project string
            feature string
            want    string
        }{
            {"myproj", "feature-a", "myproj-feature-a"},
            {"myproj", "feature_b", "myproj-feature-b"},
            {"myproj", "Feature.C", "myproj-feature-c"},
        }
        for _, tt := range tests {
            t.Run(tt.want, func(t *testing.T) {
                got := resolve.ContainerName(tt.project, tt.feature)
                assert.Equal(t, tt.want, got)
            })
        }
    }

    func TestImageName(t *testing.T) {
        got := resolve.ImageName("myproj", "feature-a")
        assert.Equal(t, "myproj-dev-feature-a", got)
    }
    ```

#### [NEW] [container.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/container.go)
*   **Description**: コンテナ名・イメージ名の解決
*   **Technical Design**:
    ```go
    package resolve

    import (
        "regexp"
        "strings"
    )

    var invalidChars = regexp.MustCompile(`[^a-z0-9-]`)

    // sanitize converts a string to a valid container name component.
    func sanitize(s string) string {
        return invalidChars.ReplaceAllString(strings.ToLower(s), "-")
    }

    // ContainerName returns "<project>-<feature>" with invalid chars replaced.
    func ContainerName(project, feature string) string {
        return sanitize(project) + "-" + sanitize(feature)
    }

    // ImageName returns "<project>-dev-<feature>" with invalid chars replaced.
    func ImageName(project, feature string) string {
        return sanitize(project) + "-dev-" + sanitize(feature)
    }
    ```

#### [NEW] [config.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/config.go)
*   **Description**: `.devrc.yaml` と `feature.yaml` の読み込み
*   **Technical Design**:
    ```go
    package resolve

    import (
        "os"
        "path/filepath"

        "gopkg.in/yaml.v3"
    )

    // GlobalConfig represents .devrc.yaml at the repo root.
    type GlobalConfig struct {
        DefaultEditor        string `yaml:"default_editor"`
        ProjectName          string `yaml:"project_name"`
        WorkDir              string `yaml:"work_dir"`
        DefaultContainerMode string `yaml:"default_container_mode"`
    }

    // FeatureConfig represents feature-level dev settings.
    type FeatureConfig struct {
        Dev struct {
            EditorDefault string `yaml:"editor_default"`
            SSHSupported  bool   `yaml:"ssh_supported"`
            Shell         string `yaml:"shell"`
        } `yaml:"dev"`
    }

    // LoadGlobalConfig loads .devrc.yaml from repoRoot.
    // Returns a zero-value config if the file does not exist.
    func LoadGlobalConfig(repoRoot string) (GlobalConfig, error) {
        var cfg GlobalConfig
        path := filepath.Join(repoRoot, ".devrc.yaml")
        data, err := os.ReadFile(path)
        if os.IsNotExist(err) {
            // Defaults
            cfg.DefaultEditor = "cursor"
            cfg.WorkDir = "work"
            cfg.DefaultContainerMode = "docker-local"
            return cfg, nil
        }
        if err != nil {
            return cfg, err
        }
        if err := yaml.Unmarshal(data, &cfg); err != nil {
            return cfg, err
        }
        if cfg.WorkDir == "" {
            cfg.WorkDir = "work"
        }
        if cfg.DefaultEditor == "" {
            cfg.DefaultEditor = "cursor"
        }
        if cfg.DefaultContainerMode == "" {
            cfg.DefaultContainerMode = "docker-local"
        }
        return cfg, nil
    }

    // LoadFeatureConfig loads feature.yaml from the feature directory.
    // Searches work/<feature>/feature.yaml then features/<feature>/feature.yaml.
    func LoadFeatureConfig(repoRoot, feature string) (FeatureConfig, error) {
        var cfg FeatureConfig
        candidates := []string{
            filepath.Join(repoRoot, "work", feature, "feature.yaml"),
            filepath.Join(repoRoot, "features", feature, "feature.yaml"),
        }
        for _, path := range candidates {
            data, err := os.ReadFile(path)
            if os.IsNotExist(err) {
                continue
            }
            if err != nil {
                return cfg, err
            }
            if err := yaml.Unmarshal(data, &cfg); err != nil {
                return cfg, err
            }
            return cfg, nil
        }
        // No feature config found; return defaults
        return cfg, nil
    }
    ```

#### [NEW] [config_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/config_test.go)
*   **Description**: 設定ファイル読み込みのテスト
*   **Technical Design**:
    ```go
    package resolve_test

    import (
        "os"
        "path/filepath"
        "testing"

        "github.com/escape-dev/devctl/internal/resolve"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
    )

    func TestLoadGlobalConfig_Defaults(t *testing.T) {
        root := t.TempDir()
        cfg, err := resolve.LoadGlobalConfig(root)
        require.NoError(t, err)
        assert.Equal(t, "cursor", cfg.DefaultEditor)
        assert.Equal(t, "work", cfg.WorkDir)
        assert.Equal(t, "docker-local", cfg.DefaultContainerMode)
    }

    func TestLoadGlobalConfig_FromFile(t *testing.T) {
        root := t.TempDir()
        content := `
project_name: testproj
default_editor: code
work_dir: workspaces
default_container_mode: devcontainer
`
        require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0644))

        cfg, err := resolve.LoadGlobalConfig(root)
        require.NoError(t, err)
        assert.Equal(t, "testproj", cfg.ProjectName)
        assert.Equal(t, "code", cfg.DefaultEditor)
        assert.Equal(t, "workspaces", cfg.WorkDir)
        assert.Equal(t, "devcontainer", cfg.DefaultContainerMode)
    }

    func TestLoadFeatureConfig_NotFound(t *testing.T) {
        root := t.TempDir()
        cfg, err := resolve.LoadFeatureConfig(root, "nonexistent")
        require.NoError(t, err)
        assert.Zero(t, cfg) // default zero-value
    }
    ```

---

### CLI コマンド定義 (`cmd/`)

#### [NEW] [root.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/root.go)
*   **Description**: cobra ルートコマンドと全オプションの定義
*   **Technical Design**:
    ```go
    package cmd

    import (
        "fmt"
        "os"

        "github.com/spf13/cobra"
        "github.com/escape-dev/devctl/internal/log"
    )

    var (
        flagUp      bool
        flagOpen    bool
        flagDown    bool
        flagStatus  bool
        flagShell   bool
        flagExec    []string
        flagEditor  string
        flagSSH     bool
        flagVerbose bool
        flagDryRun  bool
        flagForce   bool
        flagRebuild bool
        flagNoBuild bool
    )

    var rootCmd = &cobra.Command{
        Use:   "devctl <feature> [flags]",
        Short: "Development environment orchestrator",
        Long:  "devctl manages feature-level development environments with matrix-driven control.",
        Args:  cobra.MinimumNArgs(1),
        RunE:  runRoot,
    }

    func init() {
        rootCmd.Flags().BoolVar(&flagUp, "up", false, "Start the container")
        rootCmd.Flags().BoolVar(&flagOpen, "open", false, "Open the editor")
        rootCmd.Flags().BoolVar(&flagDown, "down", false, "Stop and remove the container")
        rootCmd.Flags().BoolVar(&flagStatus, "status", false, "Show feature status")
        rootCmd.Flags().BoolVar(&flagShell, "shell", false, "Open a shell in the container")
        rootCmd.Flags().StringSliceVar(&flagExec, "exec", nil, "Execute a command in the container")
        rootCmd.Flags().StringVar(&flagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
        rootCmd.Flags().BoolVar(&flagSSH, "ssh", false, "Enable SSH mode")
        rootCmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Show debug logs")
        rootCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show planned actions without executing")
        rootCmd.Flags().BoolVar(&flagForce, "force", false, "Skip confirmation prompts")
        rootCmd.Flags().BoolVar(&flagRebuild, "rebuild", false, "Rebuild the container image")
        rootCmd.Flags().BoolVar(&flagNoBuild, "no-build", false, "Skip image build")
    }

    // Execute runs the root command.
    func Execute() error {
        return rootCmd.Execute()
    }

    func runRoot(cmd *cobra.Command, args []string) error {
        feature := args[0]
        logger := log.New(os.Stderr, flagVerbose)
        _ = feature
        _ = logger
        // Action dispatch will be implemented here.
        // Step 1: detect environment
        // Step 2: resolve capabilities
        // Step 3: plan actions
        // Step 4: execute
        // Step 5: fallback if needed
        fmt.Fprintf(os.Stderr, "[INFO] devctl: feature=%s (not yet implemented)\n", feature)
        return nil
    }
    ```
*   **Logic**:
    *   第1引数を feature 名として取得
    *   各フラグの組み合わせに応じて内部的にアクションを決定
    *   `--up` も `--open` もない場合は usage エラーとする（将来実装）
    *   実際のアクション dispatch は Step-by-Step で段階的に実装

---

### アクションパッケージ (`internal/action/`)

#### [NEW] [runner.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/runner.go)
*   **Description**: Docker コマンド実行の共通ヘルパー
*   **Technical Design**:
    ```go
    package action

    import (
        "fmt"
        "os"
        "os/exec"

        "github.com/escape-dev/devctl/internal/log"
    )

    // Runner executes Docker commands.
    type Runner struct {
        Logger *log.Logger
        DryRun bool
    }

    // DockerRun executes "docker <args...>".
    // In dry-run mode, it only logs the command.
    func (r *Runner) DockerRun(args ...string) error {
        r.Logger.Debug("docker %v", args)
        if r.DryRun {
            r.Logger.Info("[DRY-RUN] docker %v", args)
            return nil
        }
        cmd := exec.Command("docker", args...)
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.Stdin = os.Stdin
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("docker %v: %w", args, err)
        }
        return nil
    }

    // DockerRunOutput executes "docker <args...>" and returns stdout.
    func (r *Runner) DockerRunOutput(args ...string) (string, error) {
        r.Logger.Debug("docker %v", args)
        cmd := exec.Command("docker", args...)
        cmd.Stderr = os.Stderr
        out, err := cmd.Output()
        if err != nil {
            return "", fmt.Errorf("docker %v: %w", args, err)
        }
        return string(out), nil
    }
    ```

#### [NEW] [status.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/status.go)
*   **Description**: feature のステータス確認
*   **Technical Design**:
    ```go
    package action

    import (
        "fmt"
        "strings"
    )

    // FeatureState represents the state of a feature environment.
    type FeatureState string

    const (
        StateNotFound         FeatureState = "NOT_FOUND"
        StateWorktreeOnly     FeatureState = "WORKTREE_ONLY"
        StateContainerRunning FeatureState = "CONTAINER_RUNNING"
        StateContainerStopped FeatureState = "CONTAINER_STOPPED"
    )

    // Status checks the state of a feature's container.
    func (r *Runner) Status(containerName, worktreePath string) FeatureState {
        // Check container state
        out, err := r.DockerRunOutput("inspect", "--format", "{{.State.Running}}", containerName)
        if err != nil {
            // Container does not exist
            if worktreePath != "" {
                return StateWorktreeOnly
            }
            return StateNotFound
        }
        if strings.TrimSpace(out) == "true" {
            return StateContainerRunning
        }
        return StateContainerStopped
    }

    // PrintStatus displays the feature status to the user.
    func (r *Runner) PrintStatus(feature, containerName, worktreePath string) {
        state := r.Status(containerName, worktreePath)
        r.Logger.Info("Feature: %s", feature)
        r.Logger.Info("Container: %s", containerName)
        r.Logger.Info("Worktree: %s", worktreePath)
        r.Logger.Info("State: %s", state)
        switch state {
        case StateContainerRunning:
            fmt.Println("✅ Container is running")
        case StateContainerStopped:
            fmt.Println("⏸  Container is stopped")
        case StateWorktreeOnly:
            fmt.Println("📁 Worktree exists, no container")
        case StateNotFound:
            fmt.Println("❌ Not found")
        }
    }
    ```

#### [NEW] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/up.go)
*   **Description**: コンテナ起動
*   **Technical Design**:
    ```go
    package action

    // UpOptions holds parameters for the up action.
    type UpOptions struct {
        ContainerName string
        ImageName     string
        WorktreePath  string
        Rebuild       bool
        NoBuild       bool
        SSHMode       bool
        Env           map[string]string
    }

    // Up starts the development container.
    func (r *Runner) Up(opts UpOptions) error {
        // Step 1: Check if container already exists and is running
        state := r.Status(opts.ContainerName, opts.WorktreePath)
        if state == StateContainerRunning {
            r.Logger.Info("Container %s is already running", opts.ContainerName)
            return nil
        }

        // Step 2: Build image if needed
        if !opts.NoBuild {
            if opts.Rebuild || !r.imageExists(opts.ImageName) {
                r.Logger.Info("Building image %s...", opts.ImageName)
                if err := r.buildImage(opts); err != nil {
                    return err
                }
            }
        }

        // Step 3: Remove stopped container if exists
        if state == StateContainerStopped {
            r.Logger.Info("Removing stopped container %s...", opts.ContainerName)
            _ = r.DockerRun("rm", opts.ContainerName)
        }

        // Step 4: Run container
        args := []string{
            "run", "-d",
            "--name", opts.ContainerName,
            "-v", opts.WorktreePath + ":/workspace",
            "-w", "/workspace",
        }

        // Add environment variables
        for k, v := range opts.Env {
            args = append(args, "-e", k+"="+v)
        }

        // SSH mode
        if opts.SSHMode {
            args = append(args, "-e", "ENABLE_SSH=1")
        }

        args = append(args, opts.ImageName)

        r.Logger.Info("Starting container %s...", opts.ContainerName)
        if err := r.DockerRun(args...); err != nil {
            return err
        }

        r.Logger.Info("Container %s started successfully", opts.ContainerName)
        return nil
    }

    func (r *Runner) imageExists(imageName string) bool {
        _, err := r.DockerRunOutput("image", "inspect", imageName)
        return err == nil
    }

    func (r *Runner) buildImage(opts UpOptions) error {
        // Attempt to build from Dockerfile in worktree
        return r.DockerRun("build", "-t", opts.ImageName, opts.WorktreePath)
    }
    ```

#### [NEW] [down.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/down.go)
*   **Description**: コンテナ停止・削除
*   **Technical Design**:
    ```go
    package action

    // Down stops and removes the development container.
    func (r *Runner) Down(containerName string) error {
        r.Logger.Info("Stopping container %s...", containerName)
        if err := r.DockerRun("stop", containerName); err != nil {
            r.Logger.Warn("Stop failed (may already be stopped): %v", err)
        }

        r.Logger.Info("Removing container %s...", containerName)
        if err := r.DockerRun("rm", containerName); err != nil {
            return err
        }

        r.Logger.Info("Container %s removed successfully", containerName)
        return nil
    }
    ```

#### [NEW] [shell.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/shell.go)
*   **Description**: コンテナへの対話的シェル接続
*   **Technical Design**:
    ```go
    package action

    import (
        "os"
        "os/exec"
    )

    // Shell opens an interactive shell in the container.
    func (r *Runner) Shell(containerName string) error {
        r.Logger.Info("Opening shell in %s...", containerName)
        if r.DryRun {
            r.Logger.Info("[DRY-RUN] docker exec -it %s bash", containerName)
            return nil
        }
        cmd := exec.Command("docker", "exec", "-it", containerName, "bash")
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        return cmd.Run()
    }
    ```

#### [NEW] [exec.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/exec.go)
*   **Description**: コンテナ内コマンド実行
*   **Technical Design**:
    ```go
    package action

    import (
        "os"
        "os/exec"
    )

    // Exec runs a command in the container and returns its exit code.
    func (r *Runner) Exec(containerName string, command []string) error {
        r.Logger.Info("Executing in %s: %v", containerName, command)
        if r.DryRun {
            r.Logger.Info("[DRY-RUN] docker exec %s %v", containerName, command)
            return nil
        }
        args := append([]string{"exec", containerName}, command...)
        cmd := exec.Command("docker", args...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        return cmd.Run()
    }
    ```

---

## Step-by-Step Implementation Guide

1.  **Go モジュール初期化**:
    *   `features/devctl/` ディレクトリを作成
    *   `go mod init github.com/escape-dev/devctl` を実行
    *   `main.go` を作成（エントリポイント）

2.  **`.gitignore` 更新**:
    *   `.gitignore` に `bin/` を追加

3.  **開発用コンテナ定義**:
    *   `features/devctl/Dockerfile` を作成
    *   `features/devctl/.devcontainer/devcontainer.json` を作成

4.  **ログパッケージの実装** (TDD):
    *   `internal/log/logger_test.go` を作成し、テストが失敗することを確認
    *   `internal/log/logger.go` を実装し、テストが成功することを確認

5.  **検出パッケージの実装** (TDD):
    *   `internal/detect/os_test.go` → `os.go`
    *   `internal/detect/editor_test.go` → `editor.go`

6.  **マトリクスパッケージの実装** (TDD):
    *   `internal/matrix/types.go` を作成（型定義のみ）
    *   `internal/matrix/capability.go` を作成
    *   `internal/matrix/matrix_test.go` → `matrix.go`
    *   全 12 OS×Editor 組み合わせを `defaultMatrix` にエンコード

7.  **解決パッケージの実装** (TDD):
    *   `internal/resolve/worktree_test.go` → `worktree.go`
    *   `internal/resolve/container_test.go` → `container.go`
    *   `internal/resolve/config_test.go` → `config.go`

8.  **CLI コマンド定義**:
    *   `cmd/root.go` を作成（cobra ルートコマンド + フラグ定義）

9.  **アクションパッケージの実装**:
    *   `internal/action/runner.go` を作成（Docker コマンド実行ヘルパー）
    *   `internal/action/status.go` を作成
    *   `internal/action/up.go` を作成
    *   `internal/action/down.go` を作成
    *   `internal/action/shell.go` を作成
    *   `internal/action/exec.go` を作成

10. **依存関係の解決**:
    *   `go mod tidy` で依存を整理

11. **ビルド確認**:
    *   `scripts/process/build.sh` で全体ビルド・ユニットテストを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **検証内容**:
        *   Go ビルドが成功すること
        *   `internal/log/` のテスト: ログレベルフィルタリングが正しく動作
        *   `internal/detect/` のテスト: OS 検出と editor パースが正しい
        *   `internal/matrix/` のテスト: 全 12 組み合わせの Capability が仕様通り
        *   `internal/resolve/` のテスト: worktree 解決、コンテナ名、設定ファイル読み込み

2.  **Integration Tests** (Part 2 で追加予定):
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl"
    ```
    *   **注記**: コンテナ操作の統合テスト（`--up`, `--down` 等のE2Eテスト）は Docker 環境が必要なため、Part 2 の `tests/devctl/` カテゴリとして実装する

## Documentation

なし（新規プロジェクトのため既存ドキュメント更新不要）

---

## 継続計画について

本計画は Part 1 であり、以下は **Part 2** で対応する。

- エディタ起動アクション（`internal/editor/` パッケージ全体、`internal/action/open.go`）
- SSH モードの詳細実装
- `devcontainer.json` の解釈（`internal/resolve/devcontainer.go`）
- 実行計画構築（`internal/plan/planner.go`）
- `cmd/root.go` へのアクション dispatch 統合
- 統合テスト（`tests/devctl/`）
- `build.sh` への devctl ビルドステップ追加
