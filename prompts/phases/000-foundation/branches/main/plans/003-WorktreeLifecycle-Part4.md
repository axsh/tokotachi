# 003-WorktreeLifecycle-Part4

> **Source Specification**: [001-WorktreeLifecycle.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/001-WorktreeLifecycle.md)

## Goal Description

devctl の Git Worktree ライフサイクル管理機能を実装する。
worktree 自動作成、状態ファイル管理、PR 作成、クローズ、ブランチ一覧表示、worktree パス変更対応を含む。

## User Review Required

> [!IMPORTANT]
> - `resolve.Worktree()` のシグネチャを `(repoRoot, feature string)` → `(repoRoot, feature, branch string)` に変更する **破壊的変更**
> - `up` サブコマンドが Git worktree を自動的に作成するため、リポジトリに commit が必要（空リポジトリでは worktree add が失敗する）
> - `close` サブコマンドは worktree 削除 + ブランチ削除を行うため、データ喪失のリスクがある。`--force` なしではマージ済みブランチのみ削除

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R2: Worktree 自動作成 | `internal/worktree/worktree.go`, `cmd/up.go` |
| R3: 状態ファイル管理 | `internal/state/state.go` |
| R4: PR 作成 | `internal/action/pr.go`, `cmd/pr.go` |
| R5: クローズ | `internal/action/close.go`, `cmd/close.go` |
| R6: worktree パス変更対応 | `internal/resolve/worktree.go`, 各 `cmd/*.go` |
| R8: ブランチ一覧 | `cmd/list.go` |
| R1: サブコマンド体系 | **Part 3 で対応済** |
| R7: 外部コマンド共通化 | **Part 3 で対応済** |
| R9: worktree 間切り替え | **対象外（将来）** |
| R10: 実行レポート | **Part 3 で対応済** |

## Proposed Changes

### Worktree 操作 (`internal/worktree/`)

#### [NEW] [worktree_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/worktree/worktree_test.go)
*   **Description**: worktree パッケージのユニットテスト（TDD 先行）
*   **テストケース**:
    -   `TestPath`: feature="devctl", branch="test-001" → `work/devctl/test-001` を返す
    -   `TestExists_True`: 存在するディレクトリで true を返す
    -   `TestExists_False`: 存在しないディレクトリで false を返す
    -   `TestCreateCmd`: dry-run で正しい git コマンド文字列が生成されることを検証
    -   `TestCreateCmd_ExistingBranch`: `-b` なしでコマンドが生成されることを検証
    -   `TestRemoveCmd`: 正しい `git worktree remove` コマンドが生成されることを検証
    -   `TestRemoveCmd_Force`: `--force` 付きで生成されることを検証
    -   `TestDeleteBranchCmd`: `git branch -d` コマンドが生成されることを検証
    -   `TestDeleteBranchCmd_Force`: `git branch -D` コマンドが生成されることを検証
    -   `TestListEntries`: ディレクトリスキャンで worktree 一覧を返すことを検証

#### [NEW] [worktree.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/worktree/worktree.go)
*   **Description**: Git Worktree 操作パッケージ
*   **Technical Design**:
    ```go
    // WorktreeInfo represents a worktree entry.
    type WorktreeInfo struct {
        Feature string
        Branch  string
        Path    string
    }

    // Manager handles git worktree operations.
    type Manager struct {
        CmdRunner *cmdexec.Runner
        RepoRoot  string
    }

    // Path returns the worktree directory path.
    func (m *Manager) Path(feature, branch string) string
    // path = filepath.Join(m.RepoRoot, "work", feature, branch)

    // Exists checks if the worktree directory exists.
    func (m *Manager) Exists(feature, branch string) bool

    // Create creates a new git worktree.
    // If the branch already exists remotely, uses it without -b.
    func (m *Manager) Create(feature, branch string) error

    // Remove removes a git worktree.
    func (m *Manager) Remove(feature, branch string, force bool) error

    // DeleteBranch deletes the local branch.
    func (m *Manager) DeleteBranch(branch string, force bool) error

    // List returns all worktree entries for a feature.
    func (m *Manager) List(feature string) ([]WorktreeInfo, error)
    ```
