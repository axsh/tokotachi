# 013-CloseErrorResilience

> **Source Specification**: [011-CloseErrorResilience.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/011-CloseErrorResilience.md)

## Goal Description

`devctl close` の worktree 削除失敗を致命的エラーから許容エラーに変更し、`os.RemoveAll` によるフォールバック削除を追加。全4ステップが確実に実行されるようにする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| worktree 削除失敗を許容エラーに変更 | Proposed Changes > close.go |
| フォールバック `os.RemoveAll` 削除 | Proposed Changes > close.go |
| ログ出力の一貫性 | Proposed Changes > close.go |

## Proposed Changes

### action パッケージ

#### [MODIFY] [close.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/action/close.go)
*   **Description**: Step 2 の worktree 削除失敗を `return err` → `[WARN]` + フォールバック削除に変更
*   **Technical Design**:
    ```go
    // Step 2: Remove worktree (tolerated failure)
    if wm.Exists(opts.Feature, opts.Branch) {
        r.Logger.Info("Removing worktree work/%s/%s...", opts.Feature, opts.Branch)
        if err := wm.Remove(opts.Feature, opts.Branch, opts.Force); err != nil {
            r.Logger.Warn("Worktree remove failed: %v", err)
            // Fallback: remove directory directly
            wtPath := wm.Path(opts.Feature, opts.Branch)
            if removeErr := os.RemoveAll(wtPath); removeErr != nil {
                r.Logger.Warn("Directory cleanup also failed: %v", removeErr)
            } else {
                r.Logger.Info("Cleaned up worktree directory directly")
            }
        }
    }
    ```
*   **Logic**:
    *   L37-39: `return fmt.Errorf(...)` を削除
    *   代わりに `r.Logger.Warn()` でログ出力
    *   `os.RemoveAll(wtPath)` でディレクトリの直接削除を試行
    *   import に `"os"` を追加

## Step-by-Step Implementation Guide

1.  **`close.go` を改修**:
    *   import に `"os"` を追加
    *   L37-38 の `return fmt.Errorf("worktree remove failed: %w", err)` を、`r.Logger.Warn()` + `os.RemoveAll` フォールバックに変更

2.  **ビルドとテスト実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```

## Documentation

なし（内部エラー処理の改善のため）
