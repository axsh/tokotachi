# 000-DevContainer-GitWorktree

> **Source Specification**: [000-DevContainer-GitWorktree.md](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/prompts/phases/000-foundation/ideas/fix-git/000-DevContainer-GitWorktree.md)

## Goal Description

Dev Container 起動時に、git worktree 構成の `.git` 参照をコンテナ内で正しく解決できるようにする。
親リポジトリの `.git/` ディレクトリと worktree メタデータを追加マウントし、コンテナ内の `.git` ファイルのパスを書き換えることで、`git status` / `git commit` / `git push` 等のコマンド、および VSCode/Cursor の Source Control パネルを動作可能にする。

## User Review Required

> [!IMPORTANT]
> **マウントパスの設計判断**: コンテナ内のマウント先を `/repo-git` と `/worktree-git` に固定する設計としています。これらのパスが既存の用途と衝突しないことをご確認ください。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: コンテナ内で git コマンドが正常動作 | Proposed Changes > `gitworktree.go` (検出) + `up.go` (マウント追加 + パス書き換え) |
| R2: worktree / 通常 git 両対応 | `gitworktree.go` の `DetectGitWorktree()` で分岐判定 |
| R3: 既存動作の後方互換性 | `DetectGitWorktree()` が `IsWorktree=false` の場合は既存ロジックのまま |
| R4: VSCode/Cursor の Source Control が動作 | R1 の実現により自動的に満たされる |
| R5: ゼロコンフィグ対応 | `cmd/up.go` 内で自動検出、devcontainer.json 変更不要 |

## Proposed Changes

### resolve パッケージ

#### [NEW] [gitworktree_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/internal/resolve/gitworktree_test.go)
*   **Description**: git worktree 検出ロジックの単体テスト（TDD: テストファーストで作成）
*   **Technical Design**:
    ```go
    // テストケース構造
    func TestDetectGitWorktree(t *testing.T) {
        // 各サブテストで t.TempDir() を使用
    }
    ```
*   **Logic**:
    *   **テストケース 1: `.git` がファイル（worktree 構成）**
        1. `t.TempDir()` にディレクトリ構造を作成:
            - `<root>/.git/worktrees/test-branch/` に `commondir`(`../..`), `gitdir`, `HEAD` ファイルを配置
            - `<root>/work/feat/test-branch/.git` に `gitdir: <root>/.git/worktrees/test-branch` を書き込み
        2. `DetectGitWorktree(<root>/work/feat/test-branch)` を呼び出し
        3. `IsWorktree == true`, `WorktreeGitDir` と `MainGitDir` が正しいパスであることを確認
    *   **テストケース 2: `.git` がディレクトリ（通常の git 構成）**
        1. `<root>/.git/` ディレクトリ（Gitリポジトリの`.git`を模擬）を作成
        2. `DetectGitWorktree(<root>)` を呼び出し
        3. `IsWorktree == false` であることを確認
    *   **テストケース 3: `.git` が存在しない**
        1. 空のディレクトリで `DetectGitWorktree()` を呼び出し
        2. `IsWorktree == false` であることを確認
    *   **テストケース 4: `.git` ファイルの `gitdir:` パスが相対パス**
        1. `gitdir: ../../.git/worktrees/test-branch` 形式のファイルを作成
        2. 正しく絶対パスに解決されることを確認

---

#### [NEW] [gitworktree.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/internal/resolve/gitworktree.go)
*   **Description**: git worktree 検出ロジック
*   **Technical Design**:
    ```go
    package resolve

    // GitWorktreeInfo holds resolved git worktree path information.
    type GitWorktreeInfo struct {
        IsWorktree     bool   // true if .git is a file (worktree)
        WorktreeGitDir string // absolute path to .git/worktrees/<name>/
        MainGitDir     string // absolute path to parent repo's .git/
    }

    // DetectGitWorktree inspects the given path's .git entry.
    // If .git is a file containing "gitdir: ...", it resolves
    // the worktree metadata dir and the main .git dir.
    func DetectGitWorktree(worktreePath string) (GitWorktreeInfo, error)
    ```
*   **Logic**:
    1. `os.Stat(filepath.Join(worktreePath, ".git"))` で `.git` の存在とタイプを確認
    2. ディレクトリの場合 → `GitWorktreeInfo{IsWorktree: false}` を返す
    3. ファイルの場合:
        - ファイル内容を読み、`gitdir: <path>` の `<path>` 部分を抽出（`strings.TrimPrefix` + `strings.TrimSpace`）
        - パスが相対の場合は `worktreePath` を基準に `filepath.Abs` で絶対パスに変換
        - 得られたパスを `WorktreeGitDir` とする（例: `C:/.../. git/worktrees/fix-git`）
        - `WorktreeGitDir` 内の `commondir` ファイルを読み取り（例: `../..`）
        - `filepath.Join(WorktreeGitDir, commondir値)` を `filepath.Abs` で絶対パスに変換 → `MainGitDir`
    4. 存在しない場合 → `GitWorktreeInfo{IsWorktree: false}` を返す

