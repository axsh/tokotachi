# 003-Doctor-Fix-Flag

> **Source Specification**: `prompts/phases/000-foundation/ideas/test-006/003-Doctor-Fix-Flag.md`

## Goal Description

`devctl doctor --fix` フラグを追加し、修正可能な項目（`.devrc.yaml` 生成、ディレクトリ作成）を自動修正する機能を実装する。修正した内容を出力に明記する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `--fix` フラグの追加 | cmd/doctor.go, doctor.go Options |
| .devrc.yaml の自動生成 | checks.go `fixGlobalConfig` |
| work/ scripts/ ディレクトリ作成 | checks.go `fixDirectory` |
| 修正後の表示（🔧 アイコン） | result.go `CheckResult.Fixed`, `PrintText` |
| JSON output に `fixed` フィールド | result.go `CheckResult.Fixed` |
| 修正不可能な項目は従来通り | checks.go（変更なし） |

## Proposed Changes

### result.go（データ構造）

#### [MODIFY] [result.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/result.go)
*   **Description**: `CheckResult` に `Fixed` フィールドを追加。`PrintText` で Fixed 項目に 🔧 アイコンを表示。`Summary` に `FixedCount` を追加。
*   **Technical Design**:
    ```go
    type CheckResult struct {
        Category string `json:"category"`
        Name     string `json:"name"`
        Status   Status `json:"status"`
        Message  string `json:"message"`
        Expected string `json:"expected,omitempty"`
        FixHint  string `json:"fix_hint,omitempty"`
        Fixed    bool   `json:"fixed,omitempty"`
    }

    type Summary struct {
        Total    int `json:"total"`
        Passed   int `json:"passed"`
        Failed   int `json:"failed"`
        Warnings int `json:"warnings"`
        Fixed    int `json:"fixed,omitempty"`
    }
    ```
*   **Logic (PrintText)**:
    - `res.Fixed == true` の場合、ステータスアイコンの代わりに `🔧` を表示
    - サマリー行に `N fixed` を追加（Fixed > 0 の場合のみ）

### checks.go（フィクサー関数）

#### [MODIFY] [checks.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks.go)
*   **Description**: 修正関数 `fixGlobalConfig`, `fixDirectory` を追加
*   **Technical Design**:
    ```go
    // fixGlobalConfig creates .devrc.yaml with default settings.
    func fixGlobalConfig(repoRoot string) error {
        path := filepath.Join(repoRoot, ".devrc.yaml")
        content := "project_name: \"\"\ndefault_editor: cursor\ndefault_container_mode: docker-local\n"
        return os.WriteFile(path, []byte(content), 0o644)
    }

    // fixDirectory creates a directory if it doesn't exist.
    func fixDirectory(repoRoot, dirName string) error {
        return os.MkdirAll(filepath.Join(repoRoot, dirName), 0o755)
    }
    ```

### doctor.go（エンジン）

#### [MODIFY] [doctor.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/doctor.go)
*   **Description**: `Options` に `Fix bool` フィールドを追加。`Run` 関数の最後で Fix が true の場合に修正を適用。
*   **Technical Design**:
    ```go
    type Options struct {
        RepoRoot      string
        FeatureFilter string
        ToolChecker   ToolChecker
        Fix           bool
    }
    ```
*   **Logic**:
    1. 通常通り全チェックを実行
    2. `opts.Fix == true` の場合、結果を走査して修正可能な項目を特定:
       - `.devrc.yaml` (Category == categoryConfig, Name == ".devrc.yaml", Status == StatusWarn) → `fixGlobalConfig` 実行
       - `work/`, `scripts/` (Category == categoryRepo, Status == StatusWarn) → `fixDirectory` 実行
    3. 修正成功: 該当結果の `Status` を `StatusPass`、`Fixed` を `true`、`Message` を修正内容に更新
    4. 修正失敗: `Status` を `StatusFail`、`Message` にエラーを追加

### cmd/doctor.go

#### [MODIFY] [cmd/doctor.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/cmd/doctor.go)
*   **Description**: `--fix` フラグを追加
*   **Logic**:
    ```go
    var doctorFlagFix bool
    // init() に追加:
    doctorCmd.Flags().BoolVar(&doctorFlagFix, "fix", false, "Auto-fix issues where possible")
    // runDoctor 内:
    opts.Fix = doctorFlagFix
    ```

## Step-by-Step Implementation Guide

1.  **テスト作成（result_test.go）**:
    *   `TestReport_PrintText_Fixed`: Fixed フラグ付き CheckResult のテキスト出力に 🔧 が含まれること
    *   `TestReport_PrintJSON_Fixed`: JSON 出力に `"fixed": true` が含まれること
    *   `TestSummary_Fixed`: Summary の Fixed カウントが正しいこと

2.  **result.go 修正**:
    *   `CheckResult` に `Fixed bool` フィールド追加
    *   `Summary` に `Fixed int` フィールド追加
    *   `PrintText` で Fixed アイコン表示ロジック追加
    *   `Summary()` で Fixed カウント追加

3.  **テスト作成（checks_test.go）**:
    *   `TestFixGlobalConfig`: TempDir に `.devrc.yaml` を生成し、内容を確認
    *   `TestFixDirectory`: TempDir にディレクトリを生成

4.  **checks.go 修正**:
    *   `fixGlobalConfig`, `fixDirectory` 関数を追加

5.  **テスト作成（doctor_test.go）**:
    *   `TestRun_WithFix`: Fix=true で .devrc.yaml 不存在の状態から実行、レポートに Fixed=true の結果が含まれること

6.  **doctor.go 修正**:
    *   `Options` に `Fix` フィールド追加
    *   `Run` に修正ロジック追加

7.  **cmd/doctor.go 修正**:
    *   `--fix` フラグ追加

8.  **ビルドと単体テスト**:
    *   `./scripts/process/build.sh`

9.  **統合テスト作成（devctl_doctor_test.go）**:
    *   `TestDevctlDoctorFix`: TempDir でリポジトリ構造を作り `devctl doctor --fix` を実行（→ 実リポジトリでやると .devrc.yaml が生成されてしまうため不可。代わにテキスト出力で `--fix` ヘルプ表示と `--help` 出力に `--fix` が含まれることを確認）

10. **統合テスト実行**:
    *   `./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `TestReport_PrintText_Fixed`, `TestReport_PrintJSON_Fixed`, `TestSummary_Fixed` が PASS
    *   `TestFixGlobalConfig`, `TestFixDirectory` が PASS
    *   `TestRun_WithFix` が PASS

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"
    ```
    *   既存の3テスト + 新規テストがすべて PASS

## Documentation

なし。
