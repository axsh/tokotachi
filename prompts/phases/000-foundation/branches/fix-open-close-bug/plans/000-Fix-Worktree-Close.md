# 000-Fix-Worktree-Close

> **Source Specification**: `prompts/phases/000-foundation/branches/fix-open-close-bug/ideas/000-Fix-Worktree-Close.md`

## Goal Description

`tt close` (および `tt delete`) で `git worktree remove` が失敗した際に、worktree メタデータとブランチがゴミとして残留する問題を修正する。サブモジュールの事前解除、リトライロジック、フォールバック時の prune 実行、処理順序改善、Create の堅牢性向上の5点を実装する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: サブモジュールの事前解除 | Proposed Changes > pkg/worktree/worktree.go (`deinitSubmodules`, `Remove`) |
| R2: フォールバック削除時の worktree prune 実行 | Proposed Changes > pkg/action/delete.go (Phase 4) |
| R3: Remove メソッド内でのリトライロジック | Proposed Changes > pkg/worktree/worktree.go (`Remove`) |
| R4: Delete アクションの処理順序改善 | Proposed Changes > pkg/action/delete.go (Phase 4) |
| R5: Create メソッドの堅牢性向上 | Proposed Changes > pkg/worktree/worktree.go (`Create`) |
| R6: リトライ回数・待機時間の定数化 | Proposed Changes > pkg/worktree/worktree.go (定数定義) |

## Proposed Changes

### pkg/worktree (Worktree Manager)

#### [MODIFY] [worktree.go](file://pkg/worktree/worktree.go)

*   **Description**: `Remove` メソッドにサブモジュール事前解除とリトライロジックを追加。`Create` メソッドに stale メタデータの事前 prune を追加。新規ヘルパーメソッド `deinitSubmodules` を追加。
*   **Technical Design**:

    **定数定義 (R6):**
    ```go
    const (
        // removeRetryDelay is the wait time before retrying git worktree remove.
        // This primarily addresses Windows file-lock issues where editors
        // may still hold references to worktree files.
        removeRetryDelay = 500 * time.Millisecond

        // removeMaxRetries is the maximum number of retries for git worktree remove.
        removeMaxRetries = 1
    )
    ```

    **`deinitSubmodules` メソッド (R1):**
    ```go
    // deinitSubmodules deinitializes git submodules in the worktree directory.
    // This is a prerequisite for git worktree remove when submodules are present.
    // Failures are tolerated (logged at WARN level) since the worktree may
    // not have initialized submodules.
    func (m *Manager) deinitSubmodules(wtPath string) {
        // Check if .gitmodules exists in the worktree
        gitmodulesPath := filepath.Join(wtPath, ".gitmodules")
        if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
            return
        }
        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
        opts := cmdexec.RunOption{
            Dir:          wtPath,
            FailLevelSet: true,
            FailLevel:    pkglog.LevelWarn,
            FailLabel:    "SKIP",
            QuietCmd:     false,
        }
        m.CmdRunner.RunWithOpts(opts, gitCmd, "submodule", "deinit", "--all", "-f")
    }
    ```

    **Logic**:
    - `.gitmodules` ファイルの存在をチェック (`os.Stat`)
    - 存在しない場合は即座にリターン (不要な git コマンド呼び出しを回避)
    - 存在する場合は `git -C <wtPath> submodule deinit --all -f` を実行
    - `RunWithOpts` の `Dir` オプションでワークツリーディレクトリを指定
    - 失敗しても処理は継続 (WARN ログのみ)

    **`Remove` メソッドの改修 (R1, R3, R6):**
    ```go
    func (m *Manager) Remove(branch string, force bool) error {
        wtPath := m.Path(branch)
        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")

        // Step 1: Deinit submodules if present (R1)
        m.deinitSubmodules(wtPath)

        // Step 2: Try git worktree remove
        args := []string{"worktree", "remove", wtPath}
        if force {
            args = []string{"worktree", "remove", "-f", wtPath}
        }

        _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...)

        // Step 3: Retry on failure (R3)
        if err != nil {
            for range removeMaxRetries {
                time.Sleep(removeRetryDelay)
                _, err = m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...)
                if err == nil {
                    break
                }
            }
        }

        if err != nil {
            return fmt.Errorf("git worktree remove failed: %w", err)
        }

        // Post-cleanup: remove remaining directory (existing behavior)
        if _, statErr := os.Stat(wtPath); statErr == nil {
            os.RemoveAll(wtPath)
        }
        return nil
    }
    ```

    **Logic**:
    1. `deinitSubmodules` でサブモジュールを事前解除 (R1)
    2. `git worktree remove` を実行
    3. 失敗時は `removeRetryDelay` (500ms) 待機後にリトライ (R3)
    4. リトライは最大 `removeMaxRetries` (1) 回
    5. それでも失敗した場合はエラーを返す (呼び出し元のフォールバックに委ねる)
    6. 成功時は残存ディレクトリをクリーンアップ (既存動作)

    **`Create` メソッドの改修 (R5):**
    ```go
    func (m *Manager) Create(branch string) error {
        wtPath := m.Path(branch)

        // Clean up ghost directory: directory exists but is not a valid worktree
        if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
            gitPath := filepath.Join(wtPath, ".git")
            if _, gitErr := os.Stat(gitPath); os.IsNotExist(gitErr) {
                // Ghost directory -- remove before creating new worktree
                os.RemoveAll(wtPath)
            }
        }

        // Prune stale worktree metadata before creating (R5)
        // This handles cases where a previous close failed and left
        // .git/worktrees/<name>/ metadata behind without the actual directory.
        m.Prune()

        gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
        // ... rest of existing Create logic unchanged ...
    }
    ```

    **Logic**:
    - 既存のゴーストディレクトリクリーンアップの後、`git worktree add` の前に `m.Prune()` を呼び出す
    - これにより `.git/worktrees/<name>/` に残存する stale メタデータが除去される
    - `Prune()` が失敗しても `Create` 自体は試行する (`Prune` の戻り値は無視)

