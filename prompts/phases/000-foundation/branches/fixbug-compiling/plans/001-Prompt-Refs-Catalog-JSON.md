# 001-Prompt-Refs-Catalog-JSON

> **Source Specification**: `prompts/phases/000-foundation/branches/fixbug-compiling/ideas/001-Prompt-Refs-Catalog-JSON.md`

## Goal Description

`tt prompt refs` サブコマンドを追加し、policy / procedure / capability ソースについて `{file, kind, id, ref}` の JSON カタログを stdout に出す。emit や解決済み相対パスは出さず、既存 `compile --dry-run` は変更しない。

## User Review Required

None.（仕様の推奨案 A を採用。任意要件 R-opt1/2/3 は本計画では先送り）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `tt prompt refs`（案 A） | Proposed Changes > `cmd/prompt.go` + `scripts/code/prompt/refs.sh` |
| R2: policy/procedure/capability のみ | Proposed Changes > `compiler/refs.go` の kind フィルタ |
| R3: JSON スキーマ `{refs:[{file,kind,id,ref}]}` | Proposed Changes > `RefEntry` / `RefsCatalog` 構造体 |
| R4: `ref` はパスオプション非依存 | Verification > 単体は純関数、統合で workspace 明示でも `ref` 不変 |
| R5: `LoadConfig` + `ParseAllEntities` のみ | Proposed Changes > `ListRefs` |
| R6: 既定は全発見（タグ未適用） | `ListRefs` はタグ処理しない。R-opt1 先送り |
| R7: 成功 0 / 致命的エラー非 0 | parse エラーが 1 件でも非 0（下記 Logic） |
| 既存 dry-run 非破壊 | `runPromptCompile` を変更しない |
| 検証シナリオ 1–7 | Verification Plan > `tt_prompt_refs_test.go` + PromptTags リグレッション |
| R-opt1 `--selected-only` | **先送り**（本計画外） |
| R-opt2 `target_vars` | **先送り** |
| R-opt3 `--pretty` | **先送り**（コンパクト JSON のみ） |

## Proposed Changes

> TDD: `_test.go` を先に書き、失敗を確認してから実装する。

### compiler（純関数 + 一覧パイプライン）

#### [NEW] [features/tt/internal/prompt/compiler/refs_test.go](file://features/tt/internal/prompt/compiler/refs_test.go)

*   **Description**: `BuildRefsCatalog` / `ListRefs` のテーブル駆動テスト（TDD 先頭）。
*   **Technical Design**:
    *   `TestBuildRefsCatalog_FiltersAndSorts`
    *   `TestBuildRefsCatalog_RefUsesKindAndID`
    *   `TestBuildRefsCatalog_IgnoresNonRefKinds`（target/guard 等を除外）
    *   `TestListRefs_ParseErrorFails`（不正 md がある fixture で error）
    *   `TestListRefs_HappyPath`（`testdata/valid` または最小 temp fixture）
*   **Logic（断言例）**:
    *   Entity `{Kind:policy, ID:coding-rules, FilePath:.../coding-rules.md}` → `file=coding-rules.md`, `ref={{policy:coding-rules}}`
    *   並び: kind 昇順（文字列比較で `capability` < `policy` < `procedure`）、同 kind 内は id 昇順
    *   `Kind:target` / `Kind:guard` / `Kind:worker` / `Kind:bundle` / `Kind:skip` は `refs` に現れない
    *   JSON に `.claude/` や `SKILL.md` 配置パス文字列を含まない（`file` はベース名のみ）

#### [NEW] [features/tt/internal/prompt/compiler/refs.go](file://features/tt/internal/prompt/compiler/refs.go)

*   **Description**: 参照カタログの型定義と生成ロジック。
*   **Technical Design**:

```go
package compiler

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// RefEntry is one catalog row for a file-backed template reference.
type RefEntry struct {
	File string `json:"file"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Ref  string `json:"ref"`
}

