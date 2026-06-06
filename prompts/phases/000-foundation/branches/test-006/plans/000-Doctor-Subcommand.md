# 000-Doctor-Subcommand

> **Source Specification**: `prompts/phases/000-foundation/ideas/test-006/000-Doctor-Subcommand.md`

## Goal Description

`devctl doctor` サブコマンドを新設し、リポジトリの構成・設定ファイル・外部ツールの健全性を一括チェックする機能を実装する。チェック結果は ✅/❌/⚠️ 形式の一覧、JSON出力、終了コード制御に対応する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 新しいサブコマンド `devctl doctor` | Proposed Changes > cmd/doctor.go, cmd/root.go |
| 外部ツール依存チェック (git, docker, gh) | Proposed Changes > internal/doctor/checks.go (`checkExternalTools`) |
| リポジトリ構造チェック | Proposed Changes > internal/doctor/checks.go (`checkRepoStructure`) |
| グローバル設定チェック (.devrc.yaml) | Proposed Changes > internal/doctor/checks.go (`checkGlobalConfig`) |
| フィーチャーチェック (feature.yaml, devcontainer.json, go.mod) | Proposed Changes > internal/doctor/checks.go (`checkFeature`) |
| ✅ PASS / ❌ FAIL / ⚠️ WARN 一覧表示 | Proposed Changes > internal/doctor/result.go (`PrintText`) |
| FAILまたはWARNに修正方法を併記 | Proposed Changes > internal/doctor/result.go (CheckResult.FixHint) |
| 終了コード制御 (FAIL → 1, それ以外 → 0) | Proposed Changes > cmd/doctor.go |
| `--feature <name>` フラグ | Proposed Changes > cmd/doctor.go |
| `--json` フラグ | Proposed Changes > internal/doctor/result.go (`PrintJSON`) |
| テキスト出力フォーマット | Proposed Changes > internal/doctor/result.go |
| JSON出力フォーマット | Proposed Changes > internal/doctor/result.go |

## Proposed Changes

### internal/doctor パッケージ (新規)

#### [NEW] [result_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/result_test.go)
*   **Description**: `result.go` の型と出力フォーマットの単体テスト
*   **Technical Design**:
    ```go
    // テーブル駆動テストケース:
    // - TestCheckResult_StatusString: Pass/Fail/Warn → "✅"/"❌"/"⚠️"
    // - TestReport_HasFailures: Fail含む→true, Warn+Passのみ→false
    // - TestReport_PrintText: テキスト出力の内容とフォーマット検証
    // - TestReport_PrintJSON: JSON出力が valid JSON であること、構造を検証
    // - TestReport_Summary: total/passed/failed/warnings カウント検証
    ```

#### [NEW] [result.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/result.go)
*   **Description**: チェック結果の型定義と出力フォーマッター
*   **Technical Design**:
    ```go
    package doctor

    type Status int
    const (
        StatusPass Status = iota
        StatusFail
        StatusWarn
    )
    // String() returns "✅", "❌", "⚠️"

    type CheckResult struct {
        Category string `json:"category"`
        Name     string `json:"name"`
        Status   Status `json:"status"`
        Message  string `json:"message"`
        Expected string `json:"expected,omitempty"` // Expected state description
        FixHint  string `json:"fix_hint,omitempty"` // How to fix
    }

    type Summary struct {
        Total    int `json:"total"`
        Passed   int `json:"passed"`
        Failed   int `json:"failed"`
        Warnings int `json:"warnings"`
    }

    type Report struct {
        Results []CheckResult `json:"results"`
    }

    // HasFailures returns true if any result has StatusFail.
    func (r *Report) HasFailures() bool

    // Summary returns aggregated counts.
    func (r *Report) Summary() Summary

    // PrintText writes human-readable output with icons and categories.
    // Groups results by Category, shows FixHint for non-Pass items.
    func (r *Report) PrintText(w io.Writer)

    // PrintJSON writes JSON output to w.
    // Output structure: {"results": [...], "summary": {...}}
    func (r *Report) PrintJSON(w io.Writer) error
    ```
*   **Logic**:
    *   `PrintText`: カテゴリ別にグルーピングし、アイコン付きで表示。FAIL/WARN には `Expected` と `FixHint` をインデントして表示。最終行にサマリ。
    *   `PrintJSON`: `json.MarshalIndent` で整形出力。Status は `MarshalJSON` で `"pass"`, `"fail"`, `"warn"` の文字列に変換。

---

#### [NEW] [checks_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks_test.go)
*   **Description**: 各チェック関数の単体テスト
*   **Technical Design**:
    ```go
    // ToolChecker インターフェースをモック化してテスト
    // テーブル駆動テスト:
    // - TestCheckExternalTools: コマンド成功→Pass, 失敗→Fail/Warn(gh)
    // - TestCheckRepoStructure: ディレクトリ存在/不存在時のStatus
    // - TestCheckGlobalConfig:
    //   - ファイルなし → Warn
    //   - 正常YAML → Pass
    //   - 不正YAML → Fail
    //   - project_name空 → Warn
    //   - 不正なeditor値 → Warn
    //   - 不正なcontainer_mode値 → Warn
    // - TestCheckFeature:
    //   - feature.yaml存在+正常 → Pass
    //   - feature.yaml不存在 → Fail
    //   - feature.yaml不正YAML → Fail
    //   - devcontainer.json存在+正常 → Pass
    //   - devcontainer.json不存在 → Warn
    //   - devcontainer.json不正JSON → Fail
    //   - go.mod存在 → Pass
    //   - go.mod不存在 (Go言語以外) → Skip (チェックしない)
    ```
