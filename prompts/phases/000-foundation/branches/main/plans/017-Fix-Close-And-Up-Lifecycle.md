# 017-Fix-Close-And-Up-Lifecycle

> **Source Specification**: [015-Fix-Close-And-Up-Lifecycle.md](file://prompts/phases/000-foundation/ideas/main/015-Fix-Close-And-Up-Lifecycle.md)

## Goal Description

`devctl close` 後に空ディレクトリ（ゴーストディレクトリ）が残存した場合、再度 `devctl up` が正常にworktreeを作成できるよう修正する。`Exists()` の検証ロジック強化、`Create()` のゴーストディレクトリ自動削除、`Remove()` の後処理追加、`resolve.Worktree()` の検証強化を行う。

## User Review Required

> [!IMPORTANT]
> **既存テストへの影響**: `close_test.go` の複数テストが `.git` ファイルなしのディレクトリを作成して `wm.Exists()` が `true` を返すことを前提としています。`Exists()` の修正に伴い、これらのテストでは `.git` ファイルも作成するよう修正する必要があります。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `wm.Exists()` の強化 — `.git` ファイル存在チェック | Proposed Changes > worktree パッケージ > `worktree.go` |
| `wm.Create()` の前処理 — ゴーストディレクトリ自動削除 | Proposed Changes > worktree パッケージ > `worktree.go` |
| `wm.Remove()` の後処理 — ディレクトリ残存防止 | Proposed Changes > worktree パッケージ > `worktree.go` |
| `resolve.Worktree()` の検証強化 | Proposed Changes > resolve パッケージ > `worktree.go` |

---

## Proposed Changes

### worktree パッケージ

#### [MODIFY] [worktree_test.go](file://features/devctl/internal/worktree/worktree_test.go)

*   **Description**: ゴーストディレクトリ対応の新規テストケース追加、既存テストの修正
*   **Technical Design**:
    *   既存 `TestExists_True` を修正: `.git` ファイルをディレクトリ内に作成
    *   新規テストケース4件追加
*   **Logic**:

    **`TestExists_True` の修正**:
    ```go
    func TestExists_True(t *testing.T) {
        m := newTestManager(t, true)
        dir := m.Path("test-001")
        require.NoError(t, os.MkdirAll(dir, 0o755))
        // Create .git file to simulate valid worktree
        require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../../.git/worktrees/test-001\n"), 0o644))
        assert.True(t, m.Exists("test-001"))
    }
    ```

    **`TestExists_GhostDirectory`** — 空ディレクトリ（`.git` なし）に `false` を返す:
    ```go
    func TestExists_GhostDirectory(t *testing.T) {
        m := newTestManager(t, true)
        dir := m.Path("ghost-branch")
        require.NoError(t, os.MkdirAll(dir, 0o755))
        // No .git file — ghost directory
        assert.False(t, m.Exists("ghost-branch"))
    }
    ```

    **`TestExists_ValidWorktree`** — `.git` ファイルありで `true` を返す:
    ```go
    func TestExists_ValidWorktree(t *testing.T) {
        m := newTestManager(t, true)
        dir := m.Path("valid-branch")
        require.NoError(t, os.MkdirAll(dir, 0o755))
        require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ..."), 0o644))
        assert.True(t, m.Exists("valid-branch"))
    }
    ```

    **`TestCreate_CleansGhostDirectory`** — ゴーストディレクトリを削除してから作成:
    ```go
    func TestCreate_CleansGhostDirectory(t *testing.T) {
        m := newTestManager(t, true)
        dir := m.Path("ghost-branch")
        require.NoError(t, os.MkdirAll(dir, 0o755))
        // No .git file = ghost directory
        err := m.Create("ghost-branch")
        require.NoError(t, err)
        // Verify: ghost directory was removed (no longer exists as empty dir)
        // Dry-run mode: git worktree add command should be recorded
        recs := m.CmdRunner.Recorder.Records()
        require.GreaterOrEqual(t, len(recs), 1)
    }
    ```

    **`TestRemove_CleansRemainingDirectory`** — Remove後にディレクトリが残っていても自動削除:
    ```go
    func TestRemove_CleansRemainingDirectory(t *testing.T) {
        m := newTestManager(t, true)
        dir := m.Path("leftover-branch")
        require.NoError(t, os.MkdirAll(dir, 0o755))
        err := m.Remove("leftover-branch", false)
        require.NoError(t, err)
        // Directory should be removed even in dry-run (git worktree remove is dry,
        // but the post-cleanup os.RemoveAll is real)
        _, statErr := os.Stat(dir)
        assert.True(t, os.IsNotExist(statErr), "directory should be removed after Remove()")
    }
    ```

---

#### [MODIFY] [worktree.go](file://features/devctl/internal/worktree/worktree.go)

*   **Description**: `Exists()`, `Create()`, `Remove()` の3関数を修正
*   **Technical Design**:

    **`Exists()` — `.git` ファイル存在チェック追加**:
    ```go
    // Exists checks if the worktree directory exists and is a valid git worktree.
    // A valid worktree has a .git file (not directory) pointing to the main repo's worktree metadata.
    func (m *Manager) Exists(branch string) bool {
        wtPath := m.Path(branch)
        info, err := os.Stat(wtPath)
        if err != nil || !info.IsDir() {
            return false
        }
        // Valid worktrees have a .git file inside
        gitPath := filepath.Join(wtPath, ".git")
        _, err = os.Stat(gitPath)
        return err == nil
    }
    ```

    **`Create()` — ゴーストディレクトリの前処理追加**:
    ```go
    func (m *Manager) Create(branch string) error {
        wtPath := m.Path(branch)

        // Clean up ghost directory: directory exists but is not a valid worktree
        if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
            gitPath := filepath.Join(wtPath, ".git")
            if _, gitErr := os.Stat(gitPath); os.IsNotExist(gitErr) {
                // Ghost directory — remove before creating new worktree
                os.RemoveAll(wtPath)
            }
        }

        // (以下、既存の git rev-parse + git worktree add ロジック維持)
        gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")
        _, err := m.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), gitCmd, "rev-parse", "--verify", branch)
        branchExists := err == nil
        // ... args construction and execution ...
    }
    ```

    **`Remove()` — ディレクトリ残存の後処理追加**:
    ```go
    func (m *Manager) Remove(branch string, force bool) error {
        wtPath := m.Path(branch)
        gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")

        args := []string{"worktree", "remove", wtPath}
        if force {
            args = []string{"worktree", "remove", "-f", wtPath}
        }

        if _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...); err != nil {
            return fmt.Errorf("git worktree remove failed: %w", err)
        }

        // Ensure directory is fully removed (git may leave empty directory)
        if _, err := os.Stat(wtPath); err == nil {
            os.RemoveAll(wtPath)
        }
        return nil
    }
    ```

---

### resolve パッケージ

#### [MODIFY] [worktree_test.go](file://features/devctl/internal/resolve/worktree_test.go)

*   **Description**: ゴーストディレクト対応のテストケース追加、既存テストの修正
*   **Technical Design**:

    **`TestResolveWorktree_Found` の修正** — `.git` ファイル作成を追加:
    ```go
    func TestResolveWorktree_Found(t *testing.T) {
        root := t.TempDir()
        branchDir := filepath.Join(root, "work", "feat-x")
        require.NoError(t, os.MkdirAll(branchDir, 0755))
        // Create .git file to simulate valid worktree
        require.NoError(t, os.WriteFile(filepath.Join(branchDir, ".git"), []byte("gitdir: ..."), 0644))

        path, err := resolve.Worktree(root, "feat-x")
        require.NoError(t, err)
        assert.Equal(t, branchDir, path)
    }
    ```

    **`TestResolveWorktree_GhostDirectory`** — ゴーストディレクトリでエラー:
    ```go
    func TestResolveWorktree_GhostDirectory(t *testing.T) {
        root := t.TempDir()
        branchDir := filepath.Join(root, "work", "ghost-branch")
        require.NoError(t, os.MkdirAll(branchDir, 0755))
        // No .git file = ghost directory

        _, err := resolve.Worktree(root, "ghost-branch")
        require.Error(t, err)
        assert.Contains(t, err.Error(), "ghost directory")
    }
    ```

    **`TestResolveWorktree_ValidWithGitFile`** — `.git`ファイルがある場合に成功:
    ```go
    func TestResolveWorktree_ValidWithGitFile(t *testing.T) {
        root := t.TempDir()
        branchDir := filepath.Join(root, "work", "valid-branch")
        require.NoError(t, os.MkdirAll(branchDir, 0755))
        require.NoError(t, os.WriteFile(filepath.Join(branchDir, ".git"), []byte("gitdir: ..."), 0644))

        path, err := resolve.Worktree(root, "valid-branch")
        require.NoError(t, err)
        assert.Equal(t, branchDir, path)
    }
    ```

---

#### [MODIFY] [worktree.go](file://features/devctl/internal/resolve/worktree.go)

*   **Description**: `.git` ファイル存在チェックを追加
*   **Technical Design**:
    ```go
    // Worktree resolves the worktree path for the given branch.
    // Returns error if directory exists but is not a valid git worktree (ghost directory).
    func Worktree(repoRoot, branch string) (string, error) {
        path := filepath.Join(repoRoot, "work", branch)
        if info, err := os.Stat(path); err == nil && info.IsDir() {
            // Validate: must have .git file (worktree) or .git directory
            gitPath := filepath.Join(path, ".git")
            if _, gitErr := os.Stat(gitPath); gitErr == nil {
                return path, nil
            }
            return "", fmt.Errorf("worktree for branch %q exists but is not a valid git worktree (ghost directory)", branch)
        }
        return "", fmt.Errorf("worktree for branch %q not found", branch)
    }
    ```

---

### action パッケージ (既存テスト修正)

#### [MODIFY] [close_test.go](file://features/devctl/internal/action/close_test.go)

*   **Description**: `Exists()` の強化に伴い、worktreeディレクトリに `.git` ファイルを作成するよう修正
*   **Technical Design**:
    *   `TestClose_WithFeature_LastFeature_CleansUpWorktree` (L73-74)
    *   `TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree` (L107-108)
    *   `TestClose_WithFeature_Force_PropagatedToCleanup` (L146-147)
    *   `TestClose_WithoutFeature_Unchanged` (L178-179)
    *   上記4箇所で、`os.MkdirAll(wtDir, ...)` の直後に `.git` ファイル作成を追加:
    ```go
    require.NoError(t, os.MkdirAll(wtDir, 0o755))
    // Add .git file so wm.Exists() recognizes this as a valid worktree
    require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))
    ```

---

## Step-by-Step Implementation Guide

TDD (テスト先行) でステップを進めます。

### ステップ1: worktree パッケージ — テスト追加・修正

1.  `features/devctl/internal/worktree/worktree_test.go` を編集
    *   `TestExists_True` に `.git` ファイル作成を追加
    *   `TestExists_GhostDirectory`, `TestExists_ValidWorktree`, `TestCreate_CleansGhostDirectory`, `TestRemove_CleansRemainingDirectory` を追加
2.  ビルドして **テストが失敗すること** を確認 (`TestExists_GhostDirectory` は古い `Exists()` では FAIL する)

### ステップ2: worktree パッケージ — 実装修正

1.  `features/devctl/internal/worktree/worktree.go` を編集
    *   `Exists()`: `.git` ファイルの存在チェックを追加
    *   `Create()`: ゴーストディレクトリの前処理を追加
    *   `Remove()`: ディレクトリ残存の後処理を追加
2.  ビルドしてテストが全てパスすることを確認

### ステップ3: resolve パッケージ — テスト追加・修正

1.  `features/devctl/internal/resolve/worktree_test.go` を編集
    *   `TestResolveWorktree_Found` に `.git` ファイル作成を追加
    *   `TestResolveWorktree_GhostDirectory`, `TestResolveWorktree_ValidWithGitFile` を追加
2.  ビルドして **テストが失敗すること** を確認

### ステップ4: resolve パッケージ — 実装修正

1.  `features/devctl/internal/resolve/worktree.go` を編集
    *   `.git` ファイル存在チェックとゴーストディレクトリ用エラーメッセージを追加
2.  ビルドしてテストが全てパスすることを確認

### ステップ5: 既存テスト修正

1.  `features/devctl/internal/action/close_test.go` を編集
    *   4箇所のworktreeディレクトリ作成後に `.git` ファイル作成を追加
2.  ビルドして全テストがパスすることを確認

### ステップ6: 全体ビルド・テスト

1.  `./scripts/process/build.sh` で全体ビルド＆単体テストを実行し、リグレッションなしを確認

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

### 検証対象テストケース

| テストファイル | テストケース | 検証内容 |
|---|---|---|
| `worktree_test.go` | `TestExists_True` | 修正版: `.git` ファイルありで `true` |
| `worktree_test.go` | `TestExists_False` | 既存: ディレクトリなしで `false` (リグレッションなし) |
| `worktree_test.go` | `TestExists_GhostDirectory` | 新規: `.git` なし空ディレクトリで `false` |
| `worktree_test.go` | `TestExists_ValidWorktree` | 新規: `.git` ありで `true` |
| `worktree_test.go` | `TestCreate_CleansGhostDirectory` | 新規: ゴーストディレクトリ削除後にコマンド実行 |
| `worktree_test.go` | `TestRemove_CleansRemainingDirectory` | 新規: Remove後のディレクトリ自動削除 |
| `resolve/worktree_test.go` | `TestResolveWorktree_Found` | 修正版: `.git` ファイルありで成功 |
| `resolve/worktree_test.go` | `TestResolveWorktree_NotFound` | 既存リグレッション確認 |
| `resolve/worktree_test.go` | `TestResolveWorktree_GhostDirectory` | 新規: ゴーストディレクトリでエラー |
| `resolve/worktree_test.go` | `TestResolveWorktree_ValidWithGitFile` | 新規: `.git` ありで成功 |
| `close_test.go` | 全4テスト | `.git` ファイル追加後もパスすること |

## Documentation

変更なし — 本修正は内部実装の改善であり、既存のドキュメントに影響しない。