// RefsCatalog is the stdout JSON document for `tt prompt refs`.
type RefsCatalog struct {
	Refs []RefEntry `json:"refs"`
}

// BuildRefsCatalog maps parsed entities to a stable refs list.
// Only policy, procedure, and capability are included.
func BuildRefsCatalog(entities []*manifest.Entity) RefsCatalog

// ListRefs loads project config, parses entities, and builds the catalog.
// Any parse/validation error from ParseAllEntities causes a non-nil error (R7).
func ListRefs(paths *PathConfig) (*RefsCatalog, error)
```

*   **Logic**（`BuildRefsCatalog`）:
    1. 空の `[]RefEntry` を用意する。
    2. 各 `entity` について `kind := entity.Kind` が `"policy"` / `"procedure"` / `"capability"` のいずれでもなければ skip。
    3. `id := entity.ID`。空 id はここに到達しない想定（`ParseEntity` が既に拒否）。防御として空なら skip せず、呼び出し側 `ListRefs` で error にする。
    4. `file := filepath.Base(entity.FilePath)`（パス区切りを含めない）。
    5. `ref := "{{" + kind + ":" + id + "}}"`（文字列連結。パスや target を混ぜない）。
    6. スライスに append。
    7. `sort.SliceStable`: 第一キー `Kind` 昇順、第二キー `ID` 昇順。
    8. `RefsCatalog{Refs: entries}` を返す（`Refs` が nil のときは空スライスにして `[]` を出す）。

*   **Logic**（`ListRefs`）:
    1. `cfg, err := LoadConfig(paths.ProjectYAML)`。失敗なら return error。
    2. `entities, parseErrors := manifest.ParseAllEntities(cfg, paths.Workspace)`。
    3. `len(parseErrors) > 0` なら、エラーメッセージをまとめて `fmt.Errorf("prompt refs: N parse error(s): ...")` を返し、カタログを返さない（R7: 部分成功しない）。
    4. フィルタ対象 kind で `entity.ID == ""` があれば `fmt.Errorf("prompt refs: empty id in %s", entity.FilePath)`。
    5. `catalog := BuildRefsCatalog(entities)` を `&catalog` で返す。
    6. `ApplyTags` / Emit / Digest / ファイル書き込みは呼ばない。

### cmd（CLI）

#### [MODIFY] [features/tt/cmd/prompt.go](file://features/tt/cmd/prompt.go)

*   **Description**: `tt prompt refs` を登録する。既存 compile/deploy/update のフラグ・ロジックは触らない。
*   **Technical Design**:

```go
import (
	"encoding/json"
	// existing imports...
)

var refsProject string

var promptRefsCmd = &cobra.Command{
	Use:   "refs",
	Short: "List prompt template refs as JSON (file, kind, id, ref)",
	Long:  "List file-backed template references as JSON. Refs are logical IDs and do not depend on --target or deploy paths." + promptPathLongSuffix,
	RunE:  runPromptRefs,
}

func runPromptRefs(cmd *cobra.Command, args []string) error {
	pathOpts, err := buildPathOptions(cmd, refsProject)
	if err != nil {
		return err
	}
	paths, err := compiler.ResolvePaths(pathOpts)
	if err != nil {
		return err
	}
	catalog, err := compiler.ListRefs(paths)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	return enc.Encode(catalog)
}
```

*   **Logic**:
    1. `init()` 内で `addPromptPathFlags(promptRefsCmd)` と `--project`（`refsProject`、デフォルト `prompts/manifest/project.yaml`）。
    2. `--target` / `--tags` / `--tag-refs` / `--pretty` は付けない（第1版）。
    3. `promptCmd.AddCommand(promptRefsCmd)`。
    4. 成功時は JSON のみ stdout（進捗メッセージなし）。エラーは cobra 経由で stderr、非 0。
    5. `runPromptCompile` / `compileDryRun` 分岐は変更しない（既存 dry-run 非破壊）。

### scripts（wrapper）

#### [NEW] [scripts/code/prompt/refs.sh](file://scripts/code/prompt/refs.sh)

*   **Description**: `compile.sh` / `deploy.sh` / `update.sh` と同型の薄いラッパー。
*   **Logic**:

```bash
#!/usr/bin/env bash
# scripts/code/prompt/refs.sh -- tt prompt refs wrapper
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../_resolve_tool.sh"

