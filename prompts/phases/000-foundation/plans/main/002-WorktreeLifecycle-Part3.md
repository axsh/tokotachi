# 002-WorktreeLifecycle-Part3

> **Source Specification**: [001-WorktreeLifecycle.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/001-WorktreeLifecycle.md)

## Goal Description

devctl をサブコマンド体系に再構成し、外部コマンド実行の共通化（`cmdexec`）、実行レポート機能を実装する。
本 Part 3 では、R1（サブコマンド体系）、R7（外部コマンド共通化）、R10（実行レポート）を対象とする。

## User Review Required

> [!IMPORTANT]
> - 既存の `cmd/root.go`（フラグベース）を完全にサブコマンドベースに書き換える **破壊的変更**
> - `action.Runner` の `DockerRun`/`DockerRunOutput` メソッドを `cmdexec` パッケージに移行する
> - `internal/editor/*.go` の `exec.Command` 呼び出しも `cmdexec` 経由に変更する

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: サブコマンド体系への変更 | `cmd/root.go`, `cmd/up.go`, `cmd/down.go`, `cmd/open.go`, `cmd/status.go`, `cmd/shell.go`, `cmd/exec.go` |
| R7: 外部コマンド実行の共通化とログ出力 | `internal/cmdexec/cmdexec.go` |
| R10: 実行レポート | `internal/report/report.go` |
| R2-R6: worktree/state/pr/close | **Part 4 で対応** |
| R8: --list | **Part 4 で対応** |
| R9: --switch | **対象外（将来）** |

## Proposed Changes

### 外部コマンド実行共通化 (`internal/cmdexec/`)

#### [NEW] [cmdexec_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/cmdexec/cmdexec_test.go)
*   **Description**: cmdexec パッケージのユニットテスト（TDD 先行）
*   **テストケース**:
    -   `TestRun_Success`: 正常実行（`echo` 等の安全なコマンド）で stdout キャプチャ、ExecRecord 記録を検証
    -   `TestRun_Failure`: 存在しないコマンドでエラー返却と ExecRecord.Success=false を検証
    -   `TestRun_DryRun`: dryRun=true で実際のコマンドが実行されないことを検証
    -   `TestLogPrefix_Normal`: `[CMD]` プレフィックスがログに出力されることを検証
    -   `TestLogPrefix_DryRun`: `[DRY-RUN]` プレフィックスがログに出力されることを検証
    -   `TestRecorder_Collect`: 複数回の Run を実行後、Recorder.Records() で全記録を取得できることを検証
    -   `TestResolveCommand_Default`: 環境変数未設定時にデフォルト値が返ることを検証
    -   `TestResolveCommand_EnvOverride`: 環境変数設定時にその値が返ることを検証

#### [NEW] [cmdexec.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/cmdexec/cmdexec.go)
*   **Description**: 全外部コマンド実行を統一する共通パッケージ
*   **Technical Design**:
    ```go
    // ExecRecord stores one command execution history.
    type ExecRecord struct {
        Command  string        // full command line (e.g. "docker run -d ...")
        Success  bool
        ExitCode int
        Duration time.Duration
        DryRun   bool
    }

    // Recorder collects ExecRecords during a session.
    type Recorder struct { /* internal slice */ }
    func NewRecorder() *Recorder
    func (r *Recorder) Records() []ExecRecord
    func (r *Recorder) Add(rec ExecRecord)

    // Runner executes external commands with logging.
    type Runner struct {
        Logger   *log.Logger
        DryRun   bool
        Recorder *Recorder
    }

    // Run executes the command, logs it, and records the result.
    // Returns stdout as string. Stderr is forwarded to os.Stderr.
    func (r *Runner) Run(name string, args ...string) (string, error)

    // RunInteractive executes the command with stdin/stdout/stderr attached.
    // Used for shell, exec, and editor launch.
    func (r *Runner) RunInteractive(name string, args ...string) error

    // ResolveCommand returns env var value or fallback default.
    func ResolveCommand(envKey, defaultCmd string) string
    ```
