# 000-ListUp-BranchOverview

> **Source Specification**: [000-ListUp-BranchOverview.md](file://prompts/phases/000-foundation/ideas/feat-devctl-list-up/000-ListUp-BranchOverview.md)

## Goal Description

`devctl list` コマンドを引数なしで実行した際に、Worktree 化されたブランチ概要一覧を表示する機能を追加する。`--path` フラグで PATH カラム表示、`--json` フラグで JSON 出力をサポートする。既存の `devctl list <branch>` の動作は変更しない。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 引数なしで実行可能にする | Proposed Changes > `cmd/list.go` — Args 変更 + 分岐 |
| R2: ブランチ概要モード（worktree + state マージ） | Proposed Changes > `internal/listing/listing.go` (NEW) |
| R3: 表示フォーマット（BRANCH, FEATURES, PATH） | Proposed Changes > `internal/listing/listing.go` — `FormatTable` |
| R4: 既存 `devctl list <branch>` の後方互換 | Proposed Changes > `cmd/list.go` — `len(args)==1` パス維持 |
| R5: `--json` フラグ | Proposed Changes > `cmd/list.go` — `flagJSON` + `FormatJSON` |
| R6: `--path` フラグ | Proposed Changes > `cmd/list.go` — `flagPath` + `FormatTable` |

## Proposed Changes

### internal/listing パッケージ (新規)

ブランチ概要一覧のデータ収集・パース・フォーマットを担うパッケージを新設する。`cmd/list.go` からの分離により、ロジックの単体テストが容易になる。

#### [NEW] [listing_test.go](file://features/devctl/internal/listing/listing_test.go)

*   **Description**: `listing` パッケージの全ロジックを検証する単体テスト。TDD のため先に作成。
*   **Technical Design**:
    ```go
    package listing_test

    // --- ParseWorktreeOutput のテスト ---
    // テーブル駆動テスト
    // input: git worktree list --porcelain の出力文字列
    // expected: []WorktreeEntry のスライス

    func TestParseWorktreeOutput(t *testing.T)
    // cases:
    //   "normal output"   — 3つのworktreeを含むporcelain出力 → 3つのエントリ
    //   "empty output"    — 空文字列 → 空スライス
    //   "bare worktree"   — bare フラグ付き → entry.Bare == true

    // --- CollectBranches のテスト ---
    // worktree一覧 + stateファイル群からBranchInfo一覧を組み立てるロジック
    func TestCollectBranches(t *testing.T)
    // cases:
    //   "worktree with state"    — worktreeあり + stateあり → features が埋まる
    //   "worktree without state" — worktreeあり + stateなし → features 空, label="(no state)"
    //   "bare worktree"          — bare → label="(main worktree)"

    // --- FormatTable のテスト ---
    func TestFormatTable(t *testing.T)
    // cases:
    //   "without path" — showPath=false → PATH列が出力に含まれない
    //   "with path"    — showPath=true  → PATH列が出力に含まれる

    // --- FormatJSON のテスト ---
    func TestFormatJSON(t *testing.T)
    // cases:
    //   "normal"    — []BranchInfo → 正しいJSON配列
    //   "empty"     — 空スライス → "[]"
    ```
*   **Logic**:
    *   各テストで `porcelain` 出力の文字列リテラルを用意し、ParseWorktreeOutput に渡して結果を `assert.Equal` で検証
    *   `CollectBranches` は `[]WorktreeEntry` と `map[string]state.StateFile` を受け取る純粋関数としてテスト可能にする
    *   `FormatTable` は `bytes.Buffer` に書き込ませて文字列を検証

#### [NEW] [listing.go](file://features/devctl/internal/listing/listing.go)

*   **Description**: ブランチ概要一覧のデータ構造とロジックを定義する。
*   **Technical Design**:
    ```go
    package listing

    import (
        "encoding/json"
        "fmt"
        "io"
        "sort"
        "strings"

        "github.com/axsh/tokotachi/features/devctl/internal/state"
    )

    // WorktreeEntry represents a parsed git worktree entry.
    type WorktreeEntry struct {
        Path   string // absolute path
        Branch string // branch name (from "branch refs/heads/<name>")
        Bare   bool   // true if main worktree (bare)
    }

    // FeatureInfo holds feature name and status for display.
    type FeatureInfo struct {
        Name   string `json:"name"`
        Status string `json:"status"`
    }

    // BranchInfo holds the merged branch overview.
    type BranchInfo struct {
        Branch       string        `json:"branch"`
        Path         string        `json:"path"`
        Features     []FeatureInfo `json:"features"`
        MainWorktree bool          `json:"main_worktree,omitempty"`
    }

    // ParseWorktreeOutput parses `git worktree list --porcelain` output.
    func ParseWorktreeOutput(output string) []WorktreeEntry

    // CollectBranches merges worktree entries with state files.
    func CollectBranches(entries []WorktreeEntry, states map[string]state.StateFile) []BranchInfo

    // FormatTable writes branch info as a human-readable table.
    func FormatTable(w io.Writer, branches []BranchInfo, showPath bool)

    // FormatJSON writes branch info as JSON.
    func FormatJSON(w io.Writer, branches []BranchInfo) error
    ```
*   **Logic**:
    *   **`ParseWorktreeOutput`**:
        1. 入力文字列を空行 (`\n\n`) で分割し、各ブロックを1つの worktree エントリとして扱う
        2. 各ブロックの行を解析:
           - `worktree <path>` → `entry.Path = <path>`
           - `branch refs/heads/<name>` → `entry.Branch = <name>`
           - `bare` → `entry.Bare = true`
        3. `entry.Branch` が空で `entry.Bare == true` の場合は、main worktree と判定
        4. ソートされた `[]WorktreeEntry` を返す
    *   **`CollectBranches`**:
        1. `entries` をイテレートし、各 worktree の branch 名で `states` マップを検索
        2. state が見つかった場合、`sf.Features` をイテレートして `[]FeatureInfo` を構築
        3. state がない場合は `Features` を空スライスのまま
        4. `entry.Bare == true` の場合、`MainWorktree = true` を設定し、branch 名は git porcelain 出力から取得（なければ `HEAD` の行から判断）
        5. ブランチ名のアルファベット順でソート
    *   **`FormatTable`**:
        1. ヘッダー行を出力: `BRANCH`, `FEATURES`, (`PATH` は `showPath=true` の場合のみ)
        2. 各 `BranchInfo` を 1 行で出力
        3. Features 列: features がある場合は `name[status]` をカンマ区切りで結合。`MainWorktree==true` なら `(main worktree)`。features も MainWorktree もない場合は `(no state)`
        4. カラム幅は `%-24s %-20s` で固定（PATH は可変長）
    *   **`FormatJSON`**:
        1. `json.NewEncoder(w).Encode(branches)` でインデント付き JSON を出力
        2. 空スライスの場合は `[]` を出力（`nil` → `[]BranchInfo{}` に変換して marshal）

### state パッケージ

#### [MODIFY] [state.go](file://features/devctl/internal/state/state.go)

*   **Description**: `work/*.state.yaml` を glob で検索する `ScanStateFiles` 関数を追加。
*   **Technical Design**:
    ```go
    // ScanStateFiles finds all state files under work/ directory.
    // Returns a map of branch name -> StateFile.
    func ScanStateFiles(repoRoot string) (map[string]StateFile, error)
    ```
*   **Logic**:
    1. `filepath.Glob(filepath.Join(repoRoot, "work", "*.state.yaml"))` で state ファイル一覧を取得
    2. 各ファイルについて:
       - ファイル名から branch 名を抽出: `strings.TrimSuffix(filepath.Base(path), ".state.yaml")`
       - `Load(path)` で state を読み込み
       - エラーの場合はログ出力してスキップ（壊れたファイルで全体が止まらないように）
    3. `map[string]StateFile` を返す

#### [NEW] [state_scan_test.go](file://features/devctl/internal/state/state_scan_test.go)

*   **Description**: `ScanStateFiles` の単体テスト。
*   **Technical Design**:
    ```go
    func TestScanStateFiles(t *testing.T)
    // cases:
    //   "no files"      — 空ディレクトリ → 空マップ
    //   "one file"      — 1つのstateファイル → 1エントリのマップ
    //   "multiple files" — 2つのstateファイル → 2エントリのマップ
    ```
*   **Logic**:
    *   `t.TempDir()` を使い、`work/` サブディレクトリを作成
    *   `state.Save` でテスト用 state ファイルを書き込み
    *   `ScanStateFiles` を呼び出し、結果のマップサイズとキーを `assert.Equal` で検証

### cmd パッケージ

#### [MODIFY] [list.go](file://features/devctl/cmd/list.go)

*   **Description**: cobra 引数制約の変更、分岐の追加、`--json`/`--path` フラグの追加。
*   **Technical Design**:
    ```go
    var (
        flagListJSON bool
        flagListPath bool
    )

    var listCmd = &cobra.Command{
        Use:   "list [branch]",          // <branch> → [branch] に変更
        Short: "List branches or features",
        Long:  "Without arguments, list all worktree branches. With a branch, list features for that branch.",
        Args:  cobra.MaximumNArgs(1),    // ExactArgs(1) → MaximumNArgs(1)
        RunE:  runList,
    }

    func init() {
        listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
        listCmd.Flags().BoolVar(&flagListPath, "path", false, "Show worktree path column")
    }
    ```
*   **Logic**:
    *   `runList` で `len(args)` を判定:
        *   `0` → `runListBranches(cmd)` を呼び出し
        *   `1` → 既存ロジック（`InitContext` → state 読み込み → feature テーブル）
    *   `runListBranches(cmd *cobra.Command) error`:
        1. `repoRoot` を `os.Getwd()` で取得
        2. `logger` / `cmdRunner` を独自に初期化（`InitContext` を使わない。branch が不要なため）
        3. `git worktree list --porcelain` を `cmdRunner.Run(...)` で実行
        4. `listing.ParseWorktreeOutput(output)` でパース
        5. `state.ScanStateFiles(repoRoot)` で state マップ取得
        6. `listing.CollectBranches(entries, states)` でマージ
        7. `flagListJSON` が true → `listing.FormatJSON(os.Stdout, branches)`
        8. `flagListJSON` が false → `listing.FormatTable(os.Stdout, branches, flagListPath)`
    *   既存の `runList` 内 `len(args) == 1` パスにも `--json` 対応を追加:
        *   `flagListJSON` が true の場合、feature 一覧を JSON で出力

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> TDD 方針に従い、各ステップでテストを先に作成してから実装します。

### Phase 1: `state.ScanStateFiles` の追加

- [x] **Step 1**: テスト作成
    *   [state_scan_test.go](file://features/devctl/internal/state/state_scan_test.go) を新規作成
    *   `TestScanStateFiles` テーブル駆動テスト（no files / one file / multiple files）を実装
    *   `./scripts/process/build.sh` でテストが FAIL することを確認

- [x] **Step 2**: 実装
    *   [state.go](file://features/devctl/internal/state/state.go) に `ScanStateFiles` 関数を追加
    *   `./scripts/process/build.sh` でテストが PASS することを確認

### Phase 2: `listing` パッケージの新規作成

- [x] **Step 3**: テスト作成
    *   [listing_test.go](file://features/devctl/internal/listing/listing_test.go) を新規作成
    *   `TestParseWorktreeOutput`、`TestCollectBranches`、`TestFormatTable`、`TestFormatJSON` を実装
    *   `./scripts/process/build.sh` でテストが FAIL することを確認

- [x] **Step 4**: 実装 — パーサーとデータ構造
    *   [listing.go](file://features/devctl/internal/listing/listing.go) を新規作成
    *   `WorktreeEntry`, `FeatureInfo`, `BranchInfo` 構造体を定義
    *   `ParseWorktreeOutput` を実装
    *   `TestParseWorktreeOutput` が PASS することを確認

- [x] **Step 5**: 実装 — コレクターとフォーマッター
    *   `CollectBranches`, `FormatTable`, `FormatJSON` を実装
    *   `./scripts/process/build.sh` で全テストが PASS することを確認

### Phase 3: `cmd/list.go` の変更

- [x] **Step 6**: `cmd/list.go` を変更
    *   cobra args を `MaximumNArgs(1)` に変更
    *   `--json`, `--path` フラグを `init()` で追加
    *   `runList` に `len(args)` 分岐を追加
    *   `runListBranches` 関数を実装
    *   既存パス (`len(args)==1`) に `--json` 対応を追加

- [x] **Step 7**: ビルド検証
    *   `./scripts/process/build.sh` で全体ビルドと単体テストを実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: `devctl list` の使い方に「引数なしでブランチ一覧表示」「`--json` フラグ」「`--path` フラグ」の説明を追加