exec "$TOOL" prompt refs "$@"
```

*   実行ビットを付与する（他 wrapper と同様）。

### 統合テスト（CLI）

#### [NEW] [tests/tt/tt_prompt_refs_test.go](file://tests/tt/tt_prompt_refs_test.go)

*   **Description**: `tt prompt refs` の CLI 検証（GUI E2E 不要）。`tests/tt/tt_prompt_template_vars_test.go` と同様に build tag なし・`runTTInDir` 利用。
*   **Technical Design**: catalog_template fixture（`copyCatalogTemplateFixture` を再利用、または同ヘルパーをこのファイルへ移さないで共通化済みなら呼ぶ）。
*   **テストケース**:

| 関数 | 検証内容（検証シナリオ対応） |
| :--- | :--- |
| `TestPromptRefs_JSONContainsPolicyAndProcedure` | stdout を JSON unmarshal。`coding-rules` → `ref={{policy:coding-rules}}`、`build-pipeline` → `ref={{procedure:build-pipeline}}`。出力全体に `.claude/rules/` / `.agent/workflows/` を含まない（シナリオ 1–5） |
| `TestPromptRefs_RefStableWithWorkspaceFlag` | 同一 fixture でデフォルト実行と `--workspace <tmp>` 明示の `ref` 集合が同一（シナリオ 6 / R4） |
| `TestPromptRefs_MissingProjectFails` | project.yaml 無し（空 dir 等）で exit code != 0（R7） |
| （既存）`PromptTags` / dry-run | 変更しない。リグレッションで `PromptTags` が通ることで dry-run YAML が残ることを確認（シナリオ 7） |

*   **Logic（断言の具体化）**:
    1. `stdout, stderr, code := runTTInDir(t, workspace, "prompt", "refs")`
    2. `require.Equal(t, 0, code, stderr)`
    3. `var catalog struct { Refs []struct{ File, Kind, ID, Ref string } }; json.Unmarshal([]byte(stdout), &catalog)`
    4. `ref` マップを作り、期待キーが存在することを assert
    5. `assert.NotContains(t, stdout, ".claude/")` および `".agent/workflows"`

## Step-by-Step Implementation Guide

1. **Failing unit tests (TDD)**:
    *   Add `features/tt/internal/prompt/compiler/refs_test.go` with the cases above.
    *   Confirm fail with `./scripts/process/build.sh --skip-frontend --skip-etc`（Windows では `--skip-etc` 不要なら通常の build でも可）。
2. **Implement refs.go**:
    *   Add types + `BuildRefsCatalog` + `ListRefs` in `features/tt/internal/prompt/compiler/refs.go`.
    *   Re-run build until unit tests pass.
3. **CLI wiring**:
    *   Add `promptRefsCmd` / `runPromptRefs` / `AddCommand` / `refsProject` in `features/tt/cmd/prompt.go`.
    *   Add `scripts/code/prompt/refs.sh`（executable）。
4. **Integration tests**:
    *   Add `tests/tt/tt_prompt_refs_test.go`.
5. **Run Verification Plan** below and record §12 総合判定.

## Verification Plan

### テスト項目設計（§11）

#### ボトムアップ順序

| Step | 層 | テスト | 確認すること |
| :---: | :--- | :--- | :--- |
| 1 | C: `BuildRefsCatalog` | `refs_test.go` | フィルタ・ソート・`ref` 文字列・非対象 kind 除外 |
| 2 | B: `ListRefs` | `refs_test.go` | LoadConfig+Parse、parse エラーで失敗 |
| 3 | A: CLI | `tt_prompt_refs_test.go` | JSON 出力と R4・シナリオ 1–6 |

#### 観点チェックリスト（§11.3）

| # | 観点 | カバー |
| :---: | :--- | :--- |
| 1 | 正常系 | 既知 id の `ref` が JSON に存在 |
| 2 | 異常系 | project 欠落 / parse エラーで非 0 |
| 3 | 外部連携 | 実 FS fixture |
| 4 | データ一貫性 | `file`=Base(FilePath), `ref`=kind+id |
| 5 | 状態遷移 | 書き込み副作用なし（deploy 成果物が増えない） |
| 6 | 設定反映 | path フラグで発見根は変わり得るが `ref` 不変 |
| 7 | 副作用 | stdout にログ混入なし（JSON のみ） |

#### セルフレビュー結果（§11.4）

1. **網羅性**: 必須 R1–R7 と検証シナリオ 1–7 をトレース。任意 R-opt1/2/3 は先送り明記。
2. **証拠**: JSON unmarshal してフィールド検証。解決パス非含有を NotContains。
3. **迂回排除**: compile dry-run を変更せず、refs 専用パスで検証。dry-run を refs 代替としない。
4. **依存**: 純関数 → ListRefs → CLI の順。

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "PromptRefs"
    ```
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "PromptTags|PromptTemplateVars"
    ```
    *   **Log Verification**: `TestPromptRefs_*` PASS。意図しない SKIP/WARN なし。既存 dry-run 系が壊れていないこと。

3. **E2E Tests**:
    *   GUI E2E は不要（CLI のみの変更のため）。
    *   CLI 動作確認は `tests/tt/tt_prompt_refs_test.go` でコード化する（手動 `tt prompt refs` だけでは代替にならない）。

### 総合判定プロセス（§12）

全テスト成功後、§12.3 フォーマットで総合判定を記録する。実行時チェック項目:

| # | チェック | 内容 |
| :---: | :--- | :--- |
| 1 | スキップ | PromptRefs が build tag で除外されていないか |
| 2 | 部分エラー | parse エラーを握りつぶして空 JSON 成功していないか |
| 3 | 迂回 | compile dry-run を refs 相当と誤認していないか |
| 4 | 設定 | `--target` 無しで動作するか |
| 5 | 順序依存 | `--specify PromptRefs` 単独で通るか |
| 6 | カバレッジ | 新機能テストと `refs.sh` の存在 |
| 7 | 外部状態 | temp fixture のみ（本番 deploy 不要） |

## Documentation

*   `prompts/specifications` に独立仕様が無ければ追加不要。
*   正本は Source Specification と本計画。
*   README の prompt サブコマンド一覧があれば 1 行追加してよい（任意・本計画必須ではない）。

---

### 総合判定結果

**判定**: ✅ 動作確認完了

#### テスト結果サマリ
- Build: `./scripts/process/build.sh` PASS
- Integration `PromptRefs`: 3 PASS
- Integration `PromptTags|PromptTemplateVars`: 9 PASS（リグレッション）
- 失敗: 0
- 事実上スキップ: 0

#### チェック項目の結果
| # | チェック項目 | 結果 | 備考 |
|---|------------|------|------|
| 1 | スキップされたテスト | ✅ | build tag なし。PromptRefs 3 件すべて実行 |
| 2 | 部分的なエラー | ✅ | parse エラー時は非 0、カタログ非返却 |
| 3 | 迂回処理による偽成功 | ✅ | refs 専用パス。dry-run 未変更 |
| 4 | アダプタ・コンフィグ | ✅ | `--target` 無しで JSON 出力 |
| 5 | テスト間の依存・順序 | ✅ | `--specify PromptRefs` 単独 PASS |
| 6 | カバレッジ | ✅ | refs.go / CLI / refs.sh / 統合テスト |
| 7 | 外部システムの状態 | ✅ | temp fixture のみ |

#### 判定理由
単体・CLI 統合・既存 prompt リグレッションがすべて成功し、解決パス非出力と exit コード要件もテストで固定できたため。