*   **Logic**:
    -   `Create`:
        1. `m.Path(feature, branch)` でパスを計算
        2. git コマンド名を `cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")` で取得
        3. ブランチがリモートに存在するか確認: `git rev-parse --verify <branch>` を実行
        4. 存在しなければ `git worktree add -b <branch> <path>`
        5. 存在すれば `git worktree add <path> <branch>`
    -   `Remove`: `git worktree remove <path>` (`force` なら `-f` を追加)
    -   `DeleteBranch`: `git branch -d <branch>` (`force` なら `-D`)
    -   `List`: `work/<feature>/` 配下のディレクトリを `os.ReadDir` で列挙

### 状態ファイル管理 (`internal/state/`)

#### [NEW] [state_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/state/state_test.go)
*   **Description**: state パッケージのユニットテスト（TDD 先行）
*   **テストケース**:
    -   `TestStatePath`: feature="devctl", branch="test-001" → `work/devctl/test-001.state.yaml`
    -   `TestSave_Load_Roundtrip`: Save で書き込み、Load で読み戻し、全フィールドが一致
    -   `TestLoad_NotFound`: 存在しないファイルでエラーを返す
    -   `TestRemove_Existing`: Save 後に Remove で削除成功
    -   `TestRemove_NotFound`: 存在しないファイルでもエラーなし

#### [NEW] [state.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/state/state.go)
*   **Description**: 状態ファイル YAML 管理
*   **Technical Design**:
    ```go
    // Status represents the worktree lifecycle status.
    type Status string
    const (
        StatusActive  Status = "active"
        StatusStopped Status = "stopped"
        StatusClosed  Status = "closed"
    )

    // StateFile represents the worktree state YAML file.
    type StateFile struct {
        Feature       string    `yaml:"feature"`
        Branch        string    `yaml:"branch"`
        CreatedAt     time.Time `yaml:"created_at"`
        ContainerMode string    `yaml:"container_mode"`
        Editor        string    `yaml:"editor"`
        Status        Status    `yaml:"status"`
    }

    // StatePath returns the state file path.
    func StatePath(repoRoot, feature, branch string) string
    // returns filepath.Join(repoRoot, "work", feature, branch+".state.yaml")

    // Load reads a state file from disk.
    func Load(path string) (StateFile, error)

    // Save writes a state file to disk.
    func Save(path string, s StateFile) error

    // Remove deletes a state file.
    func Remove(path string) error
    ```

### PR 作成アクション (`internal/action/`)

#### [NEW] [pr.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/pr.go)
*   **Description**: PR 作成アクション
*   **Technical Design**:
    ```go
    // PR creates a GitHub Pull Request using gh CLI.
    func (r *Runner) PR(worktreePath string) error
    ```
*   **Logic**:
    1. `gh` コマンド名を `cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")` で取得
    2. worktreePath をカレントディレクトリとして `gh pr create` を実行
    3. 注: `CmdRunner.RunInteractive` を使用（ユーザー入力が必要な場合があるため）

### クローズアクション (`internal/action/`)

#### [NEW] [close.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/close.go)
*   **Description**: クローズアクション
*   **Technical Design**:
    ```go
    // CloseOptions holds parameters for the close action.
    type CloseOptions struct {
        ContainerName string
        Feature       string
        Branch        string
        Force         bool
        RepoRoot      string
    }

    // Close performs the full close sequence:
    // 1. Down container (if running)
    // 2. Remove worktree
    // 3. Delete branch
    // 4. Remove state file
    func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error
    ```
*   **Logic**:
    1. コンテナが稼働中であれば `r.Down(opts.ContainerName)` を実行
    2. `wm.Remove(opts.Feature, opts.Branch, opts.Force)` で worktree 削除
    3. `wm.DeleteBranch(opts.Branch, opts.Force)` でローカルブランチ削除
    4. `state.Remove(state.StatePath(opts.RepoRoot, opts.Feature, opts.Branch))` で状態ファイル削除

### worktree パス変更 (`internal/resolve/`)

#### [MODIFY] [worktree.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/worktree.go)
*   **Description**: シグネチャを `(repoRoot, feature, branch string)` に変更
*   **Technical Design**:
    ```go
    // Worktree resolves the worktree path for the given feature and branch.
    // Search order: work/<feature>/<branch> → work/<feature> (backward compat)
    func Worktree(repoRoot, feature, branch string) (string, error)
    ```
*   **Logic**:
    1. `work/<feature>/<branch>` を確認 → 存在すれば返す
    2. フォールバック: `work/<feature>` を確認 → 存在すれば返す
    3. どちらも存在しなければエラー

