# 015-WorktreeStaleDir-Resilience

> **Source Specification**: [013-WorktreeStaleDir-Resilience.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/013-WorktreeStaleDir-Resilience.md)

## Goal Description

`devctl close` 実行時にエディタのファイルロック等でワークツリーの削除が不完全になった場合に、残留する空ディレクトリやstateファイルが次回の `devctl up` / `devctl close` を妨げないよう、不整合検出・自動復旧ロジックを追加する。

## User Review Required

> [!IMPORTANT]
> **安全制約**: コンテナまたは git worktree 登録のいずれか一方でも存在する場合、クリーンアップロジックは実行しない。このガード条件は `up.go` と `close.go` の両方に実装する。

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: up時に空ディレクトリを検出し自動復旧 | Proposed Changes > cmd/up.go |
| R2: close再実行で残骸を削除 + WARNメッセージ | Proposed Changes > action/close.go |
| R3: 正常なworktreeはスキップ（後方互換） | Proposed Changes > worktree/worktree.go (`Exists()`) |
| R4: up時に残留stateファイルをクリーンアップ | Proposed Changes > cmd/up.go |
| R5: `Exists()` を `.git` 有無で判定に改善 | Proposed Changes > worktree/worktree.go + worktree_test.go |
| 安全制約: コンテナ or worktree登録が有ならクリーンアップ禁止 | up.go ガード条件, close.go ガード条件 |

## Proposed Changes

### worktree パッケージ

#### [MODIFY] [worktree_test.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/worktree/worktree_test.go)

*   **Description**: `Exists()` の新ロジック（`.git` チェック）に対する単体テストを追加・修正
*   **Technical Design**:
    *   既存の `TestExists_True` を修正: `os.MkdirAll(dir)` 後に `.git` ファイルも作成する
    *   `TestExists_EmptyDir` を新規追加
    *   ```go
        // TestExists_True: .git ファイルが存在する場合 → true
        func TestExists_True(t *testing.T) {
            m := newTestManager(t, true)
            dir := m.Path("devctl", "test-001")
            // Path structure: work/test-001/features/devctl
            require.NoError(t, os.MkdirAll(dir, 0o755))
            require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: dummy\n"), 0o644))
            assert.True(t, m.Exists("devctl", "test-001"))
        }

        // TestExists_EmptyDir: ディレクトリは存在するが .git が無い → false
        func TestExists_EmptyDir(t *testing.T) {
            m := newTestManager(t, true)
            dir := m.Path("devctl", "test-001")
            require.NoError(t, os.MkdirAll(dir, 0o755))
            assert.False(t, m.Exists("devctl", "test-001"))
        }

        // TestExists_False: ディレクトリが存在しない → false（既存テスト、変更なし）
        ```

---

#### [MODIFY] [worktree.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/worktree/worktree.go)

*   **Description**: `Exists()` に `.git` の存在チェックを追加
*   **Technical Design**:
    *   ```go
        // Exists checks if the worktree directory exists and is a valid git worktree.
        // A directory without .git is considered stale (not a valid worktree).
        func (m *Manager) Exists(feature, branch string) bool {
            wtPath := m.Path(feature, branch)
            info, err := os.Stat(wtPath)
            if err != nil || !info.IsDir() {
                return false
            }
            _, err = os.Stat(filepath.Join(wtPath, ".git"))
            return err == nil
        }
        ```
*   **Logic**:
    1. `os.Stat(wtPath)` でディレクトリの存在を確認
    2. `os.Stat(wtPath/.git)` でgitメタデータの存在を追加確認
    3. `.git` が無ければ `false`（空ディレクトリ = 不整合パターンA）

---

### action パッケージ

#### [MODIFY] [close.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/internal/action/close.go)

*   **Description**: Step 2 に残留ディレクトリのクリーンアップと改善されたWARNメッセージを追加
*   **Technical Design**:
    *   ```go
        func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error {
            // Step 1: Down container ... (変更なし)

            // Step 2: Remove worktree (tolerated failure)
            wtPath := wm.Path(opts.Feature, opts.Branch)
            if wm.Exists(opts.Feature, opts.Branch) {
                r.Logger.Info("Removing worktree %s...", wtPath)
                if err := wm.Remove(opts.Feature, opts.Branch, opts.Force); err != nil {
                    r.Logger.Warn("Worktree remove failed: %v", err)
                    if removeErr := os.RemoveAll(wtPath); removeErr != nil {
                        r.Logger.Warn("Directory cleanup also failed (editor may be locking it): %v", removeErr)
                        r.Logger.Warn("Please close the editor and run 'devctl close' again")
                    } else {
                        r.Logger.Info("Cleaned up worktree directory directly")
                    }
                }
            } else if info, statErr := os.Stat(wtPath); statErr == nil && info.IsDir() {
                // Stale directory: Exists()=false but physical directory remains
                r.Logger.Info("Removing stale worktree directory %s...", wtPath)
                if err := os.RemoveAll(wtPath); err != nil {
                    r.Logger.Warn("Stale directory cleanup failed: %v", err)
                } else {
                    r.Logger.Info("Stale directory cleaned up successfully")
                }
            }

            // Step 3, 4: unchanged
        }
        ```

