# 002-Fix-NestedWorktree-RecursiveDeletion

> **Source Specification**: [002-Fix-NestedWorktree-RecursiveDeletion.md](file://prompts/phases/000-foundation/ideas/fix-nested-worktree-deletion/002-Fix-NestedWorktree-RecursiveDeletion.md)

## Goal Description

`Delete()` の再帰呼び出し時に `RepoRoot` と `worktree.Manager` が親のまま渡されるバグを修正し、ネストworktreeが正しいパスで `git worktree remove` される状態にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 再帰呼び出し時の `RepoRoot` 更新 | Proposed Changes > `delete.go` |
| R2: 子worktree用 `Manager` の新規作成 | Proposed Changes > `delete.go` |
| R3: 既存テストの修正 | Proposed Changes > `delete_test.go` |
| R4: 後方互換性の維持 | Verification Plan（既存テストの全パス確認） |

## Proposed Changes

### action パッケージ

#### [MODIFY] [delete_test.go](file://features/tt/internal/action/delete_test.go)

*   **Description**: 子worktreeの `worktree remove` に渡されるパスが正しいことを検証するテストを追加。既存の `TestDelete_WithNestedWorktrees_DeletesChildrenFirst` は実行順序の検証のみで、パスの正しさを検証していないため補強する。
*   **Technical Design**:
    *   新テスト関数 `TestDelete_WithNestedWorktrees_UsesCorrectChildPath` を追加
    *   既存の `TestDelete_WithNestedWorktrees_DeletesChildrenFirst` もパス検証を追加して強化
*   **Logic**:
    *   **テストケース: `TestDelete_WithNestedWorktrees_UsesCorrectChildPath`**
        1. `env.RepoRoot` 以下に `work/parent-branch/` ディレクトリを作成し `.git` ファイルを配置
        2. `work/parent-branch/work/child-branch/` ディレクトリを作成し `.git` ファイルを配置
        3. `Delete(branch="parent-branch", RepoRoot=env.RepoRoot, Depth=10, Yes=true)` を実行
        4. `Recorder.Records()` から `worktree remove` を含むレコードを抽出
        5. **検証**: 子worktreeの `worktree remove` コマンドに渡されるパスが `<env.RepoRoot>/work/parent-branch/work/child-branch` を含むことを `assert.Contains` で確認
        6. **検証**: 親worktreeの `worktree remove` コマンドに渡されるパスが `<env.RepoRoot>/work/parent-branch` を含むことを確認
    *   **既存テスト `TestDelete_WithNestedWorktrees_DeletesChildrenFirst` の強化**:
        1. 既存の順序検証に加え、子の `worktree remove` コマンドに `parent-branch/work/child-branch` パスが含まれることを検証

---

#### [MODIFY] [delete.go](file://features/tt/internal/action/delete.go)

*   **Description**: Phase 3 の再帰呼び出し部分で `RepoRoot` を親worktreeのパスに更新し、子worktree用の `Manager` を新規作成して渡す。
*   **Technical Design**:
    *   `childRepoRoot := wm.Path(opts.Branch)` で子の `RepoRoot` を計算
    *   `childOpts.RepoRoot` に `childRepoRoot` を設定
    *   `&worktree.Manager{CmdRunner: wm.CmdRunner, RepoRoot: childRepoRoot}` で子用 `Manager` を作成
*   **Logic**:
    *   Phase 3 のループ内（現在の96-110行目）:
    ```go
    for _, childBranch := range nested {
        r.Logger.Info("Recursively deleting nested worktree: %s", childBranch)
        // 子の RepoRoot は親worktreeのパス
        childRepoRoot := wm.Path(opts.Branch)
        childOpts := DeleteOptions{
            Branch:      childBranch,
            Force:       opts.Force,
            RepoRoot:    childRepoRoot,  // 修正: 親のRepoRootではなく子のRepoRoot
            ProjectName: opts.ProjectName,
            Depth:       opts.Depth - 1,
            Yes:         true,
            Stdin:       nil,
        }
        // 子用の Manager を新しい RepoRoot で作成
        childWM := &worktree.Manager{
            CmdRunner: wm.CmdRunner,
            RepoRoot:  childRepoRoot,
        }
        if err := r.Delete(childOpts, childWM); err != nil {
            r.Logger.Warn("Failed to delete nested worktree %s: %v", childBranch, err)
        }
    }
    ```

## Step-by-Step Implementation Guide

- [x] **Step 1: テスト追加 (TDD - Red)**
    - [x] `delete_test.go` に `TestDelete_WithNestedWorktrees_UsesCorrectChildPath` を追加
    - [x] `TestDelete_WithNestedWorktrees_DeletesChildrenFirst` にパス検証アサーションを追加
    - [x] `scripts/process/build.sh` を実行し、新テストが**失敗する**ことを確認（Red）

- [x] **Step 2: バグ修正 (Green)**
    - [x] `delete.go` の Phase 3 を修正:
        - `childRepoRoot := wm.Path(opts.Branch)` を追加
        - `childOpts.RepoRoot` を `childRepoRoot` に変更
        - `childWM := &worktree.Manager{...}` を新規作成
        - `r.Delete(childOpts, childWM)` に差し替え
    - [x] `close_test.go` の `TestClose_WithNestedWorktrees_ClosesChildrenFirst` のstate file配置を修正
    - [x] `scripts/process/build.sh` を実行し、全テストが**パスする**ことを確認（Green）

- [x] **Step 3: リグレッション確認**
    - [x] 既存テスト群がすべてパスすることを再確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        - `TestDelete_WithNestedWorktrees_UsesCorrectChildPath` がパスすること
        - `TestDelete_WithNestedWorktrees_DeletesChildrenFirst` がパスすること（パス検証含む）
        - 既存テスト群（`TestDelete_RemovesWorktreeAndBranch`, `TestDelete_BlockedByActiveContainers` 等）がすべてパスすること

## Documentation

本修正はバグフィックスのみであり、外部仕様の変更を伴わないため、ドキュメントの更新は不要です。
