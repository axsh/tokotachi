# 000-NestedWorktree-Deletion

> **Source Specification**: [000-NestedWorktree-Deletion.md](file://prompts/phases/000-foundation/ideas/fix-nested-worktree-deletion/000-NestedWorktree-Deletion.md)

## Goal Description

親 worktree を `devctl close` した際に、ネストされた子 worktree を再帰的に close し、孤立した worktree メタデータの強制クリーンアップ手段を提供する。close コマンドは Dry-run 前提の確認プロンプト付き UX とし、`--depth` で再帰の深さ制限、`--yes` で確認スキップを可能にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: close 時のネスト worktree 検出と再帰的 close（深さ制限付き） | `worktree.go` (`FindNestedWorktrees`), `close.go` (再帰 close + depth カウンタ) |
| R2: 孤立 worktree メタデータの強制クリーンアップ | `worktree.go` (`Prune`), `close.go` (`--force` 時の prune) |
| R3: close 処理のユーザーへのフィードバック改善 | `close.go` (ログメッセージ) |
| R4: `devctl up` 実行時のネスト警告 (任意) | 本計画では先送り |
| R5: Dry-run 前提の確認プロンプト | `close.go` (`confirmClose` メソッド), `cmd/close.go` (Stdin 渡し) |
| R6: `--yes` オプション | `cmd/close.go` (`--yes` フラグ), `close.go` (`Yes` フィールドで分岐) |

## Proposed Changes

### worktree パッケージ

#### [MODIFY] [worktree_test.go](file://features/devctl/internal/worktree/worktree_test.go)
*   **Description**: `FindNestedWorktrees` と `Prune` メソッドの単体テストを追加
*   **Technical Design**:
    *   テストヘルパー `newTestManager()` はそのまま再利用
    *   `FindNestedWorktrees` テストではファイルシステム上にネスト構造を作成して検出を検証
*   **Logic**:

    **テスト 1: `TestFindNestedWorktrees_WithChildren`**
    *   `work/parent/work/child-a/` と `work/parent/work/child-b/` ディレクトリを作成
    *   `child-a` には `.git` ファイルを配置（有効な worktree）
    *   `child-b` は `.git` ファイルなし（ゴーストディレクトリ → 検出対象外）
    *   `FindNestedWorktrees("parent")` → `["child-a"]` のみ返る

    **テスト 2: `TestFindNestedWorktrees_NoChildren`**
    *   `work/parent/` は存在するが `work/parent/work/` は存在しない
    *   `FindNestedWorktrees("parent")` → 空スライス

    **テスト 3: `TestFindNestedWorktrees_NoWorkDir`**
    *   `work/parent/` が存在しない
    *   `FindNestedWorktrees("nonexistent")` → 空スライス

    **テスト 4: `TestPrune`**
    *   `Prune()` を呼び出し、`Recorder` に `"worktree prune"` コマンドが記録される

---

#### [MODIFY] [worktree.go](file://features/devctl/internal/worktree/worktree.go)
*   **Description**: `FindNestedWorktrees` と `Prune` メソッドを追加
*   **Technical Design**:
    ```go
    // FindNestedWorktrees returns child worktree branch names
    // under the given branch's work/ directory.
    // Only directories with a .git file (valid worktrees) are returned.
    func (m *Manager) FindNestedWorktrees(branch string) []string

    // Prune runs 'git worktree prune' to clean up stale metadata.
    func (m *Manager) Prune() error
    ```
*   **Logic**:

    **`FindNestedWorktrees(branch string) []string`**:
    1. `childWorkDir := filepath.Join(m.Path(branch), "work")`
    2. `os.ReadDir(childWorkDir)` が失敗 → 空スライスを返す
    3. 各エントリ: ディレクトリかつ `filepath.Join(childWorkDir, entry.Name(), ".git")` が存在 → 結果に追加

    **`Prune() error`**:
    1. `gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")`
    2. `m.CmdRunner.Run(gitCmd, "worktree", "prune")` を実行
    3. エラー時 `fmt.Errorf("git worktree prune failed: %w", err)`

---

### action パッケージ

#### [MODIFY] [close_test.go](file://features/devctl/internal/action/close_test.go)
*   **Description**: ネスト worktree の再帰 close、深さ制限、確認プロンプト、`--yes`/prune テストを追加
*   **Technical Design**:
    *   既存の `testEnv` / `newTestEnv()` / `setupStateFile()` / `hasRecordContaining()` を再利用
    *   確認プロンプトのテストでは `CloseOptions.Stdin` に `strings.NewReader()` を注入
    *   既存テストは `Yes: true` を設定して確認プロンプトをスキップ（後方互換）
*   **Logic**:

    **テスト 5: `TestClose_WithNestedWorktrees_ClosesChildrenFirst`**
    1. `repoRoot/work/parent/` + `.git` ファイルを作成
    2. `repoRoot/work/parent/work/child/` + `.git` ファイルを作成
    3. 親・子両方の state ファイルを作成
    4. `Close(CloseOptions{Branch: "parent", Yes: true, Depth: 10, ...})` 呼び出し
    5. `Recorder.Records()` で子の `worktree remove` が親の `worktree remove` より前に出現することを検証
    6. 両方の state ファイルが削除されていることを検証

    **テスト 6: `TestClose_DepthLimitStopsRecursion`**
    1. 3段階ネスト: `work/a/work/b/work/c/` を作成（各 `.git` ファイル付き）
    2. `Close(CloseOptions{Branch: "a", Yes: true, Depth: 1, ...})`
    3. `b` の `worktree remove` は記録されるが、`c` の `worktree remove` は記録されないことを検証

    **テスト 7: `TestClose_Force_RunsPrune`**
    1. `Close(CloseOptions{Force: true, Yes: true, Depth: 10, ...})`
    2. `Recorder` に `"worktree prune"` が含まれることを検証

    **テスト 8: `TestClose_NoForce_SkipsPrune`**
    1. `Close(CloseOptions{Force: false, Yes: true, Depth: 10, ...})`
    2. `Recorder` に `"worktree prune"` が **含まれない** ことを検証

    **テスト 9: `TestClose_ConfirmYes_Executes`**
    1. `Stdin: strings.NewReader("y\n"), Yes: false, Depth: 10` でテスト
    2. `worktree remove` が `Recorder` に記録されることを検証

    **テスト 10: `TestClose_ConfirmNo_Aborts`**
    1. `Stdin: strings.NewReader("N\n"), Yes: false, Depth: 10` でテスト
    2. `worktree remove` が `Recorder` に記録されないことを検証
    3. close が `nil` を返す（エラーなし、単にキャンセル）

    **テスト 11: `TestClose_YesFlag_SkipsConfirmation`**
    1. `Yes: true, Depth: 10` でテスト（Stdin は nil）
    2. `worktree remove` が正常に `Recorder` に記録されることを検証

    **既存テストの修正**: 既存4テストの `CloseOptions` に `Yes: true, Depth: 10` を追加して後方互換を維持

---

#### [MODIFY] [close.go](file://features/devctl/internal/action/close.go)
*   **Description**: `CloseOptions`の拡張、ネスト再帰 close、確認プロンプト、prune を追加
*   **Technical Design**:
    ```go
    type CloseOptions struct {
        Feature     string
        Branch      string
        Force       bool
        RepoRoot    string
        ProjectName string
        Depth       int       // 再帰の最大深さ (default: 10)
        Yes         bool      // true = 確認プロンプトをスキップ
        Stdin       io.Reader // 確認プロンプト用の入力ソース
    }
    ```
*   **Logic**:

    **Close メソッドの全体フロー**:

    ```
    Close(opts, wm) {
      // Phase 1: ネスト検出 + プレビュー生成
      nested := wm.FindNestedWorktrees(opts.Branch)
      hasDepthWarning := false
      if len(nested) > 0 && opts.Depth > 0 {
        // 各子について再帰的にさらにネストがないか確認（depth制限の警告用）
        // depth=0 で子にさらに子がいれば hasDepthWarning = true
        log "Detected N nested worktree(s): ..."
      }

      // Phase 2: 確認プロンプト（--dry-run なら表示のみで return）
      if !opts.Yes {
        // プレビュー表示: 削除対象の一覧
        // hasDepthWarning なら警告メッセージ追加
        // [y/N] プロンプト → "y"/"yes" 以外なら return nil
      }

      // Phase 3: 再帰 close（深さ制限付き）
      if len(nested) > 0 && opts.Depth > 0 {
        for _, child := range nested {
          childOpts := opts  // コピー
          childOpts.Branch = child
          childOpts.Depth = opts.Depth - 1
          childOpts.Yes = true  // 子の確認はスキップ（親で承認済み）
          r.Close(childOpts, wm)
        }
      }

      // Phase 4: 既存の close 処理（Feature分岐含む）
      // ... 既存コードそのまま

      // Phase 5: --force 時の prune
      if opts.Force {
        wm.Prune()
      }
    }
    ```

    **確認プロンプトの実装（Phase 2 の詳細）**:
    1. `fmt.Fprintf(os.Stderr, ...)` でプレビューを表示:
       - `Branch: <branch>`
       - ネスト worktree がある場合: `Nested worktrees: [child-a, child-b, ...]`
       - `hasDepthWarning` の場合: `⚠ Warning: Depth limit (N) may leave deeper nested worktrees behind.`
    2. `fmt.Fprint(os.Stderr, "Proceed? [y/N]: ")`
    3. `bufio.NewScanner(opts.Stdin)` で入力を読む
    4. `strings.ToLower(strings.TrimSpace(...))` が `"y"` or `"yes"` でなければ `return nil`

    **depth カウンタの動作**:
    - 親 close 呼び出し: `Depth = N`（デフォルト 10）
    - 子の再帰呼び出し: `Depth = N - 1`
    - `Depth <= 0` の場合: ネスト検出をスキップし、`FindNestedWorktrees` を呼ばない

---

### cmd パッケージ

#### [MODIFY] [close.go](file://features/devctl/cmd/close.go)
*   **Description**: `--depth` と `--yes` フラグの追加、`CloseOptions` への値の受け渡し
*   **Technical Design**:
    ```go
    var (
        closeFlagForce bool
        closeFlagDepth int    // デフォルト: 10
        closeFlagYes   bool
    )

    func init() {
        closeCmd.Flags().BoolVar(&closeFlagForce, "force", false, "Force delete even if branch is not merged")
        closeCmd.Flags().IntVar(&closeFlagDepth, "depth", 10, "Maximum depth for recursive nested worktree close")
        closeCmd.Flags().BoolVar(&closeFlagYes, "yes", false, "Skip [y/N] confirmation and execute immediately")
    }
    ```
*   **Logic**:
    `runClose` 関数内で `CloseOptions` に新フィールドを追加:
    ```go
    action.CloseOptions{
        Feature:     ctx.Feature,
        Branch:      ctx.Branch,
        Force:       closeFlagForce,
        RepoRoot:    ctx.RepoRoot,
        ProjectName: projectName,
        Depth:       closeFlagDepth,    // 追加
        Yes:         closeFlagYes,      // 追加
        Stdin:       os.Stdin,          // 追加
    }
    ```

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> TDD アプローチに従い、テストを先に書いてから実装を行います。

### Phase 1: worktree パッケージの拡張

- [x] **Step 1: `FindNestedWorktrees` と `Prune` のテスト作成**
  - `worktree_test.go` に `TestFindNestedWorktrees_WithChildren`, `TestFindNestedWorktrees_NoChildren`, `TestFindNestedWorktrees_NoWorkDir`, `TestPrune` を追加

- [x] **Step 2: `FindNestedWorktrees` と `Prune` の実装**
  - `worktree.go` に両メソッドを追加

- [x] **Step 3: ビルド & 単体テスト**
  - `scripts/process/build.sh` で worktree パッケージの全テストがパスすることを確認

### Phase 2: CloseOptions の拡張と既存テストの修正

- [x] **Step 4: `CloseOptions` にフィールド追加**
  - `close.go` の `CloseOptions` 構造体に `Depth int`, `Yes bool`, `Stdin io.Reader` を追加
  - `import "io"` を追加

- [x] **Step 5: 既存テストに `Yes: true, Depth: 10` を追加**
  - `close_test.go` の既存4テストの `CloseOptions` に `Yes: true, Depth: 10` を追加
  - この時点でビルドが通ることを確認

### Phase 3: close テストの追加

- [x] **Step 6: 新規テストの作成**
  - `close_test.go` にテスト 5〜11 を追加（Proposed Changes のテスト 5〜11 参照）
  - `import "strings"` を追加

### Phase 4: close ロジックの実装

- [x] **Step 7: ネスト検出 + 確認プロンプト + 再帰 close + prune の実装**
  - `close.go` の `Close()` メソッドに Phase 1〜5 のロジックを実装
  - `import ("bufio", "fmt", "io", "strings")` を追加

- [x] **Step 8: ビルド & 単体テスト**
  - `scripts/process/build.sh` で全テストがパスすることを確認

### Phase 5: cmd/close.go の修正

- [x] **Step 9: `--depth` と `--yes` フラグの追加**
  - `cmd/close.go` にフラグ定義と `CloseOptions` への値渡しを追加

- [x] **Step 10: 最終ビルド & 全テスト**
  - `scripts/process/build.sh` で全テストがパスすることを確認
  - 既存テストが壊れていないことをリグレッション確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    全体ビルドと単体テストを実行します。
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**:
        *   新規 worktree テスト4件 (`TestFindNestedWorktrees_*`, `TestPrune`) が PASS
        *   新規 close テスト7件（テスト 5〜11）が PASS
        *   既存 close テスト4件が `Yes: true, Depth: 10` 追加後も PASS
        *   既存 worktree テスト8件が引き続き PASS

## Documentation

#### [MODIFY] [000-NestedWorktree-Deletion.md](file://prompts/phases/000-foundation/ideas/fix-nested-worktree-deletion/000-NestedWorktree-Deletion.md)
*   **更新内容**: R4（`devctl up` 実行時のネスト警告）を先送りした旨を記載（更新済み）
