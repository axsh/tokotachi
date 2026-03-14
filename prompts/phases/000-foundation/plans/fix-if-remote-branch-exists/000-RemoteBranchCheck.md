# 000-RemoteBranchCheck

> **Source Specification**: [000-RemoteBranchCheck.md](file://prompts/phases/000-foundation/ideas/fix-if-remote-branch-exists/000-RemoteBranchCheck.md)

## Goal Description

`pkg/worktree/worktree.go` の `Create` メソッドを拡張し、ローカルにブランチが存在しない場合にリモートリポジトリ (`origin`) に同名ブランチが存在するかを確認する。リモートに存在すればフェッチしてからワークツリーを作成し、存在しなければ従来通り新規ブランチを作成する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| リモートブランチの確認 | Proposed Changes > `pkg/worktree/worktree.go` - `RemoteBranchExists` メソッド |
| リモートブランチのフェッチ | Proposed Changes > `pkg/worktree/worktree.go` - `Create` メソッド内 fetch 処理 |
| リモートブランチからのワークツリー作成 | Proposed Changes > `pkg/worktree/worktree.go` - `Create` メソッド内分岐 |
| 既存動作の維持 (リモートにない場合) | Proposed Changes > `pkg/worktree/worktree.go` - `Create` メソッド 既存分岐維持 |
| ローカルブランチ優先 | Proposed Changes > `pkg/worktree/worktree.go` - `Create` メソッド 既存の `rev-parse` チェック維持 |
| フェッチ失敗時フォールバック | Proposed Changes > `pkg/worktree/worktree.go` - `Create` メソッド fetch エラーハンドリング |

## Proposed Changes

### pkg/worktree

#### [MODIFY] [worktree_test.go](file://pkg/worktree/worktree_test.go)

*   **Description**: リモートブランチ確認ロジックのテストケースを追加する
*   **Technical Design**:
    *   dry-run モードの `cmdexec.Recorder` を使って、実行されるgitコマンドの順序と引数を検証する
    *   dry-run モードでは `RunWithOpts` は常に `("", nil)` を返すため、`ls-remote` の成功は「リモートブランチあり」として扱われる。この特性を利用し、dry-run でのコマンド記録順序を検証する
*   **Logic**:
    *   追加するテストケース:
        1.  `TestCreate_RemoteBranchExists`: ローカルブランチなし & dry-run（= `ls-remote` 成功として扱われる）時、`rev-parse` → `ls-remote` → `fetch` → `worktree add` の順序でコマンドが記録されることを検証。特に `worktree add` に `-b` フラグが**含まれない**こと、`fetch origin <branch>` が記録されることを確認
        2.  `TestCreate_RemoteBranchCheckOrder`: dry-run下での全体的なコマンド記録順序を検証。ローカル存在チェック → リモート確認 → フェッチ → ワークツリー追加 の4ステップが正しい順序で記録されること
    *   テストのパターン:
        ```go
        func TestCreate_RemoteBranchExists(t *testing.T) {
            m := newTestManager(t, true) // dry-run = true
            err := m.Create("remote-branch")
            require.NoError(t, err)
            recs := m.CmdRunner.Recorder.Records()
            // dry-run では rev-parse が成功("", nil)を返すため、
            // ローカルブランチ存在として処理される。
            // → リモートチェックには到達しない。
            // 
            // リモートチェック到達を検証するには、
            // rev-parse を失敗させる仕組みが必要。
            // → RemoteBranchExists を独立メソッドとしてテスト可能にする。
        }
        ```

> [!IMPORTANT]
> **テスト戦略の要点**: dry-run モードでは `CheckOpt()` 付きの `RunWithOpts` も `("", nil)` を返すため、`rev-parse` が常に "ブランチ存在" と判定される。リモートチェックの分岐をテストするためには、 `RemoteBranchExists` メソッドを `Create` から分離して公開し、単体テスト可能にする。`Create` 全体の統合テストは Integration Test で検証する。

#### [MODIFY] [worktree.go](file://pkg/worktree/worktree.go)

*   **Description**: `Create` メソッドにリモートブランチ確認・フェッチロジックを追加する。リモート確認を独立メソッドとして分離する。
*   **Technical Design**:
    *   新規メソッド `RemoteBranchExists(branch string) bool` を追加
    *   新規メソッド `FetchBranch(branch string) error` を追加
    *   `Create` メソッドのローカルブランチ不在時の分岐を拡張
    *   関数シグネチャ:
        ```go
        // RemoteBranchExists checks if a branch exists on the remote (origin).
        // Uses "git ls-remote --heads origin <branch>" to check.
        // Returns true if the remote has a matching ref.
        func (m *Manager) RemoteBranchExists(branch string) bool

        // FetchBranch fetches a specific branch from the remote (origin).
        func (m *Manager) FetchBranch(branch string) error
        ```
*   **Logic**:
    *   `RemoteBranchExists` の実装:
        1.  `git ls-remote --heads origin <branch>` を `CheckOpt()` 付きで実行
        2.  コマンドが成功し、出力が空でなければ `true` を返す
        3.  コマンドが失敗した場合（ネットワークエラー等）は `false` を返す
        4.  出力の解析: `ls-remote` は `<hash>\trefs/heads/<branch>` 形式で出力する。`strings.TrimSpace(output)` が空文字列でなければブランチが存在する
    *   `FetchBranch` の実装:
        1.  `git fetch origin <branch>` を実行
        2.  エラーがあればそのまま返す
    *   `Create` メソッドの変更:
        ```
        既存: branchExists → (Y) worktree add --force / (N) worktree add -b
        変更後:
          branchExists → (Y) worktree add --force
          branchExists → (N) → remoteBranchExists?
            → (Y) fetch → worktree add <path> <branch>
            → (N) worktree add -b <branch> <path>
        ```
    *   フェッチ失敗時のフォールバック:
        ```go
        if remoteBranchExists {
            if fetchErr := m.FetchBranch(branch); fetchErr != nil {
                // フェッチ失敗 → 新規ブランチ作成にフォールバック
                args = []string{"worktree", "add", "-b", branch, wtPath}
            } else {
                // フェッチ成功 → リモート追跡ブランチとしてワークツリー作成
                args = []string{"worktree", "add", wtPath, branch}
            }
        }
        ```

## Step-by-Step Implementation Guide

1.  **テスト作成 (TDD - Red phase)**: ✅
    *   `pkg/worktree/worktree_test.go` に以下のテストを追加:
        - [x] `TestRemoteBranchExists_DryRun`: `RemoteBranchExists` メソッドを呼び出し、dry-runで `ls-remote` コマンドが Recorder に記録されることを検証
        - [x] `TestFetchBranch_DryRun`: `FetchBranch` メソッドを呼び出し、dry-runで `fetch origin <branch>` コマンドが Recorder に記録されることを検証
    *   [x] ビルドを実行し、コンパイルエラーを確認（メソッド未実装のため）

2.  **`RemoteBranchExists` メソッドの実装 (TDD - Green phase)**: ✅
    *   [x] `pkg/worktree/worktree.go` に `RemoteBranchExists` メソッドを追加
    *   [x] `git ls-remote --heads origin <branch>` を `CheckOpt()` 付きで実行
    *   [x] 出力が空でなければ `true` を返す

3.  **`FetchBranch` メソッドの実装 (TDD - Green phase)**: ✅
    *   [x] `pkg/worktree/worktree.go` に `FetchBranch` メソッドを追加
    *   [x] `git fetch origin <branch>` を実行

4.  **ビルド & テスト実行で Green 確認**: ✅
    *   [x] `scripts/process/build.sh` を実行し、テストが通ることを確認

5.  **`Create` メソッドの修正**: ✅
    *   [x] ローカルブランチ不在時にリモートチェック・フェッチ分岐を追加
    *   [x] フェッチ失敗時のフォールバック処理を実装

6.  **ビルド & テスト実行で全体 Green 確認**: ✅
    *   [x] `scripts/process/build.sh` を実行し、全テストが通ることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   `pkg/worktree` パッケージの全テストが PASS すること
    *   新規追加テスト (`TestRemoteBranchExists_DryRun`, `TestFetchBranch_DryRun`, `TestCreate_CommandSequence_DryRun`) が PASS すること

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Open"
    ```
    *   既存の `TestTtOpen_SyntaxSugar_DryRun` が引き続き PASS すること
    *   `TestTtOpen_SyntaxSugar_WithFeature_DryRun` が引き続き PASS すること

## Documentation

影響を受ける既存ドキュメントはありません。