*   **Logic**:
    -   `Run` / `RunInteractive`: 実行前に `fmt.Fprintf(os.Stdout, "[CMD] %s %s\n", name, strings.Join(args, " "))` を出力
    -   `dryRun` 時は `[DRY-RUN]` プレフィックスで出力し、実行せず即 return
    -   実行後に `ExecRecord` を `Recorder` に追加
    -   実行結果（exit code、成功/失敗）をログ出力

### 実行レポート (`internal/report/`)

#### [NEW] [report_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/report/report_test.go)
*   **Description**: レポートのユニットテスト（TDD 先行）
*   **テストケース**:
    -   `TestReport_Print`: stdout 出力に Feature, Branch, Steps セクションが含まれること
    -   `TestReport_EnvVars`: 環境変数テーブルが正しく出力されること（設定済み/未設定）
    -   `TestReport_WriteMarkdown`: ファイル出力が Markdown 形式であること
    -   `TestReport_EmptyRecords`: 実行記録 0 件でもクラッシュしないこと

#### [NEW] [report.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/report/report.go)
*   **Description**: 実行レポート生成
*   **Technical Design**:
    ```go
    // EnvVar represents a resolved environment variable.
    type EnvVar struct {
        Name     string
        Value    string // actual value or ""
        Default  string // fallback default
        WasSet   bool   // true if env var was explicitly set
    }

    // Report aggregates execution context and results.
    type Report struct {
        StartTime     time.Time
        Feature       string
        Branch        string
        OS            string
        Editor        string
        ContainerMode string
        EnvVars       []EnvVar
        Steps         []StepEntry
        OverallResult string // "SUCCESS" or "FAILED"
    }

    // StepEntry describes one step in the execution.
    type StepEntry struct {
        Name    string       // e.g. "Worktree creation"
        Record  *cmdexec.ExecRecord // nil if no command was executed
        Success bool
    }

    // Print writes the report to the given writer (stdout).
    func (r *Report) Print(w io.Writer)

    // WriteMarkdown writes the report as a Markdown file.
    func (r *Report) WriteMarkdown(path string) error
    ```

*   **Logic**:
    -   `Print`: stdout にセクション区切りで出力（Date, Feature, Environment Variables テーブル, Detected Environment, Steps, Result）
    -   `WriteMarkdown`: Markdown 形式でファイル書き出し。`Print` の出力をバッファして書き込む

### サブコマンド再構成 (`cmd/`)

#### [MODIFY] [root.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/root.go)
*   **Description**: 全フラグを削除し、グローバルフラグ + サブコマンド登録のみにする
*   **Technical Design**:
    -   `rootCmd` の `Args` を `cobra.NoArgs` に変更（サブコマンド必須）
    -   `RunE` を削除（root 直接実行不可）
    -   グローバルの `PersistentFlags`: `--verbose`, `--dry-run`, `--report <file>`
    -   `cmdexec.Recorder` と `report.Report` を `rootCmd.PersistentPreRunE` で初期化
    -   `rootCmd.PersistentPostRunE` でレポートを出力
    -   各サブコマンドを `init()` で `rootCmd.AddCommand(...)` する

#### [NEW] [common.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/common.go)
*   **Description**: サブコマンド間で共有する初期化ロジック
*   **Technical Design**:
    ```go
    // AppContext holds shared state for all subcommands.
    type AppContext struct {
        Logger        *log.Logger
        CmdRunner     *cmdexec.Runner
        Recorder      *cmdexec.Recorder
        Report        *report.Report
        RepoRoot      string
        Feature       string
        Branch        string
        DryRun        bool
        Verbose       bool
        ReportFile    string
    }

    // ParseFeatureBranch extracts feature and optional branch from args.
    // If branch is omitted, defaults to feature name.
    func ParseFeatureBranch(args []string) (feature, branch string)

    // InitContext builds AppContext from global flags and args.
    func InitContext(cmd *cobra.Command, args []string) (*AppContext, error)

    // ResolveEnvironment loads config, resolves editor, detects OS.
    func (ctx *AppContext) ResolveEnvironment(editorFlag string) (
        detect.OS, detect.Editor, matrix.ContainerMode, error)

    // CollectEnvVars gathers all DEVCTL_* env vars for the report.
    func CollectEnvVars() []report.EnvVar
    ```
