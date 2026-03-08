# 000-GlobalEnvOption

> **Source Specification**: `prompts/phases/000-foundation/ideas/fix-env-option/000-GlobalEnvOption.md`

## Goal Description

`--env` フラグをルートコマンドの `PersistentFlags` に移動し、レポートを出力するすべてのコマンドで環境変数セクションの表示を制御できるようにする。デフォルトでは環境変数セクションを非表示にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `--env` をルートコマンドの PersistentFlags に移動 | Proposed Changes > `cmd/root.go` |
| デフォルトで環境変数セクションを非表示 | Proposed Changes > `internal/report/report.go` (`ShowEnvVars` フィールド) |
| すべてのレポート出力コマンドで `--env` が動作 | Proposed Changes > `cmd/common.go` (`InitContext` に `flagEnv` 反映) |
| `list` コマンドのローカルフラグ削除 | Proposed Changes > `cmd/list.go` |

## Proposed Changes

### report パッケージ

#### [MODIFY] [report_test.go](file://features/devctl/internal/report/report_test.go)
*   **Description**: `ShowEnvVars` フィールドによる表示制御のテストを追加
*   **Technical Design**:
    *   既存の `TestReport_EnvVars` は `ShowEnvVars=true` を前提とするよう更新
    *   新規テスト `TestReport_EnvVars_Hidden` を追加
*   **Logic**:
    *   `TestReport_EnvVars`: `sampleReport()` に `ShowEnvVars: true` を追加し、`EnvVars` セクションが出力されることを確認（既存のアサーション維持）
    *   `TestReport_EnvVars_Hidden`: `ShowEnvVars: false`（デフォルトのゼロ値）の場合、`EnvVars` が設定されていても `"Environment Variables"` が出力に含まれないことを確認

---

#### [MODIFY] [report.go](file://features/devctl/internal/report/report.go)
*   **Description**: `Report` 構造体に `ShowEnvVars` フィールドを追加し、`Print` メソッドの出力条件を変更
*   **Technical Design**:
    ```go
    type Report struct {
        // ...既存フィールド...
        EnvVars       []EnvVar
        ShowEnvVars   bool        // true の場合のみ環境変数セクションを表示
        Steps         []StepEntry
        OverallResult string
    }
    ```
*   **Logic**:
    *   `Print` メソッド内の環境変数セクション出力条件を変更:
        *   変更前: `if len(r.EnvVars) > 0 {`
        *   変更後: `if r.ShowEnvVars && len(r.EnvVars) > 0 {`
    *   これにより `ShowEnvVars` がデフォルト (`false`) の場合、環境変数セクションは表示されなくなる

---

### cmd パッケージ

#### [MODIFY] [common_test.go](file://features/devctl/cmd/common_test.go)
*   **Description**: `InitContext` で `flagEnv` が `Report.ShowEnvVars` に反映されるテストを追加
*   **Technical Design**:
    *   新規テスト `TestInitContext_ShowEnvVarsReflectsFlagEnv` を追加
*   **Logic**:
    *   `flagEnv = false` の状態で `InitContext` を実行し、`ctx.Report.ShowEnvVars` が `false` であることを確認
    *   `flagEnv = true` に設定して `InitContext` を実行し、`ctx.Report.ShowEnvVars` が `true` であることを確認
    *   テスト後に `flagEnv` を元に戻す（`defer` で復元）

---

#### [MODIFY] [root.go](file://features/devctl/cmd/root.go)
*   **Description**: `--env` フラグをルートコマンドの `PersistentFlags` に追加
*   **Technical Design**:
    ```go
    var (
        flagVerbose bool
        flagDryRun  bool
        flagReport  string
        flagEnv     bool   // 追加
    )

    func init() {
        // ...既存のフラグ...
        rootCmd.PersistentFlags().BoolVar(&flagEnv, "env", false, "Show environment variables in report")
    }
    ```

---

#### [MODIFY] [common.go](file://features/devctl/cmd/common.go)
*   **Description**: `InitContext` で `flagEnv` を `Report.ShowEnvVars` に反映
*   **Technical Design**:
    ```go
    ctx.Report = &report.Report{
        Feature:     feature,
        Branch:      branch,
        EnvVars:     CollectEnvVars(),
        ShowEnvVars: flagEnv,  // グローバルフラグを反映
    }
    ```

---