#### [MODIFY] [worktree_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/resolve_test.go)
*   **Description**: テストを 3 引数対応に更新
*   **テストケース追加**:
    -   `TestWorktree_FeatureBranch`: `work/<feature>/<branch>` のパスが返ること
    -   `TestWorktree_FallbackFeatureOnly`: `work/<feature>` のフォールバックが効くこと

### サブコマンド (`cmd/`)

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/up.go)
*   **Description**: worktree 自動作成 + 状態ファイル作成を追加
*   **Logic**:
    1. `worktree.Manager` を初期化
    2. `wm.Exists(feature, branch)` で確認
    3. 存在しなければ `wm.Create(feature, branch)` で自動作成
    4. `resolve.Worktree(repoRoot, feature, branch)` でパス取得
    5. コンテナ起動後に `state.Save(...)` で状態ファイルを作成/更新

#### [MODIFY] [down.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/down.go)
*   **Description**: 状態ファイル更新を追加。`resolve.Worktree` を 3 引数に変更
*   **Logic**: down 後に `state.Save(...)` で `status: stopped` に更新

#### [MODIFY] [open.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/open.go)
*   **Description**: `resolve.Worktree` を 3 引数に変更

#### [MODIFY] [status.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/status.go)
*   **Description**: `resolve.Worktree` を 3 引数に変更

#### [NEW] [pr.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/pr.go)
*   **Description**: `pr` サブコマンド
*   **Logic**: `InitContext` → `resolve.Worktree` → `action.Runner.PR(worktreePath)` → レポート記録

#### [NEW] [close.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/close.go)
*   **Description**: `close` サブコマンド
*   **フラグ**: `--force`
*   **Logic**: `InitContext` → `action.Runner.Close(opts, wm)` → レポート記録

#### [NEW] [list.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/list.go)
*   **Description**: `list` サブコマンド（feature 引数のみ、branch 不要）
*   **Logic**:
    1. `worktree.Manager.List(feature)` で全 worktree を取得
    2. 各 worktree の `state.Load()` で状態取得
    3. テーブル形式で出力（Branch, Status, ContainerMode, CreatedAt）

## Step-by-Step Implementation Guide

### Phase 1: worktree パッケージ (TDD)

- [x] 1. `internal/worktree/worktree_test.go` を作成（テストケース 10 件）
- [x] 2. `internal/worktree/worktree.go` を実装（Manager, Create, Remove, DeleteBranch, List, Exists, Path）
- [x] 3. テスト実行 → 全 PASS 確認

### Phase 2: state パッケージ (TDD)

- [x] 4. `internal/state/state_test.go` を作成（テストケース 5 件）
- [x] 5. `internal/state/state.go` を実装（StateFile, Status, Load, Save, Remove, StatePath）
- [x] 6. テスト実行 → 全 PASS 確認

### Phase 3: resolve.Worktree 変更

- [x] 7. `internal/resolve/worktree.go` を 3 引数に変更
- [x] 8. `internal/resolve/resolve_test.go` にテストケース追加
- [x] 9. テスト実行 → 全 PASS 確認

### Phase 4: action パッケージ拡張

- [x] 10. `internal/action/pr.go` を作成
- [x] 11. `internal/action/close.go` を作成

### Phase 5: サブコマンド更新・追加

- [x] 12. `cmd/up.go` を修正（worktree 自動作成 + 状態ファイル）
- [x] 13. `cmd/down.go` を修正（状態ファイル更新 + 3 引数対応）
- [x] 14. `cmd/open.go`, `cmd/status.go` を修正（3 引数対応）
- [x] 15. `cmd/pr.go` を作成
- [x] 16. `cmd/close.go` を作成
- [x] 17. `cmd/list.go` を作成
- [x] 18. `cmd/root.go` にサブコマンド追加登録
- [x] 19. ビルド・全テスト → 全 PASS 確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **手動動作確認（dry-run）**:
    ```bash
    # worktree 自動作成 + コンテナ起動（dry-run）
    ./bin/devctl up devctl test-001 --dry-run --verbose

    # 状態確認
    ./bin/devctl status devctl test-001

    # 一覧表示
    ./bin/devctl list devctl

    # PR 作成（dry-run）
    ./bin/devctl pr devctl test-001 --dry-run

    # クローズ（dry-run）
    ./bin/devctl close devctl test-001 --dry-run
    ```

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/escape/features/devctl/README.md)
*   **更新内容**: `pr`/`close`/`list` サブコマンドの説明追加。状態ファイルの説明追加。