---

### action パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/internal/action/up.go)
*   **Description**: `UpOptions` に `GitWorktree` フィールドを追加し、`Up()` でマウント追加とパス書き換えを行う
*   **Technical Design**:
    ```go
    type UpOptions struct {
        // ... existing fields ...
        GitWorktree *resolve.GitWorktreeInfo // nil if not a worktree
    }
    ```
*   **Logic**:

    **Step 4 の `docker run` 引数構築部分に追加**:

    `opts.GitWorktree != nil && opts.GitWorktree.IsWorktree` が true の場合:
    1. 以下の2つのマウントを `args` に追加:
        ```
        -v <MainGitDir>:/repo-git
        -v <WorktreeGitDir>:/worktree-git
        ```
    2. コンテナ起動成功後（Step 5 の安定確認後）、`docker exec` で `.git` パスを書き換え:
        ```bash
        # .git ファイルの gitdir パスをコンテナ内パスに書き換え
        docker exec <container> sh -c 'echo "gitdir: /worktree-git" > /workspace/.git'
        
        # commondir をコンテナ内パスに書き換え
        docker exec <container> sh -c 'echo "/repo-git" > /worktree-git/commondir'
        
        # gitdir（逆参照）をコンテナ内パスに書き換え
        docker exec <container> sh -c 'echo "/workspace/.git" > /worktree-git/gitdir'
        ```

    **注意**: `wsFolder`（workspaceFolder）を使ってコンテナ内パスを構築する。デフォルトは `/workspace`。

    **新規ヘルパーメソッド**:
    ```go
    // setupGitWorktree configures git worktree paths inside the container.
    func (r *Runner) setupGitWorktree(containerName, wsFolder string) error
    ```
    - 3つの `docker exec` コマンドを実行
    - いずれかが失敗した場合はエラーを返す（ただしコンテナ停止はしない）

---

### cmd パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/cmd/up.go)
*   **Description**: worktree 検出結果を `UpOptions` に渡す
*   **Logic**:
    - `resolve.Worktree()` で worktreePath を取得した後に `resolve.DetectGitWorktree(worktreePath)` を呼び出す
    - 結果を `upOpts.GitWorktree` にセット
    - 検出されたら `ctx.Logger.Debug("Git worktree detected: mainGitDir=%s, worktreeGitDir=%s", ...)` でログ出力
    
    ```go
    // After worktreePath resolution
    gitInfo, err := resolve.DetectGitWorktree(worktreePath)
    if err != nil {
        ctx.Logger.Warn("Git worktree detection failed: %v", err)
        // Continue without git worktree support
    }
    
    // Set in upOpts
    if gitInfo.IsWorktree {
        ctx.Logger.Debug("Git worktree detected: mainGitDir=%s", gitInfo.MainGitDir)
        upOpts.GitWorktree = &gitInfo
    }
    ```

## Step-by-Step Implementation Guide

1.  **テストファイル作成 (TDD)**:
    *   `internal/resolve/gitworktree_test.go` を作成
    *   4つのテストケース（worktree構成、通常git、.gitなし、相対パス）を実装
    *   `./scripts/process/build.sh` で**テストが失敗する**ことを確認

2.  **検出ロジック実装**:
    *   `internal/resolve/gitworktree.go` を作成
    *   `GitWorktreeInfo` 構造体と `DetectGitWorktree()` 関数を実装
    *   `./scripts/process/build.sh` で**テストがパスする**ことを確認

3.  **`action.UpOptions` 拡張**:
    *   `internal/action/up.go` に `GitWorktree` フィールドを追加
    *   `Up()` メソッドにマウント追加ロジックを実装
    *   `setupGitWorktree()` ヘルパーメソッドを追加
    *   `./scripts/process/build.sh` でビルド成功を確認

4.  **`cmd/up.go` 統合**:
    *   worktreePath 取得後に `DetectGitWorktree()` を呼び出す処理を追加
    *   `./scripts/process/build.sh` でビルド成功を確認

5.  **統合テスト追加**:
    *   `tests/integration-test/devctl_up_git_test.go` に統合テストを追加
    *   テスト内容: `devctl up` 後、`docker exec <container> git status` が成功すること
    *   `./scripts/process/integration_test.sh --specify "GitWorktree"` で実行・確認

6.  **全体検証**:
    *   `./scripts/process/build.sh && ./scripts/process/integration_test.sh` で全テストパス確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `gitworktree_test.go` の4ケースがすべてパスすること
    *   既存テスト（`devcontainer_test.go`, `worktree_test.go` 等）がリグレッションなくパスすること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --specify "GitWorktree"
    ```
    *   **Log Verification**: `git status` の出力にブランチ名が含まれること、エラーが出ないこと

3.  **Full Test Suite**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   既存の統合テスト（`TestDevctlUpStartsContainer`, `TestDevctlUpIdempotent`）も引き続きパスすること

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/README.md)
*   **更新内容**: git worktree 構成でのDev Container利用時にGitが自動的に設定される旨を記載