#### [MODIFY] [list.go](file://features/devctl/cmd/list.go)
*   **Description**: ローカルの `flagListEnv` を削除し、グローバルの `flagEnv` を使用
*   **Technical Design**:
    *   変数宣言 `flagListEnv bool` を削除
    *   `init()` 内の `listCmd.Flags().BoolVar(&flagListEnv, "env", ...)` を削除
    *   `runListBranches()` 内の参照を `flagListEnv` → `flagEnv` に変更

---

### 統合テスト

#### [MODIFY] [devctl_list_code_test.go](file://tests/integration-test/devctl_list_code_test.go)
*   **Description**: `list` コマンドで `--env` なしの場合に環境変数セクションが出ないことを確認するテストを追加
*   **Technical Design**:
    *   新規テスト `TestDevctlListCode_NoEnvByDefault` を追加
*   **Logic**:
    *   `devctl list` を実行し、stdout と stderr の両方に `"Environment Variables"` が含まれないことをアサート

#### [NEW] [devctl_env_option_test.go](file://tests/integration-test/devctl_env_option_test.go)
*   **Description**: `--env` フラグがレポート出力コマンド全般で動作することを確認する統合テスト
*   **Technical Design**:
    *   `TestDevctlOpen_NoEnvByDefault`: `devctl open --dry-run <branch>` を実行し、stdout に `"Environment Variables"` が含まれないことを確認
    *   `TestDevctlOpen_WithEnvFlag`: `devctl open --env --dry-run <branch>` を実行し、stdout に `"Environment Variables"` が含まれることを確認
*   **Logic**:
    *   `--dry-run` フラグを使用して実際のエディタ起動やコンテナ操作を回避
    *   `runDevctl` ヘルパーを使用
    *   ブランチ名には既存 worktree のブランチ（例: テスト実行中のカレントブランチ）を使用

## Step-by-Step Implementation Guide

- [x] 1. **`report_test.go` にテストを追加 (TDD: Red)**
    - `TestReport_EnvVars` を更新: `sampleReport()` で `ShowEnvVars: true` を設定
    - `TestReport_EnvVars_Hidden` を新規追加: `ShowEnvVars: false` で `"Environment Variables"` が出力に含まれないことを確認
    - この時点ではコンパイルエラーが発生する（`ShowEnvVars` フィールドが未定義のため）

- [x] 2. **`report.go` を修正 (TDD: Green)**
    - `Report` 構造体に `ShowEnvVars bool` フィールドを追加
    - `Print` メソッドの条件を `if r.ShowEnvVars && len(r.EnvVars) > 0 {` に変更

- [x] 3. **`common_test.go` にテストを追加 (TDD: Red)**
    - `TestInitContext_ShowEnvVarsReflectsFlagEnv` を追加

- [x] 4. **`root.go` に `flagEnv` を追加**
    - グローバル変数 `flagEnv bool` を宣言
    - `init()` で `rootCmd.PersistentFlags().BoolVar(&flagEnv, "env", false, "Show environment variables in report")` を追加

- [x] 5. **`common.go` を修正 (TDD: Green)**
    - `InitContext` 内の `Report` 初期化に `ShowEnvVars: flagEnv` を追加

- [x] 6. **`list.go` を修正**
    - `flagListEnv` の宣言を削除
    - `init()` 内の `--env` フラグ登録を削除
    - `runListBranches()` 内の `flagListEnv` を `flagEnv` に変更

- [x] 7. **ビルドと単体テスト**
    - `./scripts/process/build.sh` を実行し、全テストがパスすることを確認

- [x] 8. **統合テストを追加**
    - `devctl_list_code_test.go` に `TestDevctlListCode_NoEnvByDefault` を追加
    - `devctl_env_option_test.go` を新規作成し、`TestDevctlOpen_NoEnvByDefault` と `TestDevctlOpen_WithEnvFlag` を追加

- [x] 9. **統合テスト実行**
    - `./scripts/process/integration_test.sh --categories "integration-test"` を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "EnvOption|NoEnvByDefault"
    ```
    *   **Log Verification**:
        *   `TestDevctlOpen_NoEnvByDefault`: stdout に `"Environment Variables"` が含まれないこと
        *   `TestDevctlOpen_WithEnvFlag`: stdout に `"Environment Variables"` が含まれること
        *   `TestDevctlListCode_NoEnvByDefault`: stdout/stderr に `"Environment Variables"` が含まれないこと
        *   `TestDevctlListCode_EnvFlagAccepted`: stderr に `"Environment Variables"` が含まれること（既存テスト）

## Documentation

影響を受ける既存ドキュメントはありません。仕様書は作成済みです。
