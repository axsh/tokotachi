# 001-Doctor-FeatureCheck-Fix

> **Source Specification**: `prompts/phases/000-foundation/ideas/test-006/001-Doctor-FeatureCheck-Fix.md`

## Goal Description

`devctl doctor` のフィーチャーチェック項目を修正する。`feature.yaml` 不存在を FAIL→WARN に変更し、`go.mod` チェックを完全に削除する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| feature.yaml チェックを WARN に変更 | Proposed Changes > checks.go, checks_test.go |
| go.mod チェックを削除 | Proposed Changes > checks.go, checks_test.go |
| 統合テストを引数なしに修正 | Proposed Changes > devctl_doctor_test.go |

## Proposed Changes

### internal/doctor パッケージ

#### [MODIFY] [checks_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks_test.go)
*   **Description**: テストケースの更新（TDD: テスト先行）
*   **Logic**:
    1. `TestCheckFeature/feature.yaml_missing_is_fail` を `TestCheckFeature/feature.yaml_missing_is_warn` にリネーム。アサーションを `StatusFail` → `StatusWarn` に変更。
    2. `TestCheckFeature/go.mod_missing_is_info_only` テストケースを削除。
    3. `TestCheckFeature/valid_feature_with_all_files` から `go.mod` ファイル作成行を削除。テスト内で `go.mod` 存在チェックの Pass アサーションも削除。

#### [MODIFY] [checks.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks.go)
*   **Description**: `checkFeature` 関数のロジック修正
*   **Logic**:
    1. `feature.yaml` 不存在時: `StatusFail` → `StatusWarn` に変更。メッセージを `"not found (used for packaging, not required for devctl)"` に変更。FixHint も `"Create features/<name>/feature.yaml (required for packaging/release)"` に更新。
    2. `go.mod` チェックブロック（`// go.mod (informational)` 以降の if/else ブロック）を完全に削除。

---

### 統合テスト

#### [MODIFY] [devctl_doctor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/tests/integration-test/devctl_doctor_test.go)
*   **Description**: `TestDevctlDoctorBasic` を引数なし実行に戻す
*   **Logic**:
    1. `runDevctl(t, "doctor", "--feature", "devctl")` → `runDevctl(t, "doctor")` に変更。
    2. アサーションメッセージを `"devctl doctor should exit 0"` に戻す。
    3. `Feature: devctl` の含有チェックは残す（全フィーチャーが出力されるため含まれる）。

## Step-by-Step Implementation Guide

1.  **単体テスト修正**:
    *   `checks_test.go` を編集:
        - `feature.yaml_missing_is_fail` → `feature.yaml_missing_is_warn` にリネーム、`StatusFail` → `StatusWarn`
        - `go.mod_missing_is_info_only` テストケース削除
        - `valid_feature_with_all_files` から go.mod 関連行を削除

2.  **プロダクションコード修正**:
    *   `checks.go` の `checkFeature` を編集:
        - feature.yaml 不存在時の Status を `StatusWarn` に変更、メッセージ更新
        - go.mod チェックブロックを削除

3.  **ビルドと単体テスト**:
    *   `./scripts/process/build.sh` 実行

4.  **統合テスト修正**:
    *   `devctl_doctor_test.go` の `TestDevctlDoctorBasic` を引数なしに修正

5.  **統合テスト実行**:
    *   `./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `TestCheckFeature/feature.yaml_missing_is_warn` が PASS すること
    *   go.mod 関連テストが存在しないこと

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"
    ```
    *   `TestDevctlDoctorBasic` が引数なし実行で exitCode 0 で PASS すること

## Documentation

#### [MODIFY] [000-Doctor-Subcommand.md](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/prompts/phases/000-foundation/ideas/test-006/000-Doctor-Subcommand.md)
*   **更新内容**: フィーチャーチェックのテーブルで `feature.yaml` のステータスを FAIL→WARN に変更し、`go.mod` 行を削除。