*   **Logic**:
    *   `testing.TempDir()` で一時ディレクトリを構築し、各テストケースの構造を準備。
    *   外部コマンドの実行は `ToolChecker` インターフェースでモックに差し替え。

#### [NEW] [checks.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks.go)
*   **Description**: 各チェック項目の実装
*   **Technical Design**:
    ```go
    package doctor

    // ToolChecker abstracts external command execution for testability.
    type ToolChecker interface {
        // CheckTool runs "<tool> --version" and returns output or error.
        CheckTool(name string) (version string, err error)
    }

    // DefaultToolChecker runs actual commands via exec.Command.
    type DefaultToolChecker struct{}
    func (d *DefaultToolChecker) CheckTool(name string) (string, error)
    // exec.Command(name, "--version").Output() を実行し、
    // 1行目を version として返す。

    // checkExternalTools checks git, docker, gh availability.
    func checkExternalTools(checker ToolChecker) []CheckResult
    // - git: Fail if not found
    // - docker: Fail if not found
    // - gh: Warn if not found (only for 'devctl pr')
    //   FixHint: "Install GitHub CLI: https://cli.github.com/"

    // checkRepoStructure checks required directories exist.
    func checkRepoStructure(repoRoot string) []CheckResult
    // - Git repo check: git rev-parse --show-toplevel (ToolChecker使用)
    //   Fail if not a git repo. FixHint: "Run 'git init' to initialize."
    // - features/ directory: Fail if missing.
    //   FixHint: "Create 'features/' directory."
    // - work/ directory: Warn if missing (may not exist before first worktree).
    //   Message: "Not found (created on first 'devctl up')"
    // - scripts/ directory: Warn if missing.

    // checkGlobalConfig checks .devrc.yaml existence and validity.
    func checkGlobalConfig(repoRoot string) []CheckResult
    // - File existence: Warn if missing.
    //   FixHint: "Create .devrc.yaml with: project_name, default_editor, default_container_mode"
    // - YAML parse: Fail if invalid.
    //   FixHint: "Fix YAML syntax in .devrc.yaml"
    // - project_name: Warn if empty.
    //   Message: "Using default 'devctl'"
    // - default_editor: Warn if not in {code, cursor, ag, claude}.
    // - default_container_mode: Warn if not in {none, devcontainer, docker-local, docker-ssh}.

    // Supported values for validation.
    var validEditors = []string{"code", "vscode", "cursor", "ag", "antigravity", "claude"}
    var validContainerModes = []string{"none", "devcontainer", "docker-local", "docker-ssh"}

    // checkFeature checks a single feature directory.
    func checkFeature(repoRoot, featureName string) []CheckResult
    // - feature.yaml existence: Fail if missing.
    //   FixHint: "Create features/<name>/feature.yaml"
    // - feature.yaml parse: Fail if invalid YAML.
    // - .devcontainer/devcontainer.json existence: Warn if missing.
    // - .devcontainer/devcontainer.json parse: Fail if exists but invalid JSON.
    // - go.mod existence: Pass if exists. (Info only, not Fail if missing.)

    // discoverFeatures lists all subdirectories under features/.
    func discoverFeatures(repoRoot string) ([]string, error)
    // os.ReadDir(filepath.Join(repoRoot, "features")) → filter directories
    ```

---

#### [NEW] [doctor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/doctor_test.go)
*   **Description**: チェック実行エンジンの単体テスト
*   **Technical Design**:
    ```go
    // テーブル駆動テスト:
    // - TestRun_AllPass: 正常環境 → HasFailures()=false
    // - TestRun_WithFailure: 不正設定 → HasFailures()=true
    // - TestRun_FeatureFilter: --feature指定 → 指定フィーチャーのみチェック
    ```

#### [NEW] [doctor.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/doctor.go)
*   **Description**: チェック実行エンジン
*   **Technical Design**:
    ```go
    package doctor

    // Options holds configuration for the doctor run.
    type Options struct {
        RepoRoot      string
        FeatureFilter string // empty = all features
        ToolChecker   ToolChecker
    }

    // Run executes all checks and returns a Report.
    func Run(opts Options) (*Report, error)
    ```
*   **Logic**:
    1. `checkExternalTools(opts.ToolChecker)` を実行し結果を追加
    2. `checkRepoStructure(opts.RepoRoot)` を実行し結果を追加
    3. `checkGlobalConfig(opts.RepoRoot)` を実行し結果を追加
    4. `FeatureFilter` が空なら `discoverFeatures` で全フィーチャーを列挙。指定ありならそのフィーチャーのみ。
    5. 各フィーチャーに対し `checkFeature(opts.RepoRoot, name)` を実行し結果を追加
    6. 集約した `Report` を返す