*   **import 追加**: `"time"` と `pkglog "github.com/axsh/tokotachi/pkg/log"` を追加

---

### pkg/action (Delete/Close Actions)

#### [MODIFY] [delete.go](file://pkg/action/delete.go)

*   **Description**: Phase 4 の処理順序を改善。フォールバック削除後に prune を実行し、prune を branch delete の前に移動する。
*   **Technical Design**:

    **Phase 4 の改修 (R2, R4):**

    現在のコード (L109-139) を以下に置き換える:

    ```go
    // Phase 4: Remove worktree, branch, and state file
    needsPrune := false
    if wm.Exists(opts.Branch) {
        r.Logger.Info("Removing worktree work/%s...", opts.Branch)
        if err := wm.Remove(opts.Branch, effectiveForce); err != nil {
            r.Logger.Warn("Worktree remove failed: %v", err)
            // Fallback: remove directory directly
            wtPath := wm.Path(opts.Branch)
            if removeErr := os.RemoveAll(wtPath); removeErr != nil {
                r.Logger.Warn("Directory cleanup also failed: %v", removeErr)
            } else {
                r.Logger.Info("Cleaned up worktree directory directly")
                needsPrune = true // Must prune after manual removal (R2)
            }
        }
    }

    // Prune stale metadata: always after fallback removal, or when force is set (R4)
    if needsPrune || effectiveForce {
        r.Logger.Info("Pruning stale worktree metadata...")
        if err := wm.Prune(); err != nil {
            r.Logger.Warn("Worktree prune failed: %v", err)
        }
    }

    // Branch delete (after prune, so metadata is cleared) (R4)
    r.Logger.Info("Deleting branch %s...", opts.Branch)
    if err := wm.DeleteBranch(opts.Branch, effectiveForce); err != nil {
        r.Logger.Warn("Branch delete failed: %v", err)
    }

    // State file cleanup
    if err := state.Remove(statePath); err != nil {
        r.Logger.Warn("State file remove failed: %v", err)
    }

    r.Logger.Info("Delete completed for branch %s", opts.Branch)
    return nil
    ```

    **Logic** (変更点):
    1. `needsPrune` フラグを導入。`wm.Remove()` が失敗し `os.RemoveAll` でフォールバック削除が成功した場合に `true` に設定 (R2)
    2. prune の実行条件を `effectiveForce` のみから `needsPrune || effectiveForce` に変更 (R2)
    3. prune を branch delete の**前**に移動 (旧コードでは branch delete の後だった) (R4)
    4. prune 実行後に branch delete を行うため、メタデータ残存による「ブランチは worktree で使用中」エラーが解消される