*   **Logic**:
    -   `ParseFeatureBranch`: args[0] = feature, args[1] = branch（省略時は feature）
    -   `InitContext`: Logger, Recorder, CmdRunner の初期化。repoRoot="."
    -   `CollectEnvVars`: 全 `DEVCTL_*` 環境変数を読み取り、設定値/デフォルト値を記録

#### [NEW] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/up.go)
*   **Description**: `up` サブコマンド
*   **フラグ**: `--editor`, `--ssh`, `--rebuild`, `--no-build`
*   **Logic**:
    1.  `InitContext` → `ResolveEnvironment`
    2.  worktree パス解決（Part 4 で worktree 自動作成を追加予定。現段階では既存パス検索）
    3.  `plan.Build(...)` で実行計画を構築
    4.  `action.Runner.Up(...)` でコンテナ起動
    5.  `--editor` が指定されている場合は `editor.NewLauncher` → `action.Runner.Open(...)` でエディタ起動
    6.  レポートにステップを記録

#### [NEW] [down.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/down.go)
*   **Description**: `down` サブコマンド
*   **Logic**: `InitContext` → コンテナ停止・削除

#### [NEW] [open.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/open.go)
*   **Description**: `open` サブコマンド
*   **フラグ**: `--editor`, `--attach`
*   **Logic**:
    -   `--attach` あり: DevContainer attach を試行（フォールバックあり）
    -   `--attach` なし: ローカル worktree を直接開く

#### [NEW] [status.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/status.go)
*   **Description**: `status` サブコマンド
*   **Logic**: `InitContext` → `action.Runner.PrintStatus(...)`

#### [NEW] [shell.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/shell.go)
*   **Description**: `shell` サブコマンド
*   **Logic**: `InitContext` → `action.Runner.Shell(...)`

#### [NEW] [exec_cmd.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/exec_cmd.go)
*   **Description**: `exec` サブコマンド（ファイル名は Go 標準の `exec` パッケージとの衝突を避ける）
*   **Logic**: `InitContext` → `action.Runner.Exec(containerName, args)`. `--` 以降を実行コマンドとして扱う

### 既存パッケージの修正

#### [MODIFY] [runner.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/runner.go)
*   **Description**: `DockerRun`/`DockerRunOutput` を `cmdexec.Runner` に委譲する
*   **Logic**:
    -   `Runner` 構造体に `CmdRunner *cmdexec.Runner` フィールドを追加
    -   `DockerRun(args)` → `r.CmdRunner.RunInteractive("docker", args...)`
    -   `DockerRunOutput(args)` → `r.CmdRunner.Run("docker", args...)`
    -   `Logger` と `DryRun` は `CmdRunner` から参照可能なので、`Runner` 自体からは削除可能（互換性を考慮して段階的に移行）

#### [MODIFY] [editor.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/editor.go)
*   **Description**: `ResolveCommand` 関数を `cmdexec.ResolveCommand` に委譲し、`LaunchOptions` に `CmdRunner` を追加
*   **Logic**:
    -   `LaunchOptions.CmdRunner *cmdexec.Runner` を追加
    -   各 Launcher の `exec.Command` 呼び出しを `opts.CmdRunner.RunInteractive(...)` に置き換え
    -   `editor.ResolveCommand` は `cmdexec.ResolveCommand` のラッパーまたは直接呼び出し

