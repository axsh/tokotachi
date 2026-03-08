# 000-DynamicColumnWidth

> **Source Specification**: [000-DynamicColumnWidth.md](file://prompts/phases/000-foundation/ideas/fix-list-column-width/000-DynamicColumnWidth.md)

## Goal Description

`devctl list` コマンドのテーブル出力において、各カラムの幅をセル内容から動的に計算するように変更する。併せてトランケーション、`--full` フラグ、`--env` フラグ、環境変数による設定を追加する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: カラム幅の動的計算 | Proposed Changes > listing.go (`calcColumnWidths`, `FormatTable`) |
| R2: カラム間パディング（デフォルト2、環境変数で設定可能） | Proposed Changes > listing.go (`TableOptions.Padding`, `FormatTable`) |
| R3: 最大カラム幅の制限とトランケーション | Proposed Changes > listing.go (`TruncateCell`, `FormatTable`) |
| R4: `--full` オプション | Proposed Changes > list.go (`flagListFull`), listing.go (`TableOptions.Full`) |
| R5: `--env` オプション（Environment Variables セクション全体の表示制御） | Proposed Changes > list.go (`flagListEnv`, `runListBranches`) |
| R6: 環境変数の優先順位 | Proposed Changes > list.go (`runListBranches` 内の環境変数読み取り) |
| R5: `DEVCTL_LIST_*` を knownEnvVars に追加 | Proposed Changes > common.go (`knownEnvVars`) |
| 最終カラムにパディングなし | Proposed Changes > listing.go (`FormatTable` 出力ロジック) |
| ヘッダー文字列はトランケーション対象外 | Proposed Changes > listing.go (`FormatTable` ヘッダー出力) |

## Proposed Changes

### listing パッケージ

#### [MODIFY] [listing_test.go](file://features/devctl/internal/listing/listing_test.go)

*   **Description**: `FormatTable` の新シグネチャ（`TableOptions`）への対応、および動的カラム幅・トランケーションのテストケース追加
*   **Technical Design**:
    *   既存の `TestFormatTable` を `TableOptions` を使う形に更新
    *   `TestTruncateCell` を新規追加
    *   `TestFormatTable_DynamicWidth` を新規追加
*   **Logic**:

    **`TestTruncateCell`** (新規):
    ```go
    func TestTruncateCell(t *testing.T)
    // テストケース:
    // - "short" (maxWidth=40) → "short" (変化なし)
    // - "abcdefghij" (maxWidth=8) → "abcd ..." (4文字 + " ..." = 8文字)
    // - "12345678" (maxWidth=8) → "12345678" (ちょうど上限なので変化なし)
    // - "123456789" (maxWidth=8) → "1234 ..." (4文字 + " ..." = 8文字)
    // - "" (maxWidth=40) → "" (空文字列は変化なし)
    // - "abc" (maxWidth=4) → "abc" (上限未満なので変化なし)
    // - "abcde" (maxWidth=4) → " ..." (maxWidth - 4 = 0 なのでカット位置0、" ..."のみ)
    ```

    **`TestFormatTable`** (更新): 既存テストの `FormatTable(&buf, branches, false)` 呼び出しを `FormatTable(&buf, branches, listing.TableOptions{})` に変更。`showPath: true` は `TableOptions{ShowPath: true}` に変更。デフォルト値 (`MaxWidth=0`, `Padding=0`) の場合の挙動: `MaxWidth=0` → デフォルト40適用、`Padding=0` → デフォルト2適用（関数内でゼロ値チェック）。

    **`TestFormatTable_DynamicWidth`** (新規):
    ```go
    func TestFormatTable_DynamicWidth(t *testing.T)
    // サブテスト:
    //
    // "columns aligned to content width":
    //   - 入力: branches = [{Branch: "ab", Features: [{Name: "x", Status: "active"}]}, {Branch: "main", ...}]
    //   - opts: TableOptions{MaxWidth: 40, Padding: 2}
    //   - 検証: ヘッダーの "BRANCH" の幅 (6) がカラム内の最大値で使われる
    //     ("ab" は 2文字だがヘッダー "BRANCH" が 6文字なので幅は6)
    //   - 各行で2文字のパディングがカラム間にあること
    //
    // "truncation applied":
    //   - 入力: Branch名が "a]*(45文字) の BranchInfo
    //   - opts: TableOptions{MaxWidth: 40, Padding: 2}
    //   - 検証: Branch が 36文字 + " ..." にトランケーションされていること
    //
    // "full disables truncation":
    //   - 入力: 上記と同じ 45文字の Branch名
    //   - opts: TableOptions{Full: true, MaxWidth: 40, Padding: 2}
    //   - 検証: Branch が全文表示（45文字）されていること
    //
    // "custom padding":
    //   - 入力: 短い Branch 名
    //   - opts: TableOptions{MaxWidth: 40, Padding: 4}
    //   - 検証: カラム間に4文字のスペースがあること
    //
    // "last column no padding":
    //   - 入力: 短い Branch 名
    //   - opts: TableOptions{MaxWidth: 40, Padding: 2}
    //   - 検証: 最終カラム（CODE）の後ろにパディングがないこと（末尾トリム後に余分なスペースがない）
    ```

---

#### [MODIFY] [listing.go](file://features/devctl/internal/listing/listing.go)

*   **Description**: `TableOptions` 構造体の新設、`FormatTable` シグネチャ変更、動的カラム幅計算、トランケーション関数追加
*   **Technical Design**:
    ```go
    // TableOptions controls the table output format.
    type TableOptions struct {
        ShowPath bool // --path: show PATH column
        Full     bool // --full: disable truncation
        MaxWidth int  // DEVCTL_LIST_WIDTH (default: 40)
        Padding  int  // DEVCTL_LIST_PADDING (default: 2)
    }

    // TruncateCell truncates s if len(s) > maxWidth.
    // Returns s[:maxWidth-4] + " ..." when truncated.
    func TruncateCell(s string, maxWidth int) string

    // FormatTable writes branch info as a human-readable table.
    // Signature change: showPath bool → opts TableOptions
    func FormatTable(w io.Writer, branches []BranchInfo, opts TableOptions)
    ```
*   **Logic**:

    **`TruncateCell(s string, maxWidth int) string`**:
    1. `len(s) <= maxWidth` または `maxWidth <= 0` → `s` をそのまま返す
    2. `cutAt := maxWidth - 4` を計算
    3. `cutAt < 0` の場合は `cutAt = 0` にクランプ
    4. `s[:cutAt] + " ..."` を返す

    **`FormatTable` の変更ロジック**:
    1. `opts.MaxWidth` がゼロの場合、デフォルト値 `40` を適用
    2. `opts.Padding` がゼロの場合、デフォルト値 `2` を適用
    3. セルデータを事前構築:
       - 全 `branches` を走査し、各行の `[branch, feature, container, code, (path)]` セル文字列を生成
       - `opts.Full` が `false` の場合: 各セル文字列に対して `TruncateCell(cell, opts.MaxWidth)` を適用
       - ヘッダー文字列はトランケーション対象外
    4. カラム幅の計算:
       - ヘッダー配列: `["BRANCH", "FEATURE", "CONTAINER", "CODE"]`（`opts.ShowPath` の場合は `"PATH"` 追加）
       - 各カラムの幅 = `max(ヘッダー長, 全行のセル長の最大値)`
    5. 出力:
       - 最終カラム以外: `fmt.Fprintf(w, "%-*s", width+opts.Padding, cell)` で出力
       - 最終カラム: `fmt.Fprintf(w, "%s", cell)` でパディングなし出力
       - 行末に改行 `\n`

---

### cmd パッケージ

#### [MODIFY] [list.go](file://features/devctl/cmd/list.go)

*   **Description**: `--full` / `--env` フラグ追加、環境変数読み取り、`TableOptions` 構築、レポートの `EnvVars` 制御
*   **Technical Design**:
    ```go
    var (
        flagListFull bool
        flagListEnv  bool
    )

    // init() に追加:
    listCmd.Flags().BoolVar(&flagListFull, "full", false, "Disable column truncation")
    listCmd.Flags().BoolVar(&flagListEnv, "env", false, "Show environment variables in report")
    ```
*   **Logic**:

    **`runListBranches()` の変更ロジック**:
    1. 環境変数の読み取り:
       ```go
       maxWidth := 40 // default
       if v := os.Getenv("DEVCTL_LIST_WIDTH"); v != "" {
           if n, err := strconv.Atoi(v); err == nil && n > 0 {
               maxWidth = n
           }
       }
       padding := 2 // default
       if v := os.Getenv("DEVCTL_LIST_PADDING"); v != "" {
           if n, err := strconv.Atoi(v); err == nil && n >= 0 {
               padding = n
           }
       }
       ```
    2. `TableOptions` 構築:
       ```go
       opts := listing.TableOptions{
           ShowPath: flagListPath,
           Full:     flagListFull,
           MaxWidth: maxWidth,
           Padding:  padding,
       }
       ```
    3. `FormatTable` 呼び出しを変更:
       ```go
       // 変更前: listing.FormatTable(os.Stdout, branches, flagListPath)
       // 変更後:
       listing.FormatTable(os.Stdout, branches, opts)
       ```
    4. `--env` フラグによるレポート制御:
       - `flagListEnv` が `true` の場合: `CollectEnvVars()` を呼び出しレポートの `EnvVars` にセット
       - `flagListEnv` が `false` の場合: レポートに `EnvVars` を空のまま渡す（セクション非表示）
       - ただし現状 `runListBranches()` はレポートを生成していない。`--env` 時のみレポートを生成・出力する:
       ```go
       if flagListEnv {
           rep := &report.Report{
               StartTime: time.Now(),
               Branch:    "(list)",
               EnvVars:   CollectEnvVars(),
           }
           rep.Print(os.Stderr)
       }
       ```

---

#### [MODIFY] [common.go](file://features/devctl/cmd/common.go)

*   **Description**: `knownEnvVars` に `DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` を追加
*   **Technical Design**:
    ```go
    var knownEnvVars = []envVarDef{
        {"DEVCTL_EDITOR", "cursor"},
        {"DEVCTL_CMD_CODE", "code"},
        {"DEVCTL_CMD_CURSOR", "cursor"},
        {"DEVCTL_CMD_AG", "antigravity"},
        {"DEVCTL_CMD_CLAUDE", "claude"},
        {"DEVCTL_CMD_GIT", "git"},
        {"DEVCTL_CMD_GH", "gh"},
        {"DEVCTL_LIST_WIDTH", "40"},     // 追加
        {"DEVCTL_LIST_PADDING", "2"},    // 追加
    }
    ```
*   **Logic**: 既存の `CollectEnvVars()` は変更不要。`knownEnvVars` スライスに2エントリ追加するだけで、レポートの Environment Variables テーブルに自動的に含まれる。

---

### 統合テスト

#### [MODIFY] [devctl_list_code_test.go](file://tests/integration-test/devctl_list_code_test.go)

*   **Description**: `--full` / `--env` フラグの受け入れテスト、動的カラム幅の基本検証を追加
*   **Technical Design**:
    ```go
    func TestDevctlListCode_FullFlagAccepted(t *testing.T)
    func TestDevctlListCode_EnvFlagAccepted(t *testing.T)
    func TestDevctlListCode_DynamicColumnWidth(t *testing.T)
    ```
*   **Logic**:

    **`TestDevctlListCode_FullFlagAccepted`**:
    1. `runDevctl(t, "list", "--full")` を実行
    2. `exitCode == 0` を検証
    3. ヘッダーに `BRANCH`, `FEATURE`, `CONTAINER`, `CODE` が含まれることを検証

    **`TestDevctlListCode_EnvFlagAccepted`**:
    1. `runDevctl(t, "list", "--env")` を実行
    2. `exitCode == 0` を検証
    3. stderr 出力に `Environment Variables` が含まれることを検証
    4. stderr 出力に `DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` が含まれることを検証

    **`TestDevctlListCode_DynamicColumnWidth`**:
    1. `runDevctl(t, "list")` を実行
    2. ヘッダー行を取得
    3. ヘッダー行にハードコードされた大量の空白がないことを検証（固定幅24文字の空白パターンがないことを確認）
    4. 各行がカラム間に少なくとも2文字のスペースでセパレートされていることを大まかに検証

## Step-by-Step Implementation Guide

1.  **`common.go` に環境変数定義を追加**:
    *   `features/devctl/cmd/common.go` の `knownEnvVars` スライスに `{"DEVCTL_LIST_WIDTH", "40"}` と `{"DEVCTL_LIST_PADDING", "2"}` を追加

2.  **`listing_test.go` にテストを追加（TDD: RED フェーズ）**:
    *   `TestTruncateCell` を追加（`TruncateCell` はまだ存在しないのでコンパイルエラー → RED）
    *   既存の `TestFormatTable` で `FormatTable` の呼び出しを `TableOptions` に変更（コンパイルエラー → RED）
    *   `TestFormatTable_DynamicWidth` を追加

3.  **`listing.go` に `TableOptions` 構造体を追加**:
    *   `TableOptions` 構造体を定義
    *   `TruncateCell` 関数を実装
    *   `FormatTable` のシグネチャを `FormatTable(w io.Writer, branches []BranchInfo, opts TableOptions)` に変更
    *   動的カラム幅計算ロジックを実装

4.  **`list.go` にフラグと環境変数読み取りを追加**:
    *   `flagListFull`, `flagListEnv` フラグ変数を宣言し `init()` で登録
    *   `runListBranches()` 内で `DEVCTL_LIST_WIDTH`, `DEVCTL_LIST_PADDING` 環境変数を読み取り
    *   `FormatTable` 呼び出しを `TableOptions` を使う形に変更
    *   `--env` 時のレポート出力を実装

5.  **ビルド & 単体テスト実行**:
    *   `./scripts/process/build.sh` を実行し、コンパイル成功と全テストPASSを確認

6.  **統合テストを追加**:
    *   `devctl_list_code_test.go` に `TestDevctlListCode_FullFlagAccepted`, `TestDevctlListCode_EnvFlagAccepted`, `TestDevctlListCode_DynamicColumnWidth` を追加

7.  **統合テスト実行**:
    *   `./scripts/process/integration_test.sh --categories "devctl" --specify "TestDevctlListCode"` を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認項目**: コンパイル成功、既存テスト + 新規テスト (`TestTruncateCell`, `TestFormatTable_DynamicWidth`) がすべて PASS

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "TestDevctlListCode"
    ```
    *   **確認項目**: `TestDevctlListCode_FullFlagAccepted`, `TestDevctlListCode_EnvFlagAccepted`, `TestDevctlListCode_DynamicColumnWidth` がすべて PASS
    *   **Log Verification**: 各テストの stdout/stderr 出力に期待するカラムヘッダーが含まれていること

## Documentation

#### [MODIFY] [README.md](file://features/devctl/README.md)
*   **更新内容**: `## Environment Variables` セクションに `DEVCTL_LIST_WIDTH` と `DEVCTL_LIST_PADDING` を追加。`devctl list` コマンドの説明に `--full` と `--env` フラグの記述を追加。
