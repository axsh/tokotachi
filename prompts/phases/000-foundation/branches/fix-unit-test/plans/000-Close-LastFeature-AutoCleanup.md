# 000-Close-LastFeature-AutoCleanup

> **Source Specification**: `prompts/phases/000-foundation/ideas/fix-unit-test/000-Close-LastFeature-AutoCleanup.md`

## Goal Description

`close.go` の feature-specific close パスに、最後の feature を閉じた際の自動クリーンアップロジック（worktree 削除・ブランチ削除・state ファイル削除）を追加し、FAIL している2件の単体テストを PASS させる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 最後の feature 閉鎖時の自動クリーンアップ | Proposed Changes > close.go |
| Force フラグの伝播 | Proposed Changes > close.go |
| 既存テストの維持 | Verification Plan > Automated Verification |

## Proposed Changes

### action パッケージ

#### [MODIFY] [close.go](file://features/devctl/internal/action/close.go)

*   **Description**: feature-specific close パス（26〜49行目）に、最後の feature を閉じた場合の自動クリーンアップ分岐を追加する。
*   **Technical Design**:
    *   `RemoveFeature()` 呼び出し後に `len(sf.Features) == 0` を判定
    *   0 件の場合: worktree 削除 → ブランチ削除 → state ファイル削除を実行
    *   1 件以上の場合: state ファイルを保存して終了（現行動作）
*   **Logic**:
    現在の実装（26〜49行目）を以下のように変更する:

    ```go
    if opts.Feature != "" {
        // --- 既存: コンテナ停止ロジック (28〜37行目) --- そのまま維持

        // --- 既存: state から feature を削除 ---
        sf, err := state.Load(statePath)
        if err == nil {
            sf.RemoveFeature(opts.Feature)

            // --- 新規: 最後の feature 判定 ---
            if len(sf.Features) == 0 {
                // 全 feature が無くなった → worktree + branch + state を削除

                // worktree 削除
                if wm.Exists(opts.Branch) {
                    r.Logger.Info("Removing worktree work/%s...", opts.Branch)
                    if rmErr := wm.Remove(opts.Branch, opts.Force); rmErr != nil {
                        r.Logger.Warn("Worktree remove failed: %v", rmErr)
                        wtPath := wm.Path(opts.Branch)
                        if dirErr := os.RemoveAll(wtPath); dirErr != nil {
                            r.Logger.Warn("Directory cleanup also failed: %v", dirErr)
                        } else {
                            r.Logger.Info("Cleaned up worktree directory directly")
                        }
                    }
                }

                // ブランチ削除
                r.Logger.Info("Deleting branch %s...", opts.Branch)
                if brErr := wm.DeleteBranch(opts.Branch, opts.Force); brErr != nil {
                    r.Logger.Warn("Branch delete failed: %v", brErr)
                }

                // state ファイル削除
                if rmErr := state.Remove(statePath); rmErr != nil {
                    r.Logger.Warn("State file remove failed: %v", rmErr)
                }
            } else {
                // feature がまだ残っている → state を保存
                if saveErr := state.Save(statePath, sf); saveErr != nil {
                    r.Logger.Warn("Failed to save state file: %v", saveErr)
                }
            }
        }

        r.Logger.Info("Close completed for feature %s on branch %s", opts.Feature, opts.Branch)
        return nil
    }
    ```

    ポイント:
    - `len(sf.Features) == 0` の分岐内のロジックは、feature なしパス（80〜103行目）と同等
    - `opts.Force` をそのまま `wm.Remove()` と `wm.DeleteBranch()` へ渡すことで Force フラグ伝播を実現
    - worktree 削除失敗時のフォールバック（`os.RemoveAll`）も feature なしパスと同様に実装

## Step-by-Step Implementation Guide

- [x] 1. **close.go の feature-specific close パスを修正**:
    - `features/devctl/internal/action/close.go` の 39〜46 行目を上記 Logic セクションの内容に置き換える
    - 具体的には `sf.RemoveFeature(opts.Feature)` の直後に `len(sf.Features) == 0` による分岐を追加
- [x] 2. **ビルド & テスト実行で修正を検証**:
    - `./scripts/process/build.sh` を実行
    - 4件すべてのテストが PASS することを確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    - **確認項目**:
      - `TestClose_WithFeature_LastFeature_CleansUpWorktree` が PASS
      - `TestClose_WithFeature_Force_PropagatedToCleanup` が PASS
      - `TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree` が PASS（リグレッションなし）
      - `TestClose_WithoutFeature_Unchanged` が PASS（リグレッションなし）
      - 全体ビルドが成功

## Documentation

本修正は既存仕様書・ドキュメントへの影響なし。更新対象なし。