---

### cmd パッケージ

#### [NEW] [doctor.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/cmd/doctor.go)
*   **Description**: cobra サブコマンド定義
*   **Technical Design**:
    ```go
    package cmd

    var (
        doctorFlagFeature string
        doctorFlagJSON    bool
    )

    var doctorCmd = &cobra.Command{
        Use:   "doctor",
        Short: "Check repository health and configuration",
        Long:  "Diagnose the repository structure, configuration files, and external tool availability.",
        Args:  cobra.NoArgs,
        RunE:  runDoctor,
    }

    func init() {
        doctorCmd.Flags().StringVar(&doctorFlagFeature, "feature", "", "Check only the specified feature")
        doctorCmd.Flags().BoolVar(&doctorFlagJSON, "json", false, "Output results in JSON format")
    }

    func runDoctor(cmd *cobra.Command, args []string) error
    ```
*   **Logic**:
    1. `repoRoot` を取得 (`os.Getwd()`)
    2. `doctor.Options{RepoRoot: repoRoot, FeatureFilter: doctorFlagFeature, ToolChecker: &doctor.DefaultToolChecker{}}` を構築
    3. `doctor.Run(opts)` を実行
    4. `doctorFlagJSON` なら `report.PrintJSON(os.Stdout)`、そうでなければ `report.PrintText(os.Stdout)`
    5. `report.HasFailures()` なら `os.Exit(1)`

#### [MODIFY] [root.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/cmd/root.go)
*   **Description**: `doctorCmd` を登録
*   **Technical Design**:
    ```diff
     rootCmd.AddCommand(listCmd)
    +rootCmd.AddCommand(doctorCmd)
    ```

---

### 統合テスト

#### [NEW] [devctl_doctor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/tests/integration-test/devctl_doctor_test.go)
*   **Description**: `devctl doctor` の end-to-end テスト
*   **Technical Design**:
    ```go
    package integration_test

    // TestDevctlDoctorBasic: devctl doctor を実行し、終了コード 0 を確認。
    //   - runDevctl(t, "doctor") → exitCode == 0
    //   - stdout に "External Tools" カテゴリが含まれること
    //   - stdout に "Repository Structure" カテゴリが含まれること

    // TestDevctlDoctorJSON: devctl doctor --json を実行。
    //   - runDevctl(t, "doctor", "--json") → exitCode == 0
    //   - stdout を json.Unmarshal して valid JSON であることを確認
    //   - "results" 配列と "summary" オブジェクトが存在すること

    // TestDevctlDoctorFeatureFilter: devctl doctor --feature devctl を実行。
    //   - runDevctl(t, "doctor", "--feature", "devctl") → exitCode == 0
    //   - stdout に "Feature: devctl" が含まれること
    //   - stdout に他のフィーチャー名が含まれないこと (integration-test 等)
    ```

## Step-by-Step Implementation Guide

1.  **result.go のテストと実装**:
    *   `internal/doctor/result_test.go` を作成し、Status文字列変換・HasFailures・Summary・PrintText・PrintJSON のテストケースを記述。
    *   `internal/doctor/result.go` を作成し、テストを通す。

2.  **checks.go のテストと実装**:
    *   `internal/doctor/checks_test.go` を作成し、ToolChecker モック、一時ディレクトリベースの各チェック関数テストを記述。
    *   `internal/doctor/checks.go` を作成し、テストを通す。

3.  **doctor.go (エンジン) のテストと実装**:
    *   `internal/doctor/doctor_test.go` を作成し、Run関数の結合テストを記述。
    *   `internal/doctor/doctor.go` を作成し、テストを通す。

4.  **cmd/doctor.go の作成と root.go の修正**:
    *   `cmd/doctor.go` を作成（cobra コマンド定義）。
    *   `cmd/root.go` に `rootCmd.AddCommand(doctorCmd)` を追加。

5.  **ビルドと単体テスト実行**:
    *   `./scripts/process/build.sh` を実行し、コンパイルと全単体テストの成功を確認。

6.  **統合テストの作成と実行**:
    *   `tests/integration-test/devctl_doctor_test.go` を作成。
    *   `./scripts/process/integration_test.sh --categories "devctl" --specify "TestDevctlDoctor"` を実行し、成功を確認。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   新規パッケージ `internal/doctor` の全テストが PASS すること。
    *   既存テスト（14ファイル）にリグレッションがないこと。

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "TestDevctlDoctor"
    ```
    *   `TestDevctlDoctorBasic`: 終了コード 0、出力にカテゴリ名が含まれること。
    *   `TestDevctlDoctorJSON`: 有効な JSON 出力であること。
    *   `TestDevctlDoctorFeatureFilter`: 指定フィーチャーのみの出力であること。

## Documentation

本計画で影響を受ける既存ドキュメントはありません。フィーチャーの README に `doctor` コマンドの説明を追加することは、将来のドキュメント整備タスクとして別途対応します。
