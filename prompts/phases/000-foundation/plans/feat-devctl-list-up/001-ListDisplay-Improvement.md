# 001-ListDisplay-Improvement

> **Source Specification**: [001-ListDisplay-Improvement.md](file://prompts/phases/000-foundation/ideas/feat-devctl-list-up/001-ListDisplay-Improvement.md)

## Goal Description

`devctl list` コマンドのテーブル出力を改善する。`FEATURES` カラムを `FEATURE` に名称変更し、新たに `STATE` カラムを追加して、feature 名とステータスを分離表示する。JSON 出力には変更を加えない。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `FEATURES` → `FEATURE` に名称変更 | Proposed Changes > `listing.go` — `FormatTable` ヘッダー変更 |
| R2: `STATE` カラムを新設 | Proposed Changes > `listing.go` — `FormatTable` ヘッダー + `stateColumn` 関数 |
| R3: 表示内容の配置変更（feature名/state 分離） | Proposed Changes > `listing.go` — `featureColumn` + `stateColumn` 関数 |
| R4: `--path` フラグの動作維持 | Proposed Changes > `listing.go` — `FormatTable` の PATH カラム位置調整 |
| R5: JSON 出力は変更しない | 変更なし（`FormatJSON` / `BranchInfo` は既存のまま） |

## Proposed Changes

### listing パッケージ

テーブル表示ロジックの変更のみ。データ構造（`BranchInfo`, `FeatureInfo`）は変更しない。

#### [MODIFY] [listing_test.go](file://features/devctl/internal/listing/listing_test.go)

*   **Description**: `TestFormatTable` のアサーションを新カラム構成に合わせて更新する。TDD のため先に修正。
*   **Technical Design**:
    ```go
    // TestFormatTable の "without path" サブテスト
    t.Run("without path", func(t *testing.T) {
        var buf bytes.Buffer
        listing.FormatTable(&buf, branches, false)
        out := buf.String()
        // ヘッダー検証
        assert.Contains(t, out, "FEATURE")     // 単数形のカラム名
        assert.NotContains(t, out, "FEATURES") // 複数形は存在しない
        assert.Contains(t, out, "STATE")       // 新カラム
        assert.NotContains(t, out, "PATH")     // --path なし

        // ボディ検証: feature 名と state が分離されている
        assert.Contains(t, out, "devctl")          // FEATURE カラム
        assert.Contains(t, out, "active")          // STATE カラム
        assert.NotContains(t, out, "devctl[active]") // 旧形式は存在しない
        assert.Contains(t, out, "(main worktree)")   // main worktree の STATE
    })

    // TestFormatTable の "with path" サブテスト
    t.Run("with path", func(t *testing.T) {
        var buf bytes.Buffer
        listing.FormatTable(&buf, branches, true)
        out := buf.String()
        // ヘッダー検証
        assert.Contains(t, out, "FEATURE")
        assert.Contains(t, out, "STATE")
        assert.Contains(t, out, "PATH")
        // ボディ検証
        assert.Contains(t, out, "/repo/work/feat-a")
        assert.Contains(t, out, "/repo")
    })
    ```
*   **Logic**:
    *   `FEATURES` への assert を `FEATURE` + `NotContains("FEATURES")` に変更
    *   `STATE` カラムの存在確認を追加
    *   `devctl[active]` が出力に含まれ**ない**ことを確認（分離表示の検証）
    *   `devctl` と `active` が個別に含まれることを確認

#### [MODIFY] [listing.go](file://features/devctl/internal/listing/listing.go)

*   **Description**: `featuresLabel` 関数を `featureColumn` と `stateColumn` の2関数に分割し、`FormatTable` のヘッダーとボディを3カラム（+ PATH）構成に変更する。
*   **Technical Design**:
    ```go
    // featureColumn builds a display string for the FEATURE column.
    // Returns feature names (comma-separated if multiple), or "-" if none.
    func featureColumn(bi BranchInfo) string

    // stateColumn builds a display string for the STATE column.
    // Returns the status of features, "(no state)", or "(main worktree)".
    func stateColumn(bi BranchInfo) string

    // FormatTable writes branch info as a human-readable table.
    // Columns: BRANCH, FEATURE, STATE, (PATH if showPath=true)
    func FormatTable(w io.Writer, branches []BranchInfo, showPath bool)
    ```
*   **Logic**:
    *   **`featureColumn(bi BranchInfo) string`**:
        1. `bi.MainWorktree == true` → `"-"` を返す
        2. `len(bi.Features) == 0` → `"-"` を返す
        3. features がある場合 → feature 名をカンマ区切りで結合して返す（例: `"devctl"`, `"devctl, editor"`）
    *   **`stateColumn(bi BranchInfo) string`**:
        1. `bi.MainWorktree == true` → `"(main worktree)"` を返す
        2. `len(bi.Features) == 0` → `"(no state)"` を返す
        3. features がある場合 → 全 feature の status をカンマ区切りで結合して返す（例: `"active"`, `"active, inactive"`）
    *   **`FormatTable` の変更**:
        1. ヘッダー行: `BRANCH`, `FEATURE`, `STATE`, (`PATH`)
        2. ボディ行: `bi.Branch`, `featureColumn(bi)`, `stateColumn(bi)`, (`bi.Path`)
        3. フォーマット幅:
            - `showPath=true`: `"%-24s %-20s %-20s %s\n"` (BRANCH, FEATURE, STATE, PATH)
            - `showPath=false`: `"%-24s %-20s %s\n"` (BRANCH, FEATURE, STATE)
    *   **`featuresLabel` 関数は削除する**（`featureColumn` と `stateColumn` に置き換わるため）

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> TDD 方針に従い、テストを先に修正してから実装を変更します。

### Phase 1: テストの更新（Red）

- [x] **Step 1**: テスト修正
    *   [listing_test.go](file://features/devctl/internal/listing/listing_test.go) の `TestFormatTable` を修正:
        *   `"without path"` サブテスト:
            - `assert.Contains(t, out, "FEATURES")` → `assert.Contains(t, out, "FEATURE")` + `assert.NotContains(t, out, "FEATURES")`
            - `assert.Contains(t, out, "STATE")` を追加
            - `assert.Contains(t, out, "devctl[active]")` → `assert.NotContains(t, out, "devctl[active]")` に変更
            - `assert.Contains(t, out, "devctl")` と `assert.Contains(t, out, "active")` を追加
        *   `"with path"` サブテスト:
            - ヘッダーに `STATE` の検証を追加
    *   `./scripts/process/build.sh` でテストが FAIL することを確認

### Phase 2: 実装の変更（Green）

- [x] **Step 2**: `listing.go` を修正
    *   `featuresLabel` 関数を削除
    *   `featureColumn` 関数を追加（上記 Logic 参照）
    *   `stateColumn` 関数を追加（上記 Logic 参照）
    *   `FormatTable` 関数のヘッダーとボディを3カラム構成に変更
    *   `./scripts/process/build.sh` で全テストが PASS することを確認

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

#### [MODIFY] [001-ListDisplay-Improvement.md](file://prompts/phases/000-foundation/ideas/feat-devctl-list-up/001-ListDisplay-Improvement.md)
*   **更新内容**: 実装後、出力イメージが実際の実装と一致しているか確認し、必要に応じて更新