#### [MODIFY] [planner.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/plan/planner.go)
*   **Description**: `Open` フィールドを削除し、`--editor` 指定の有無でエディタ起動を判定する変更
*   **Logic**:
    -   `Input.Open bool` を削除
    -   `Input.Editor detect.Editor` を追加（空文字＝エディタ指定なし）
    -   `up` コマンドでは `Editor != ""` → `ShouldOpenEditor=true`
    -   `open` コマンド（外部からの呼び出し）は常に `ShouldOpenEditor=true`
    -   `Input.Attach bool` を追加（`open --attach` 用）
    -   `Attach=true` かつ Capability が `CanTryDevcontainerAttach` の場合のみ `TryDevcontainerAttach=true`

## Step-by-Step Implementation Guide

### Phase 1: cmdexec パッケージ (TDD)

- [x] 1. `internal/cmdexec/cmdexec_test.go` を作成（テストケース 8 件）
- [x] 2. `internal/cmdexec/cmdexec.go` を実装（ExecRecord, Recorder, Runner, ResolveCommand）
- [x] 3. テスト実行 → 全 PASS 確認

### Phase 2: report パッケージ (TDD)

- [x] 4. `internal/report/report_test.go` を作成（テストケース 4 件）
- [x] 5. `internal/report/report.go` を実装（Report, EnvVar, StepEntry, Print, WriteMarkdown）
- [x] 6. テスト実行 → 全 PASS 確認

### Phase 3: 既存パッケージの cmdexec 移行

- [x] 7. `internal/action/runner.go` を修正 → `CmdRunner` 委譲
- [x] 8. `internal/editor/*.go` を修正 → `CmdRunner` 経由に変更
- [x] 9. `internal/editor/editor.go` の `ResolveCommand` を `cmdexec.ResolveCommand` に移行
- [x] 10. `internal/plan/planner.go` を修正 → `Open` 削除、`Editor`/`Attach` 追加
- [x] 11. 既存テスト修正 → 全 PASS 確認

### Phase 4: サブコマンド再構成

- [x] 12. `cmd/common.go` を作成（AppContext, ParseFeatureBranch, InitContext, CollectEnvVars）
- [x] 13. `cmd/root.go` を書き換え（グローバルフラグのみ、サブコマンド登録、PersistentPre/PostRun）
- [x] 14. `cmd/up.go` を作成
- [x] 15. `cmd/down.go` を作成
- [x] 16. `cmd/open.go` を作成（`--attach` フラグ含む）
- [x] 17. `cmd/status.go` を作成
- [x] 18. `cmd/shell.go` を作成
- [x] 19. `cmd/exec_cmd.go` を作成
- [x] 20. ビルド・全テスト → 全 PASS 確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **手動動作確認（dry-run）**:
    ```bash
    # サブコマンドが認識されること
    ./bin/devctl --help
    ./bin/devctl up --help

    # dry-run + verbose でログ確認
    ./bin/devctl up devctl test-001 --editor cursor --dry-run --verbose

    # レポート出力
    ./bin/devctl up devctl test-001 --dry-run --report tmp/test-report.md
    ```

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/escape/features/devctl/README.md)
*   **更新内容**: CLI 使用法をサブコマンド体系に更新。`--attach` フラグの説明追加。レポート機能の説明追加。

## 継続計画について

本計画（Part 3）はインフラ層とサブコマンド再構成を対象とする。
以下の要件は **Part 4** として別ファイルで計画する:

- R2: Worktree 自動作成（`internal/worktree/`）
- R3: 状態ファイル管理（`internal/state/`）
- R4: PR 作成（`cmd/pr.go`, `internal/action/pr.go`）
- R5: クローズ（`cmd/close.go`, `internal/action/close.go`）
- R6: worktree パス変更対応（`internal/resolve/worktree.go`）
- R8: ブランチ一覧（`cmd/list.go`）