---

### pkg/worktree (Test)

#### [MODIFY] [worktree_test.go](file://pkg/worktree/worktree_test.go)

*   **Description**: サブモジュール deinit、リトライ、Create の stale prune に関するテストを追加。
*   **Technical Design**:

    **`TestRemove_DeinitsSubmodulesBeforeRemove` (R1):**
    ```go
    func TestRemove_DeinitsSubmodulesBeforeRemove(t *testing.T) {
        m := newTestManager(t, true) // dry-run
        branch := "test-branch"
        wtDir := m.Path(branch)
        require.NoError(t, os.MkdirAll(wtDir, 0o755))
        // Create .gitmodules file to trigger submodule deinit
        require.NoError(t, os.WriteFile(
            filepath.Join(wtDir, ".gitmodules"),
            []byte("[submodule \"sub\"]\n\tpath = sub\n\turl = https://example.com/sub.git\n"),
            0o644,
        ))

        err := m.Remove(branch, false)
        require.NoError(t, err)

        recs := m.CmdRunner.Recorder.Records()
        // Find indices of submodule deinit and worktree remove
        deinitIdx := -1
        removeIdx := -1
        for i, r := range recs {
            if strings.Contains(r.Command, "submodule deinit") {
                deinitIdx = i
            }
            if strings.Contains(r.Command, "worktree remove") {
                removeIdx = i
            }
        }
        assert.GreaterOrEqual(t, deinitIdx, 0,
            "submodule deinit should be recorded, recs: %v", recs)
        assert.GreaterOrEqual(t, removeIdx, 0,
            "worktree remove should be recorded, recs: %v", recs)
        assert.Less(t, deinitIdx, removeIdx,
            "submodule deinit must come before worktree remove, recs: %v", recs)
    }
    ```

    **`TestRemove_SkipsDeinitWhenNoSubmodules` (R1):**
    ```go
    func TestRemove_SkipsDeinitWhenNoSubmodules(t *testing.T) {
        m := newTestManager(t, true) // dry-run
        branch := "test-branch"
        wtDir := m.Path(branch)
        require.NoError(t, os.MkdirAll(wtDir, 0o755))
        // No .gitmodules file

        err := m.Remove(branch, false)
        require.NoError(t, err)

        recs := m.CmdRunner.Recorder.Records()
        for _, r := range recs {
            assert.NotContains(t, r.Command, "submodule deinit",
                "submodule deinit should not be called when no .gitmodules, recs: %v", recs)
        }
    }
    ```

    **`TestCreate_PrunesBeforeAdd` (R5):**
    ```go
    func TestCreate_PrunesBeforeAdd(t *testing.T) {
        m := newTestManager(t, true) // dry-run
        branch := "test-branch"

        err := m.Create(branch)
        require.NoError(t, err)

        recs := m.CmdRunner.Recorder.Records()
        pruneIdx := -1
        addIdx := -1
        for i, r := range recs {
            if strings.Contains(r.Command, "worktree prune") {
                pruneIdx = i
            }
            if strings.Contains(r.Command, "worktree add") {
                addIdx = i
            }
        }
        assert.GreaterOrEqual(t, pruneIdx, 0,
            "worktree prune should be recorded, recs: %v", recs)
        assert.GreaterOrEqual(t, addIdx, 0,
            "worktree add should be recorded, recs: %v", recs)
        assert.Less(t, pruneIdx, addIdx,
            "worktree prune must come before worktree add, recs: %v", recs)
    }
    ```

    **import 追加**: `"strings"` を import に追加

---

### pkg/action (Test)

#### [MODIFY] [delete_test.go](file://pkg/action/delete_test.go)

