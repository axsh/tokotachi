# 002-Doctor-Remove-FeatureYaml-Check

> **Source Specification**: `prompts/phases/000-foundation/ideas/test-006/002-Doctor-Remove-FeatureYaml-Check.md`

## Goal Description

`checkFeature` 関数から `feature.yaml` チェックブロックを完全に削除し、doctor 出力から `feature.yaml` 行を消す。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| feature.yaml チェックブロックの完全削除 | Proposed Changes > checks.go |
| 出力に feature.yaml 行が含まれないこと | Verification Plan |

## Proposed Changes

### internal/doctor パッケージ

#### [MODIFY] [checks_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks_test.go)
*   **Description**: feature.yaml 関連テストケースを削除（TDD: テスト先行）
*   **Logic**:
    1. `TestCheckFeature/feature.yaml_missing_is_warn` テストケースを削除
    2. `TestCheckFeature/valid_feature_with_all_files` から `feature.yaml` 作成行を削除（devcontainer.json のみ）

#### [MODIFY] [checks.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/checks.go)
*   **Description**: `checkFeature` 内の feature.yaml チェックブロック全体を削除
*   **Logic**:
    1. 279行目〜314行目の `// feature.yaml` セクション（`featureYAML` 変数宣言から `StatusPass` 結果追加まで）を完全に削除
    2. 不要になった `"gopkg.in/yaml.v3"` import があれば削除（devcontainer.json は json パッケージを使用しているため yaml は不要になる可能性）

#### [MODIFY] [doctor_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/test-006/features/devctl/internal/doctor/doctor_test.go)
*   **Description**: `TestRun_AllPass` から `feature.yaml` 作成行を削除
*   **Logic**:
    1. `os.WriteFile(filepath.Join(root, "features", "myfeature", "feature.yaml", ...)` 行を削除

## Step-by-Step Implementation Guide

1.  **単体テスト修正**:
    *   `checks_test.go`: `feature.yaml_missing_is_warn` テストケースを削除。`valid_feature_with_all_files` から feature.yaml 作成行を削除。
    *   `doctor_test.go`: `TestRun_AllPass` から feature.yaml 作成行を削除。

2.  **プロダクションコード修正**:
    *   `checks.go`: `checkFeature` 内の feature.yaml チェックブロック全体を削除。不要 import を整理。

3.  **ビルドと単体テスト**:
    *   `./scripts/process/build.sh`

4.  **統合テスト**:
    *   `./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlDoctor"
    ```
    *   **Log Verification**: `devctl doctor` 出力に `feature.yaml` が含まれないことを確認

## Documentation

なし（仕様書 001 は前回の修正で更新済み。002 は 001 の延長であり追加更新不要）。
