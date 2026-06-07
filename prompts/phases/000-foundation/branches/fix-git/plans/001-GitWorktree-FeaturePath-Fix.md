# 001-GitWorktree-FeaturePath-Fix

> **Source Specification**: [001-GitWorktree-FeaturePath-Fix.md](file:///prompts/phases/000-foundation/ideas/fix-git/001-GitWorktree-FeaturePath-Fix.md)

## Goal Description

feature指定時（`work/<branch>/features/<feature>/`）のworktreeフォルダで、ホスト側の`.git`ファイルがコンテナ内パス（`/worktree-git`）に書き換えられたまま残る問題を修正する。`setupGitWorktree`のアプローチを、ホスト側の`.git`ファイルを直接書き換える方式から、一時ファイルを`.git`パスにオーバーライドマウントする方式に変更する。

## User Review Required

> [!IMPORTANT]
> **`.git`ファイルのマウント方式の変更**: 現在の`docker exec`による`.git`書き換えを廃止し、`docker run`時に一時ファイルをオーバーライドマウントする方式に変更します。これにより`.git.devctl-backup`の仕組みは不要になります。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: feature指定時のホスト側Git動作 | `action/up.go` の`.git`オーバーライドマウント + `setupGitWorktree`の`.git`書き換え廃止 |
| R2: feature指定なし時の後方互換 | 変更なし（既存ロジックのまま、テストで確認） |
| R3: ホスト側`.git`ファイルの維持 | `action/up.go` の`.git`書き換え廃止 + `gitworktree.go` のバックアップ復元コード簡素化 |
| R4: コンテナ内Git動作 | `action/up.go` のオーバーライドマウント + `setupGitWorktree`のcommondir/gitdir書き換え維持 |

## Proposed Changes

### resolve パッケージ

#### [MODIFY] [gitworktree.go](file:///features/devctl/internal/resolve/gitworktree.go)
*   **Description**: バックアップ復元ロジックの簡素化と新ヘルパー関数`CreateContainerGitFile`の追加
*   **Technical Design**:
    ```go
    // CreateContainerGitFile creates a temporary file on the host containing
    // the container-internal gitdir path, for use as an override mount.
    // Returns the path to the temporary file.
    func CreateContainerGitFile(tempDir string) (string, error)
    ```
*   **Logic**:
    *   `CreateContainerGitFile`:
        *   `tempDir` に `dot-git-override` という名前のファイルを作成
        *   内容: `gitdir: /worktree-git\n`
        *   作成したファイルの絶対パスを返す
    *   `DetectGitWorktree` のバックアップ復元ロジック（L60-92）:
        *   `.git`ファイルの内容が`/worktree-git`を指している場合（コンテナ内パスが残留している場合）のハンドリングを追加
        *   バックアップファイル（`.git.devctl-backup`）が存在する場合、バックアップから復元してから検出を続行
        *   バックアップが存在しない場合、コンテナパス残留としてエラーを返す
        *   既存のバックアップ復元ロジックは維持するが、コンテナパス(`/worktree-git`等)の検出をより明確にする

#### [MODIFY] [gitworktree_test.go](file:///features/devctl/internal/resolve/gitworktree_test.go)
*   **Description**: `CreateContainerGitFile`のテストとコンテナパス残留のバックアップ復元テストを追加
*   **Technical Design**: テーブル駆動テスト
*   **Logic**:
    *   `TestCreateContainerGitFile`: 一時ファイルが正しい内容で作成されることを確認
    *   `TestDetectGitWorktree_ContainerPathWithBackup`: `.git`がコンテナパスを指している状態でバックアップが存在する場合、バックアップから復元が行われることを確認
    *   `TestDetectGitWorktree_ContainerPathWithoutBackup`: `.git`がコンテナパスを指していてバックアップがない場合、エラーが返ることを確認

---

### action パッケージ

#### [MODIFY] [up.go](file:///features/devctl/internal/action/up.go)
*   **Description**: `.git`ファイルのオーバーライドマウント追加と`setupGitWorktree`の書き換えロジック廃止
*   **Technical Design**:
    *   `UpOptions` に新フィールド追加:
        ```go
        type UpOptions struct {
            // ... 既存フィールド ...
            GitWorktree      *resolve.GitWorktreeInfo // nil if not a worktree
            GitOverrideFile  string                   // path to temp .git override file (for container mount)
        }
        ```
    *   `Up()` の `docker run` args 構築部分（L66-79）で、`GitOverrideFile`が指定されている場合、`.git`ファイルのオーバーライドマウントを追加:
        ```go
        // 既存: ワークスペースマウント
        args = append(args, "-v", opts.WorktreePath+":"+wsFolder)
        // 既存: repo-git, worktree-git-src マウント
        args = append(args, "-v", opts.GitWorktree.MainGitDir+":/repo-git:ro")
        args = append(args, "-v", opts.GitWorktree.WorktreeGitDir+":/worktree-git-src:ro")
        // 新規: .git ファイルのオーバーライドマウント（ホスト側 .git を変更しない）
        args = append(args, "-v", opts.GitOverrideFile+":"+wsFolder+"/.git")
        ```
    *   `setupGitWorktree()`:
        *   Step 1（worktreeメタデータのコピー）: **維持** — `/worktree-git-src` → `/worktree-git`
        *   Step 2（`.git`ファイルの書き換え）: **削除** — オーバーライドマウントで代替
        *   Step 3（commondir書き換え）: **維持**
        *   Step 4（gitdir逆参照書き換え）: **維持**

#### [MODIFY] [up_test.go ※新規](file:///features/devctl/internal/action/up_test.go)
*   **Description**: 現在action パッケージにはテストファイルが存在しないため、不要。`GitOverrideFile`の動作は統合テストで検証する。

---

### cmd パッケージ

#### [MODIFY] [up.go](file:///features/devctl/cmd/up.go)
*   **Description**: `DetectGitWorktree`呼び出し後に`CreateContainerGitFile`を呼び出し、`UpOptions.GitOverrideFile`に設定する
*   **Logic**:
    *   L108-116（git worktree検出部分）の後に追加:
        ```go
        if gitErr == nil && gitInfo.IsWorktree {
            // Create temp .git override file for container mount
            overrideFile, err := resolve.CreateContainerGitFile(os.TempDir())
            if err != nil {
                ctx.Logger.Warn("Failed to create git override file: %v", err)
            } else {
                upOpts.GitOverrideFile = overrideFile
            }
        }
        ```
    *   **注意**: 一時ファイルはコンテナが起動している間はDockerのバインドマウントが参照し続けるため、`defer os.Remove`で即削除してはいけない。一時ファイルは`os.TempDir()`に配置されるためOSの一時ファイルクリーンアップに任せる。名前を固定（`dot-git-override`）にすることで、次回実行時に上書きされる。

#### [MODIFY] [open.go](file:///features/devctl/cmd/open.go)
*   **Description**: `--up`フラグ使用時の`Up()`呼び出しでも同様に`GitOverrideFile`を設定する
*   **Logic**:
    *   L98-136（`--up`フラグ処理、git worktree検出部分）で同様の`CreateContainerGitFile`呼び出しを追加

---

### gitworktree.go バックアップ関連のクリーンアップ

#### [MODIFY] [gitworktree.go](file:///features/devctl/internal/resolve/gitworktree.go)
*   **Description**: 新方式では`.git.devctl-backup`の作成は不要になるが、既存のバックアップから復元するロジックは**互換性のために維持**する。新規でバックアップを作成するコードは`setupGitWorktree`から削除する。
*   **Logic**:
    *   `DetectGitWorktree`のバックアップ復元ロジック（L60-92）: 既存のまま維持
    *   将来的にバックアップ機能を削除する際は、別の仕様として対応

## Step-by-Step Implementation Guide

### Phase 1: テスト先行（TDD）

1.  **`gitworktree_test.go` にテスト追加**:
    *   `TestCreateContainerGitFile` を追加: `CreateContainerGitFile(t.TempDir())`を呼び出し、返されたファイルパスにファイルが存在し、内容が`gitdir: /worktree-git\n`であることを確認
    *   `TestDetectGitWorktree_ContainerPathWithBackup` を追加: `.git`に`gitdir: /worktree-git`を書き込み、`.git.devctl-backup`に正しいgitdirを書き込んだ状態で`DetectGitWorktree`を呼び出し、バックアップから復元されることを確認
    *   ビルドを実行 → テストが失敗することを確認（`CreateContainerGitFile`が存在しないため）

2.  **`CreateContainerGitFile` を実装**:
    *   `gitworktree.go` に`CreateContainerGitFile(tempDir string) (string, error)` を追加
    *   ビルドを実行 → テストがパスすることを確認

### Phase 2: action/up.go の修正

3.  **`UpOptions` に `GitOverrideFile` フィールドを追加**:
    *   `up.go` の`UpOptions`構造体に`GitOverrideFile string`フィールドを追加

4.  **`Up()` にオーバーライドマウントを追加**:
    *   L74-79（git worktreeマウント追加部分）の後に、`GitOverrideFile`が設定されている場合の`.git`オーバーライドマウントを追加:
        ```go
        if opts.GitOverrideFile != "" {
            args = append(args, "-v", opts.GitOverrideFile+":"+wsFolder+"/.git")
        }
        ```

5.  **`setupGitWorktree()` から`.git`書き換えを削除**:
    *   Step 2（`.git`バックアップ＆書き換え、L151-160）を削除
    *   Step 1, 3, 4 はそのまま維持

### Phase 3: cmd の修正

6.  **`cmd/up.go` に`CreateContainerGitFile`呼び出しを追加**:
    *   L158-161（GitWorktree設定部分）の後に、`CreateContainerGitFile`呼び出しを追加
    *   `upOpts.GitOverrideFile` に設定

7.  **`cmd/open.go` に同様の修正を追加**:
    *   L135-137（`--up`フラグ内のGitWorktree設定部分）の後に、同様の`CreateContainerGitFile`呼び出しを追加

### Phase 4: ビルドと検証

8.  **ビルド & 単体テスト**:
    *   `scripts/process/build.sh` を実行
    *   全テストがパスすることを確認

9.  **統合テスト**:
    *   `scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlUpGitWorktree"` を実行
    *   コンテナ内で`git status`が動作することを確認

10. **ホスト側`.git`ファイルの確認（手動）**:
    *   統合テスト後、worktreeフォルダの`.git`ファイルがコンテナ内パスに汚染されていないことを手動確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        *   `TestCreateContainerGitFile` がパス
        *   `TestDetectGitWorktree_ContainerPathWithBackup` がパス
        *   既存の`TestDetectGitWorktree_*` テストが全てパス（後方互換）

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDevctlUpGitWorktree"
    ```
    *   **Log Verification**: コンテナ内で`git status`が成功し、`fatal:`エラーが含まれないこと

### Manual Verification

統合テスト実行後に以下を手動確認:

1. worktreeフォルダの`.git`ファイルの内容を確認:
   ```bash
   cat work/<branch>/features/<feature>/.git
   ```
   期待値: `gitdir: <ホスト側の正しいパス>` （`/worktree-git`ではないこと）

## Documentation

#### [MODIFY] [000-DevContainer-GitWorktree.md](file:///prompts/phases/000-foundation/ideas/fix-git/000-DevContainer-GitWorktree.md)
*   **更新内容**: 本修正により、「コンテナ内の `.git` ファイル書き換え」方式から「`.git`ファイルのオーバーライドマウント」方式に変更された旨を追記（実現方針セクション）