---

### cmd パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/cmd/up.go)

*   **Description**: ワークツリー作成前に不整合検出・自動復旧ロジックを追加
*   **Technical Design**:
    *   現在の `if !wm.Exists(...)` ブロック（L69-77）内の先頭に、残留ディレクトリと残留stateファイルのクリーンアップを追加
    *   **安全制約ガード条件**: feature有りの場合、クリーンアップ前にコンテナの存在をチェック
    *   ```go
        if !wm.Exists(ctx.Feature, ctx.Branch) {
            wtPath := wm.Path(ctx.Feature, ctx.Branch)

            // Safety guard: skip cleanup if container exists (legitimate state)
            containerActive := false
            if ctx.HasFeature() && containerName != "" {
                cs := ctx.ActionRunner.Status(containerName, "")
                containerActive = cs == action.StateContainerRunning ||
                                  cs == action.StateContainerStopped
            }

            if !containerActive {
                // Clean stale directory (inconsistency pattern A)
                if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
                    ctx.Logger.Info("Stale worktree directory found, cleaning up %s...", wtPath)
                    if err := os.RemoveAll(wtPath); err != nil {
                        return fmt.Errorf("failed to remove stale worktree directory: %w", err)
                    }
                }
                // Clean stale state file (inconsistency pattern B)
                staleStatePath := state.StatePath(ctx.RepoRoot, ctx.Feature, ctx.Branch)
                if _, err := os.Stat(staleStatePath); err == nil {
                    ctx.Logger.Info("Removing stale state file: %s", staleStatePath)
                    _ = state.Remove(staleStatePath)
                }
            }

            // Create worktree ...
        }
        ```
*   **Logic**:
    1. `wm.Exists()` が `false` の場合に不整合チェック開始
    2. **ガード条件**: feature有りかつcontainerNameが解決可能な場合、`Status()` でコンテナの存在を確認。active ならクリーンアップ禁止
    3. ガード条件通過後: 残留ディレクトリ削除 → 残留stateファイル削除 → 通常のworktree作成

## Step-by-Step Implementation Guide

- [ ] 1. **[テスト: Exists の新ロジック]**:
    *   Edit `features/devctl/internal/worktree/worktree_test.go`:
        *   `TestExists_True` を修正: `.git` ファイルを作成追加
        *   `TestExists_EmptyDir` を新規追加
    *   run `scripts/process/build.sh` → テスト失敗を確認（TDD: Failed First）

- [ ] 2. **[実装: Exists の改善]**:
    *   Edit `features/devctl/internal/worktree/worktree.go`:
        *   `Exists()` メソッドに `filepath.Join(wtPath, ".git")` の存在チェックを追加
    *   run `scripts/process/build.sh` → テストパスを確認

- [ ] 3. **[実装: close.go の残留ディレクトリクリーンアップ]**:
    *   Edit `features/devctl/internal/action/close.go`:
        *   Step 2 を上記 Technical Design のとおり拡張
    *   run `scripts/process/build.sh` → 既存テストパスを確認

- [ ] 4. **[実装: up.go の自動復旧ロジック]**:
    *   Edit `features/devctl/cmd/up.go`:
        *   `if !wm.Exists(...)` ブロック内の先頭に、安全制約ガード + 残留ディレクトリ削除 + 残留stateファイル削除ロジックを追加
    *   run `scripts/process/build.sh` → 既存テストパスを確認

- [ ] 5. **[テスト: 統合テスト作成]**:
    *   Create `tests/integration-test/devctl_up_stale_test.go`:
        *   `TestUpStaleDirCleanup` と `TestUpStaleStateCleanup` を作成
    *   run `scripts/process/integration_test.sh --categories "integration-test" --specify "TestUpStale"` → テストパスを確認

- [ ] 6. **[最終検証]**:
    *   run `scripts/process/build.sh` → 全単体テストパス
    *   run `scripts/process/integration_test.sh --categories "integration-test"` → 全統合テストパス

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        *   `TestExists_True`: `.git` ファイル有りの場合 `true` が返ること
        *   `TestExists_EmptyDir`: 空ディレクトリの場合 `false` が返ること
        *   `TestExists_False`: ディレクトリ無しの場合 `false` が返ること（既存テスト）
        *   既存テスト全件パス

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestUpStale"
    ```

3.  **全統合テスト（リグレッション確認）**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```

## Documentation

影響を受ける既存ドキュメントはありません。
