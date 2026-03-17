# 018-Fix-PostActions-TemplateVar

> **Source Specification**: [016-Fix-PostActions-TemplateVar.md](file://prompts/phases/000-foundation/ideas/main/016-Fix-PostActions-TemplateVar.md)

## Goal Description

`scaffold apply` 実行時、`ApplyPostActions` に渡される `placement.BaseDir` のテンプレート変数（`{{feature_name}}`）が未展開のまま使用されるバグを修正する。`ApplyFiles` と同じパターンで、呼び出し元で `ProcessTemplatePath` を使って展開済みの `baseDir` を渡すようにする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.1: `applySingleScaffold` の修正 | Proposed Changes > `scaffold.go` (`applySingleScaffold`) |
| Req.2: `applyDependencyChain` の修正 | Proposed Changes > `scaffold.go` (`applyDependencyChain`) |
| Req.3: 既存の `ProcessTemplatePath` を再利用 | Proposed Changes > `scaffold.go` (両箇所) |
| Req.4: `ApplyFiles` と同じ展開結果 | テスト: `TestApplyFiles` と同一 `optionValues` での `baseDir` 展開確認 |
| Req.5: テスト追加 | Proposed Changes > `applier_test.go` |

## Proposed Changes

### scaffold パッケージ

#### [MODIFY] [applier_test.go](file://pkg/scaffold/applier_test.go)

> **テストファースト**: 実装修正の前にテストを追加する。

*   **Description**: テンプレート変数を含む `baseDir` での `applyFilePermissions` 動作確認テストを追加する。本バグ修正はアプローチ A（呼び出し側で展開）のため、`ApplyPostActions` 自体には展開済みパスが渡される前提。ここでは **呼び出し元が展開を怠った場合にエラーとなること** を確認するテストと、**展開済みパスが正しく動作すること** を確認するテストの2つを追加する。
*   **Technical Design**:
    *   `TestApplyFilePermissions_WithTemplateBaseDir_Unexpanded`: 未展開 `baseDir`（`features/{{feature_name}}`）を渡すと `filepath.WalkDir` がエラーとなることを確認
    *   `TestApplyFilePermissions_WithTemplateBaseDir_Expanded`: 展開済み `baseDir`（`features/myapp`）を渡すと正常に動作することを確認
*   **Logic**:
    1. `t.TempDir()` 内に `features/myapp/scripts/run.sh` を作成
    2. 未展開パス `features/{{feature_name}}` で `applyFilePermissions` を呼ぶ → エラーが返ること
    3. 展開済みパス `features/myapp` で `applyFilePermissions` を呼ぶ → エラーなし、パーミッション設定成功

---

#### [MODIFY] [scaffold.go](file://pkg/scaffold/scaffold.go)

*   **Description**: `applySingleScaffold` と `applyDependencyChain` の `ApplyPostActions` 呼び出し前に `baseDir` のテンプレート変数展開処理を追加する。
*   **Technical Design**:

    **修正箇所1: `applySingleScaffold` (276行目付近)**

    現在:
    ```go
    if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
        return fmt.Errorf("failed to apply post-actions: %w", err)
    }
    ```

    修正後:
    ```go
    // Resolve template variables in baseDir for post-actions (same as ApplyFiles)
    postActionsBaseDir := placement.BaseDir
    if optionValues != nil && strings.Contains(postActionsBaseDir, "{{") {
        var tmplErr error
        postActionsBaseDir, tmplErr = ProcessTemplatePath(postActionsBaseDir, optionValues)
        if tmplErr != nil {
            return fmt.Errorf("failed to process base_dir template for post-actions: %w", tmplErr)
        }
    }
    if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, postActionsBaseDir); err != nil {
        return fmt.Errorf("failed to apply post-actions: %w", err)
    }
    ```

    **修正箇所2: `applyDependencyChain` (339行目付近)**

    現在:
    ```go
    if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
        return fmt.Errorf("failed to apply post-actions for %s/%s: %w",
            category, name, err)
    }
    ```

    修正後:
    ```go
    // Resolve template variables in baseDir for post-actions
    postActionsBaseDir := placement.BaseDir
    if dp.OptionValues != nil && strings.Contains(postActionsBaseDir, "{{") {
        var tmplErr error
        postActionsBaseDir, tmplErr = ProcessTemplatePath(postActionsBaseDir, dp.OptionValues)
        if tmplErr != nil {
            return fmt.Errorf("failed to process base_dir template for post-actions %s/%s: %w",
                category, name, tmplErr)
        }
    }
    if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, postActionsBaseDir); err != nil {
        return fmt.Errorf("failed to apply post-actions for %s/%s: %w",
            category, name, err)
    }
    ```

*   **Logic**:
    - `ApplyFiles` 内の baseDir 展開処理（applier.go:54-60）と同一パターン
    - `optionValues`（または `dp.OptionValues`）が `nil` でない、かつ `baseDir` に `{{` が含まれる場合のみ展開
    - 展開失敗時は `fmt.Errorf` でラップしたエラーを返す

## Step-by-Step Implementation Guide

- [x] **Step 1: テスト追加 (TDD: Red)**
    *   `pkg/scaffold/applier_test.go` に `TestApplyFilePermissions_WithTemplateBaseDir_Unexpanded` と `TestApplyFilePermissions_WithTemplateBaseDir_Expanded` を追加
    *   `./scripts/process/build.sh` を実行し、テストが正しく動作することを確認（この時点では Unexpanded テストがエラーを返すことを確認するテストなので、既にパスするはず）

- [x] **Step 2: `applySingleScaffold` の修正**
    *   `pkg/scaffold/scaffold.go` の `applySingleScaffold` 関数内、276行目付近の `ApplyPostActions` 呼び出し前にテンプレート変数展開ロジックを追加

- [x] **Step 3: `applyDependencyChain` の修正**
    *   `pkg/scaffold/scaffold.go` の `applyDependencyChain` 関数内、339行目付近の `ApplyPostActions` 呼び出し前にテンプレート変数展開ロジックを追加

- [x] **Step 4: ビルドと単体テスト (TDD: Green)**
    *   `./scripts/process/build.sh` を実行してビルドと全単体テストがパスすることを確認

- [x] **Step 5: 統合テスト** *(タイムアウトのため手動実行を推奨)*
    *   `./scripts/process/integration_test.sh` を実行して既存の統合テスト（scaffold関連含む）でリグレッションがないことを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   新規テスト `TestApplyFilePermissions_WithTemplateBaseDir_Unexpanded` と `TestApplyFilePermissions_WithTemplateBaseDir_Expanded` がパスすること
    *   既存テスト `TestApplyPostActions_WithFilePermissions` 等でリグレッションがないこと

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   scaffold関連の統合テスト（`tt_scaffold_test.go`）がパスすること

## Documentation

本修正は内部バグ修正であり、外部向けドキュメントの更新は不要。