*   **Description**: フォールバック削除後の prune 実行と、prune -> branch delete の順序を検証するテストを追加。

    **`TestDelete_PruneBeforeBranchDelete` (R4):**

    通常の delete 操作で、prune が branch delete より前に実行されることを検証する。force=true のケースで確認する (force=true の場合のみ prune が実行されるため)。

    ```go
    func TestDelete_PruneBeforeBranchDelete(t *testing.T) {
        env := newTestEnv(t)
        branch := "test-branch"

        wtDir := filepath.Join(env.RepoRoot, "work", branch)
        require.NoError(t, os.MkdirAll(wtDir, 0o755))
        require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
            []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

        err := env.Runner.Delete(action.DeleteOptions{
            Branch:      branch,
            Force:       true,
            RepoRoot:    env.RepoRoot,
            ProjectName: "test",
            Depth:       10,
            Yes:         true,
        }, env.WM)
        require.NoError(t, err)

        recs := env.Recorder.Records()
        pruneIdx := -1
        branchDelIdx := -1
        for i, r := range recs {
            if strings.Contains(r.Command, "worktree prune") {
                pruneIdx = i
            }
            if strings.Contains(r.Command, "branch -") {
                branchDelIdx = i
            }
        }
        assert.GreaterOrEqual(t, pruneIdx, 0,
            "worktree prune should be recorded, recs: %v", recs)
        assert.GreaterOrEqual(t, branchDelIdx, 0,
            "branch delete should be recorded, recs: %v", recs)
        assert.Less(t, pruneIdx, branchDelIdx,
            "worktree prune must come before branch delete, recs: %v", recs)
    }
    ```

#### [MODIFY] [close_test.go](file://pkg/action/close_test.go)

*   **Description**: `TestClose_NoForce_SkipsPrune` テストを修正する必要がある可能性がある。現在のテストは「force なしでは prune が呼ばれない」ことを検証しているが、改修後は Remove の deinitSubmodules の呼び出しで `submodule` コマンドが追加されるだけで、prune の呼び出し条件自体は変わらない (force なし + フォールバックなし = prune なし)。dry-run ではフォールバックが発生しないため、既存テストはそのまま動作する。

    **`TestClose_SubmoduleDeinit_CalledBeforeRemove` (R1 の統合確認):**
    ```go
    func TestClose_SubmoduleDeinit_CalledBeforeRemove(t *testing.T) {
        env := newTestEnv(t)
        branch := "test-branch"

        // Create worktree directory with .gitmodules
        wtDir := filepath.Join(env.RepoRoot, "work", branch)
        require.NoError(t, os.MkdirAll(wtDir, 0o755))
        require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
            []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))
        require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".gitmodules"),
            []byte("[submodule \"sub\"]\n\tpath = sub\n\turl = https://example.com/sub.git\n"),
            0o644))

        err := env.Runner.Close(action.CloseOptions{
            Branch:      branch,
            Force:       false,
            RepoRoot:    env.RepoRoot,
            ProjectName: "test",
            Yes:         true,
            Depth:       10,
        }, env.WM)
        require.NoError(t, err)

        recs := env.Recorder.Records()
        assert.True(t, hasRecordContaining(recs, "submodule deinit"),
            "submodule deinit should be called during close, recs: %v", recs)
    }
    ```

---

## Step-by-Step Implementation Guide

### Step 1: worktree_test.go にテストを追加 (Red Phase)

- [x] `pkg/worktree/worktree_test.go` に以下のテストを追加:
  - `TestRemove_DeinitsSubmodulesBeforeRemove`
  - `TestRemove_SkipsDeinitWhenNoSubmodules`
  - `TestCreate_PrunesBeforeAdd`
- [x] import に `"strings"` を追加
- [x] テストが失敗することを確認 (`deinitSubmodules` と prune が未実装のため)
- [x] コミット: `test: add failing tests for submodule deinit and create prune`

### Step 2: worktree.go に実装を追加 (Green Phase)

- [x] `pkg/worktree/worktree.go` に以下を追加:
  - 定数 `removeRetryDelay` (500ms) と `removeMaxRetries` (1) を定義
  - `deinitSubmodules` メソッドを追加
  - `Remove` メソッドを改修 (deinit 呼び出し + リトライロジック)
  - `Create` メソッドを改修 (`m.Prune()` を `git worktree add` の前に追加)
