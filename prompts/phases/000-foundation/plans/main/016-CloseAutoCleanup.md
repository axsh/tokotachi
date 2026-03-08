# 016-CloseAutoCleanup

> **Source Specification**: [014-CloseAutoCleanup.md](file://prompts/phases/000-foundation/ideas/main/014-CloseAutoCleanup.md)

## Goal Description

Feature 指定の `devctl close <branch> <feature>` 実行後、state ファイル内に active な feature が残っていない場合に worktree ディレクトリ・ブランチ・state ファイルを自動クリーンアップする機能を追加する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 要件1: 自動クリーンアップ (worktree/branch/state 削除) | Proposed Changes > `close.go` - `cleanupWorktree()` 呼び出し追加 |
| 要件2: active feature がある場合は worktree を残す | Proposed Changes > `close.go` - `HasActiveFeatures()` チェック |
| 要件3: `--force` フラグの伝播 | Proposed Changes > `close.go` - `cleanupWorktree()` に `opts.Force` を渡す |
| 要件4: feature 未指定 close は従来通り | Proposed Changes > `close.go` - feature 無指定パスは `cleanupWorktree()` を呼ぶだけでリファクタリング |
| 要件5: ログ出力 | Proposed Changes > `close.go` - `cleanupWorktree()` 内で INFO ログ |

## Proposed Changes

### action パッケージ

#### [NEW] [close_test.go](file://features/devctl/internal/action/close_test.go)
*   **Description**: Close アクションの単体テスト。DryRun + Recorder パターンを使用。
*   **Technical Design**:
    *   パッケージ: `action_test`
    *   テストヘルパー:
        ```go
        // newTestRunner: DryRun=true の Runner を生成
        func newTestRunner(t *testing.T) *action.Runner
        
        // newTestWorktreeManager: DryRun=true の worktree.Manager を生成（TempDir 使用）
        func newTestWorktreeManager(t *testing.T) *worktree.Manager
        
        // setupStateFile: テスト用 state ファイルを作成するヘルパー
        // features マップを受け取り、指定された features を持つ state ファイルを書き出す
        func setupStateFile(t *testing.T, repoRoot, branch string, features map[string]state.FeatureState)
        ```
    *   テストケース:

    | テスト関数 | 概要 | 検証内容 |
    |---|---|---|
    | `TestClose_WithFeature_LastFeature_CleansUpWorktree` | 最後の feature を close | state ファイルが削除されること。worktree remove コマンドが Recorder に記録されること。 |
    | `TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree` | 2つの feature のうち1つを close | state ファイルが残り、残った feature が含まれること。worktree remove コマンドが記録されないこと。 |
    | `TestClose_WithFeature_Force_PropagatedToCleanup` | `Force=true` で最後の feature を close | Recorder に `-f` フラグ付きの worktree remove と `-D` フラグ付きの branch delete が記録されること。 |
    | `TestClose_WithoutFeature_Unchanged` | Feature 未指定 close | worktree remove + branch delete + state 削除のコマンドが記録されること（リファクタリング後も動作が同じ）。 |

*   **Logic**:
    各テストは以下の手順で実行:
    1. `t.TempDir()` で一時ディレクトリを作成
    2. `setupStateFile()` で初期 state ファイルを書き出し
    3. worktree ディレクトリを `os.MkdirAll` で作成（`wm.Exists()` が true を返すように）
    4. `Runner.Close()` を呼び出し
    5. state ファイルの存在/内容、Recorder の記録を検証

---

#### [MODIFY] [close.go](file://features/devctl/internal/action/close.go)
*   **Description**: クリーンアップロジックを `cleanupWorktree()` に抽出し、feature 指定 close パスに自動クリーンアップを追加。
*   **Technical Design**:
    ```go
    // cleanupWorktree removes the worktree directory, deletes the branch,
    // and removes the state file.
    // This is the shared cleanup logic used by both feature-specific and
    // feature-less close paths.
    func (r *Runner) cleanupWorktree(opts CloseOptions, wm *worktree.Manager, statePath string)
    ```
*   **Logic**:
    1. `cleanupWorktree()` メソッドを追加:
        - `wm.Exists(opts.Branch)` で worktree 存在チェック
        - 存在すれば `wm.Remove(opts.Branch, opts.Force)` で削除
        - 失敗時は `os.RemoveAll(wm.Path(opts.Branch))` でフォールバック
        - `wm.DeleteBranch(opts.Branch, opts.Force)` でブランチ削除
        - `state.Remove(statePath)` で state ファイル削除
        - 各ステップで INFO / WARN ログ出力
    2. Feature 指定パス（`if opts.Feature != ""` ブロック）を変更:
        - state から feature を削除・保存した後、`sf.HasActiveFeatures()` をチェック
        - `false` の場合: `r.cleanupWorktree(opts, wm, statePath)` を呼び出してフルクリーンアップ
        - `true` の場合: 従来通り feature 削除のみで完了
    3. Feature 未指定パス（既存の line 52〜107）をリファクタリング:
        - コンテナ停止ループとfailCountチェックはそのまま維持
        - worktree/branch/state の削除部分を `r.cleanupWorktree()` 呼び出しに置換
        - 動作は変わらないが、コードの重複を解消

## Step-by-Step Implementation Guide

- [x] **Step 1: テストファイル作成**
    - `features/devctl/internal/action/close_test.go` を新規作成
    - ヘルパー関数 `newTestRunner`, `newTestWorktreeManager`, `setupStateFile` を実装
    - 4つのテストケースを実装
    - `scripts/process/build.sh` でビルド確認（テストは失敗するはず = Red フェーズ）

- [x] **Step 2: `cleanupWorktree()` メソッド追加**
    - `close.go` に `cleanupWorktree()` メソッドを追加
    - この時点ではまだ呼び出し元は変更しない

- [x] **Step 3: Feature 未指定パスのリファクタリング**
    - Feature 未指定 close パス（line 74-104）の worktree/branch/state 削除部分を `cleanupWorktree()` に置換
    - `scripts/process/build.sh` でリグレッションなしを確認

- [x] **Step 4: Feature 指定 close に自動クリーンアップ追加**
    - Feature 指定パスの state 保存後に `HasActiveFeatures()` チェックと `cleanupWorktree()` 呼び出しを追加
    - `scripts/process/build.sh` でテスト通過を確認（Green フェーズ）

- [x] **Step 5: ビルド最終確認**
    - `scripts/process/build.sh` で全ビルド・全テスト通過を確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    - `close_test.go` の全4テストケースが PASS すること
    - 既存の全テストがリグレッションなく PASS すること

## Documentation

該当する既存ドキュメントの更新は不要（close コマンドの引数仕様や usage に変更はないため）。
