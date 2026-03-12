# 000-CloseGuardrail-PendingChangesWarning

> **Source Specification**: [000-CloseGuardrail-PendingChangesWarning.md](file://prompts/phases/000-foundation/ideas/feat-guardrail-notify-commits/000-CloseGuardrail-PendingChangesWarning.md)

## Goal Description

`tt close` コマンドで worktree を削除する前に、4カテゴリの保留中の変更（未追跡ファイル、ステージされていない変更、ステージ済み・未コミット、未プッシュコミット）を検出して表示し、ユーザーに確認を求めるガードレール機能を追加する。

## User Review Required

> [!IMPORTANT]
> **2段階確認**: 保留変更がある場合、新しい警告プロンプト `[y/N]` の後に、既存の `Delete` 内の `Proceed? [y/N]` が続く2段階確認になります。`--yes` フラグで両方スキップ。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| (1) 未追跡ファイルの検出 | Proposed Changes > `close.go` > `collectPendingChanges` |
| (2) ステージされていない変更の検出 | Proposed Changes > `close.go` > `collectPendingChanges` |
| (3) ステージ済み・未コミットの検出 | Proposed Changes > `close.go` > `collectPendingChanges` |
| (4) 未プッシュコミットの検出 | Proposed Changes > `close.go` > `collectPendingChanges` |
| 各カテゴリ最大10行表示、省略時に合計件数表示 | Proposed Changes > `close.go` > `displayPendingChanges` |
| `--verbose` で全件表示 | Proposed Changes > `cmd/close.go` > フラグ追加、`close.go` > `displayPendingChanges` |
| 1件以上で英語の警告メッセージ + `[y/N]` | Proposed Changes > `close.go` > `Close()` 内の確認ロジック |
| 2段階確認（警告 → 既存 `Proceed?`） | Proposed Changes > `close.go` > `Close()` の流れ制御 |
| `--yes` で両方の確認をスキップ | 既存 `CloseOptions.Yes` を警告プロンプトにも適用 |

## Proposed Changes

### action パッケージ

#### [MODIFY] [close_test.go](file://features/tt/internal/action/close_test.go)

*   **Description**: 保留変更の検出・表示・確認ロジックのテストを追加
*   **Technical Design**:
    *   既存の `newTestEnv()` と `testEnv` を再利用する
    *   DryRun モードでは gitコマンドは実行されず空文字列を返すため、`collectPendingChanges` のテストでは **`cmdexec.Runner` の出力をモックするアプローチ** が必要
    *   テスト戦略: `collectPendingChanges` と `displayPendingChanges` は `Close()` から呼ばれる内部関数だが、**テスト用にパッケージレベル関数としてエクスポートし、個別にテスト可能にする**
    *   あるいは、`collectPendingChanges` はgitコマンドの実行結果をパースする純粋なロジック部分を分離してテスト可能にする

*   **テストケース設計 (テーブル駆動)**:

    ```go
    // TestParsePendingChanges: git出力文字列からPendingChangesへのパースをテスト
    func TestParsePendingChanges(t *testing.T)
    tests := []struct {
        name            string
        untrackedOutput string // git ls-files --others 出力
        unstagedOutput  string // git diff --name-only 出力
        stagedOutput    string // git diff --cached --name-only 出力
        unpushedOutput  string // git log @{upstream}..HEAD --oneline 出力
        expectTotal     int
    }{
        {name: "all empty", expectTotal: 0},
        {name: "untracked only",  untrackedOutput: "a.txt\nb.txt", expectTotal: 2},
        {name: "mixed categories", untrackedOutput: "new.go", unstagedOutput: "mod.go", stagedOutput: "staged.go", unpushedOutput: "abc1234 msg", expectTotal: 4},
    }
    ```

    ```go
    // TestFormatPendingChanges: 表示制限（10行省略）のテスト
    func TestFormatPendingChanges(t *testing.T)
    tests := []struct {
        name     string
        items    []string
        verbose  bool
        expectContains   []string // 含まれるべき文字列
        expectNotContains []string // 含まれてはいけない文字列
    }{
        {name: "under limit", items: generate(5), verbose: false},         // 5件 → 省略なし
        {name: "at limit", items: generate(10), verbose: false},           // 10件 → 省略なし
        {name: "over limit", items: generate(15), verbose: false},         // 15件 → "...and 5 more (15 total)"
        {name: "over limit verbose", items: generate(15), verbose: true},  // 15件 verbose → 全件表示
    }
    ```

    ```go
    // TestClose_PendingChanges_ConfirmNo_Aborts: E2E的な確認テスト
    // DryRunモードではgitコマンドが空文字列を返すため、pending changesは0件になる
    // → 警告なしでDeleteに進むことを確認（既存テストと同じ動作）
    func TestClose_PendingChanges_ConfirmNo_Aborts(t *testing.T)

    // TestClose_YesFlag_SkipsPendingPrompt: --yes時に警告もスキップ
    func TestClose_YesFlag_SkipsPendingPrompt(t *testing.T)
    ```

#### [MODIFY] [close.go](file://features/tt/internal/action/close.go)

*   **Description**: `Close()` に保留変更チェックロジックを追加し、新しいデータ構造とヘルパー関数を実装
*   **Technical Design**:

    **データ構造**:
    ```go
    // PendingChanges holds categorized pending changes in a worktree.
    type PendingChanges struct {
        UntrackedFiles  []string // git ls-files --others --exclude-standard
        UnstagedChanges []string // git diff --name-only
        StagedChanges   []string // git diff --cached --name-only
        UnpushedCommits []string // git log @{upstream}..HEAD --oneline
    }

    // TotalCount returns total number of pending items across all categories.
    func (p PendingChanges) TotalCount() int {
        return len(p.UntrackedFiles) + len(p.UnstagedChanges) +
               len(p.StagedChanges) + len(p.UnpushedCommits)
    }
    ```

    **新規関数**:

    ```go
    // collectPendingChanges runs git commands in the worktree directory
    // to detect all 4 categories of pending changes.
    // Uses cmdexec.RunOption{Dir: worktreePath} to run commands in the worktree.
    // Each git command uses CheckOpt-like settings (QuietCmd, FailLevel=Debug)
    // so that failures (e.g. no upstream) are silently handled.
    func collectPendingChanges(cmdRunner *cmdexec.Runner, worktreePath string) PendingChanges
    // - git ls-files --others --exclude-standard  → UntrackedFiles
    // - git diff --name-only                       → UnstagedChanges
    // - git diff --cached --name-only              → StagedChanges
    // - git log @{upstream}..HEAD --oneline        → UnpushedCommits (失敗時は空)
    // 各出力を改行で分割し、空行を除外してスライスに格納
    ```

    ```go
    // parseLinesFromOutput splits git command output into non-empty lines.
    func parseLinesFromOutput(output string) []string
    ```

    ```go
    const maxDisplayLines = 10

    // displayPendingChanges formats and prints pending changes via logger.
    // If verbose is false, each category is truncated to maxDisplayLines.
    // Truncated output shows: "  ... and N more (M total)"
    func displayPendingChanges(logger *log.Logger, changes PendingChanges, verbose bool)
    // カテゴリごとにヘッダーを表示:
    //   "Untracked files (N):"
    //   "Unstaged changes (N):"
    //   "Staged changes (N):"
    //   "Unpushed commits (N):"
    // 各項目は "  " (インデント2) + ファイル名/コミット行
    // 0件のカテゴリは "(none)" と表示
    ```

    ```go
    // formatCategory formats a single category's items with optional truncation.
    // Returns formatted lines as a string slice.
    func formatCategory(header string, items []string, verbose bool) []string
    ```

*   **`Close()` メソッドの変更ロジック**:

    `Delete()` に委譲する **直前** の各箇所に、以下のロジックを挿入する:

    ```
    1. worktreePath := wm.Path(opts.Branch)
    2. if worktree dir exists:
    3.   changes := collectPendingChanges(r.CmdRunner, worktreePath)
    4.   if changes.TotalCount() > 0 && !opts.Yes:
    5.     displayPendingChanges(r.Logger, changes, opts.Verbose)
    6.     fmt.Fprintf(os.Stderr, "WARNING: Found %d pending change(s)... [y/N]: ", total)
    7.     scan stdin → if not "y"/"yes" → return nil (Aborted)
    8. // proceed to Delete()
    ```

    挿入箇所は2箇所:
    - Feature指定で最後のfeatureがclose → `Delete()` 呼び出し前 (line 68付近)
    - Feature未指定 → `Delete()` 呼び出し前 (line 115付近)

    両箇所を共通化するため、**ヘルパーメソッド `checkPendingChangesAndConfirm()` を定義**する:

    ```go
    // checkPendingChangesAndConfirm checks for pending changes and prompts
    // for confirmation. Returns true if the user confirms or there are no
    // pending changes or --yes is set. Returns false if user aborts.
    func (r *Runner) checkPendingChangesAndConfirm(
        opts CloseOptions, worktreePath string,
    ) bool
    ```

### cmd パッケージ

#### [MODIFY] [cmd/close.go](file://features/tt/cmd/close.go)

*   **Description**: `--verbose` フラグを追加し、`CloseOptions` に渡す
*   **Technical Design**:
    ```go
    // 新しいフラグ変数
    var closeFlagVerbose bool

    // init() に追加
    closeCmd.Flags().BoolVar(&closeFlagVerbose, "verbose", false,
        "Show all pending changes without truncation")

    // runClose() で CloseOptions に渡す
    Verbose: closeFlagVerbose,
    ```

#### [MODIFY] [cmd/delete.go](file://features/tt/cmd/delete.go)

*   **Description**: 将来的に `delete` にも同様の機能を追加する可能性があるが、本計画のスコープ外。変更なし。

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> TDD: テストを先に書き、失敗を確認してから実装を行う。

### [x] Step 1: `CloseOptions` に `Verbose` フィールドを追加

*   Edit `features/tt/internal/action/close.go`:
    *   `CloseOptions` 構造体に `Verbose bool` フィールドを追加
    *   コメント: `// show all pending changes without truncation`

### [x] Step 2: `cmd/close.go` に `--verbose` フラグを追加

*   Edit `features/tt/cmd/close.go`:
    *   `closeFlagVerbose` 変数を追加
    *   `init()` に `closeCmd.Flags().BoolVar(...)` を追加
    *   `runClose()` の `CloseOptions` 初期化に `Verbose: closeFlagVerbose` を追加

### [x] Step 3: テストを先に作成（TDD）

*   Edit `features/tt/internal/action/close_test.go`:
    *   `TestParseLinesFromOutput`: 出力文字列を行に分割するテスト
        *   空文字列 → 空スライス
        *   1行 → 1要素
        *   複数行（末尾改行あり・なし）
        *   空行を含む出力 → 空行は除外
    *   `TestFormatCategory`: カテゴリ表示の省略ロジックテスト
        *   5件 verbose=false → 全件表示
        *   10件 verbose=false → 全件表示（ちょうど上限）
        *   15件 verbose=false → 10件 + `"... and 5 more (15 total)"`
        *   15件 verbose=true → 全件表示
        *   0件 → `"(none)"` 表示
    *   `TestPendingChanges_TotalCount`: `TotalCount()` のテスト
        *   各カテゴリに異なる件数を設定し、合計が正しいことを確認
    *   既存の `TestClose_ConfirmYes_Executes` と `TestClose_ConfirmNo_Aborts` は
        **DryRunモードではgitコマンドが空文字を返す → pending changes = 0** のため
        そのまま動作する（警告プロンプトが出ないので既存テストは壊れない）

### [x] Step 4: `PendingChanges` 構造体と `TotalCount()` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `PendingChanges` 構造体を定義
    *   `TotalCount()` メソッドを実装
*   テスト実行 → `TestPendingChanges_TotalCount` が PASS することを確認

### [x] Step 5: `parseLinesFromOutput` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `parseLinesFromOutput(output string) []string` を実装
    *   `strings.Split(output, "\n")` で分割し、`strings.TrimSpace` で各行をトリム、空行を除外
*   テスト実行 → `TestParseLinesFromOutput` が PASS することを確認

### [x] Step 6: `formatCategory` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `formatCategory(header string, items []string, verbose bool) []string` を実装
    *   `maxDisplayLines = 10` 定数を定義
    *   items が 0件の場合: `["<header>:", "  (none)"]` を返す
    *   items が maxDisplayLines 以下の場合: ヘッダー + 全アイテム
    *   items が maxDisplayLines を超え verbose=false の場合: ヘッダー + 先頭10件 + `"  ... and N more (M total)"`
    *   verbose=true の場合: 常に全件表示
*   テスト実行 → `TestFormatCategory` が PASS することを確認

### [x] Step 7: `collectPendingChanges` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `collectPendingChanges(cmdRunner *cmdexec.Runner, worktreePath string) PendingChanges` を実装
    *   `cmdexec.RunOption` として `Dir: worktreePath`, `QuietCmd: true`, `FailLevelSet: true`, `FailLevel: log.LevelDebug`, `FailLabel: "SKIP"` を使用
    *   4つのgitコマンドを実行:
        1. `git ls-files --others --exclude-standard`
        2. `git diff --name-only`
        3. `git diff --cached --name-only`
        4. `git log @{upstream}..HEAD --oneline`
    *   各出力を `parseLinesFromOutput` で変換

### [x] Step 8: `displayPendingChanges` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `displayPendingChanges(logger *log.Logger, changes PendingChanges, verbose bool)` を実装
    *   ヘッダー行を出力:  `"== Pending changes in worktree =="`
    *   4つのカテゴリそれぞれに `formatCategory` を呼び出してログ出力

### [x] Step 9: `checkPendingChangesAndConfirm` を実装

*   Edit `features/tt/internal/action/close.go`:
    *   `checkPendingChangesAndConfirm(opts CloseOptions, worktreePath string) bool` を実装
    *   `opts.Yes` なら `true` を即座に返す
    *   `collectPendingChanges` を呼び出し
    *   `TotalCount() == 0` なら `true` を返す
    *   `displayPendingChanges` を呼び出し
    *   `fmt.Fprintf(os.Stderr, "WARNING: Found %d pending change(s) in worktree. Are you sure you want to delete? [y/N]: ", total)`
    *   stdinから読み取り → `"y"` / `"yes"` なら `true`、それ以外は `false`

### [x] Step 10: `Close()` メソッドにチェックを挿入

*   Edit `features/tt/internal/action/close.go`:
    *   **挿入箇所1**: Feature指定で最後のfeatureがclose → `return r.Delete(deleteOpts, wm)` の直前
    *   **挿入箇所2**: Feature未指定 → `return r.Delete(deleteOpts, wm)` の直前
    *   両箇所に以下を追加:
        ```
        worktreePath := wm.Path(opts.Branch)
        if !r.checkPendingChangesAndConfirm(opts, worktreePath) {
            r.Logger.Info("Aborted.")
            return nil
        }
        ```

### [x] Step 11: ビルドとテスト

*   `./scripts/process/build.sh` を実行してビルドと全体テストを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    全体ビルドとユニットテストの実行:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    Close関連の統合テスト:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Close"
    ```

## Documentation

本計画では既存の仕様書やドキュメントに対する更新は不要です。CLIのヘルプテキストは `--verbose` フラグの追加により自動的に更新されます。