- [x] import に `"time"` と `pkglog` を追加
- [x] Step 1 のテストが全て通ることを確認
- [x] コミット: `feat: add submodule deinit, retry logic, and create prune to worktree manager`

### Step 3: delete_test.go にテストを追加 (Red Phase)

- [x] `pkg/action/delete_test.go` に以下のテストを追加:
  - `TestDelete_PruneBeforeBranchDelete`
- [x] テストが失敗することを確認 (prune が branch delete の後にあるため)
- [x] コミット: `test: add failing test for prune-before-branch-delete order`

### Step 4: delete.go の Phase 4 を改修 (Green Phase)

- [x] `pkg/action/delete.go` の Phase 4 (L109-139) を改修:
  - `needsPrune` フラグを導入
  - フォールバック削除成功時に `needsPrune = true`
  - prune の実行条件を `needsPrune || effectiveForce` に変更
  - prune を branch delete の**前**に移動
  - 末尾の `effectiveForce` による prune ブロックを削除 (上に統合済み)
- [x] Step 3 のテストが通ることを確認
- [x] コミット: `fix: improve delete action to prune before branch delete`

### Step 5: close_test.go にテストを追加

- [x] `pkg/action/close_test.go` に以下のテストを追加:
  - `TestClose_SubmoduleDeinit_CalledBeforeRemove`
- [x] テストが通ることを確認 (Step 2 で実装済みのため Green)
- [x] コミット: `test: add close integration test for submodule deinit`

### Step 6: ビルド・全体検証

- [x] `./scripts/process/build.sh --skip-frontend --skip-etc` を実行
- [x] 全単体テスト成功を確認
- [x] `./scripts/process/integration_test.sh --categories "common"` を実行
- [x] リグレッションがないことを確認
- [ ] コミット (必要に応じて)、プッシュ

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2. **Backend Integration Tests (common)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```

    本修正は `pkg/worktree` と `pkg/action` に閉じたバックエンドの変更であり、フロントエンド (GUI)、LLM、TaskEngine、Template 関連のテストには影響しない。

### Test Item Design Self-Review (testing-rules Section 11.4)

1. **網羅性の検証**: 全テストが成功すれば、以下の動作が保証される:
   - サブモジュール付き worktree の Remove で deinit が事前実行される
   - サブモジュールなしの worktree ではオーバーヘッドなし
   - Remove 失敗時にリトライが実行される
   - Create で stale メタデータが prune される
   - Delete のフォールバック時に prune が実行される
   - prune が branch delete の前に実行される
   - Close から Delete への委譲パスでも deinit が動作する

2. **証拠の十分性**: 各テストは dry-run モードの `Recorder` でコマンド記録を検証しており、「コマンドが記録された」だけでなく「実行順序が正しい」ことも検証している。

3. **迂回・抜け道の排除**: dry-run テストの制約として、実際の `git worktree remove` の失敗をシミュレートすることはできない。ただし、処理ロジック (deinit -> remove -> retry -> error return) と、呼び出し元のフォールバック処理 (needsPrune フラグ) は別々にテストすることで、全体の正しさを担保する。

4. **依存関係**: テストはボトムアップ順序で設計されている:
   - Step 1-2: `pkg/worktree` (末端の worktree 操作)
   - Step 3-4: `pkg/action/delete` (worktree を呼び出す上位層)
   - Step 5: `pkg/action/close` (delete を呼び出す最上位層)

### 総合判定 (testing-rules Section 12)

全テスト完了後、以下の総合判定を実施する:

1. 全単体テスト成功 (build.sh)
2. common カテゴリの統合テスト成功 (integration_test.sh)
3. 新規追加テスト 5 件全て成功
4. 既存テスト (close_test.go の `TestClose_NoForce_SkipsPrune` 等) がリグレッションなく成功

## Documentation

本修正は内部実装の堅牢性向上であり、`tt close` / `tt delete` / `tt open` コマンドの外部インターフェースに変更はない。ユーザーマニュアルやカタログ仕様書の更新は不要。
