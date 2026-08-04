# 000-Manifest-Tags

> **Source Specification**: prompts/phases/000-foundation/branches/feat-tags/ideas/000-Manifest-Tags.md

## Goal Description

`prompts/manifest/code_content/` 配下の Manifest に選択タグ `tags` を導入し、`prompt compile` / `deploy` / `update` の `--tags` / `--tag-refs`（および `TT_TAGS` / `TT_TAG_REFS`）で compile・deploy・update 対象を OR 条件で取捨選択できるようにする。暗黙タグは `baseline`。memory / safety / targets は常に対象。カタログとプロジェクト作業ツリーのスキーマ・Manifest を同期更新する。

## User Review Required

None.（仕様レビューで方針確定済み。実装時の命名 `--tag-refs` / 定数配置は本計画に従う。）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| code_content に `tags`（capability/policy/procedure/skip） | Proposed Changes > schemas + Entity EffectiveTags |
| safety/targets はタグ対象外、memory 常時対象 | Proposed Changes > tags.go `IsTaggableKind` / ApplyTagSelection |
| 暗黙 `baseline`、明示時は置換 | Proposed Changes > `EffectiveTags` |
| `--tags` を compile/deploy/update に追加（OR） | Proposed Changes > prompt.go + Options 伝播 |
| `TT_TAGS`（CLI > env > baseline） | Proposed Changes > prompt.go `resolveTagsFlag` |
| 後方互換（省略時 = baseline） | EffectiveTags + CLI デフォルト |
| カタログ＋プロジェクトに `tags: [baseline]` 明示 | Proposed Changes > md 一括更新 |
| スカラー/配列、空配列禁止、kebab-case 強制 | schemas oneOf + EffectiveTags 検証 |
| CLI トリム・重複 Warning | `NormalizeRequestedTags` |
| `--tag-refs include\|strict` + `TT_TAG_REFS` | `ApplyTagSelection` + prompt.go |
| resolved 全件 + `selected` | Entity.Selected + MarshalYAML + emitters |
| digest に RequestedTags + TagRefs | `ComputeSourceDigest` 拡張 |
| 旧 Classification `tags` 廃止 | policy/skip schema 置換 |

## Proposed Changes

### 0. Constants & Tag Selection Core (`features/tt/internal/prompt/manifest`)

#### [NEW] [tags.go](file://features/tt/internal/prompt/manifest/tags.go)
*   **Description**: タグ定数・正規化・EffectiveTags・選択適用の中核。
*   **Technical Design**:
    ```go
    package manifest

    const (
        BaselineTag       = "baseline"
        TagRefsInclude    = "include"
        TagRefsStrict     = "strict"
        EnvKeyTags        = "TT_TAGS"
        EnvKeyTagRefs     = "TT_TAG_REFS"
        tagNamePattern    = `^[a-z0-9]+(-[a-z0-9]+)*$`
    )

    // IsTaggableKind reports whether kind is subject to --tags filtering.
    // policy / procedure / capability / skip are taggable (code_content).
    // guard / worker / bundle / target are not.
    func IsTaggableKind(kind string) bool

    // NormalizeRequestedTags parses a comma-separated tags string.
    // - Trims each element
    // - Drops empty segments after trim
    // - Deduplicates (first-seen order preserved); duplicate removals append a warning
    // - Validates kebab-case; invalid name returns error
    // - Empty input after parse returns error (caller should not pass empty; CLI uses baseline default before call)
    func NormalizeRequestedTags(raw string) (tags []string, warnings []string, err error)

    // NormalizeTagRefsMode returns include|strict.
    // Empty → include. Unknown → error.
    func NormalizeTagRefsMode(raw string) (string, error)

    // EffectiveTags returns the effective tag set for an entity.
    // - If Raw has no "tags" key → []string{BaselineTag}
    // - If tags is string → normalize to one-element slice (kebab-case check)
    // - If tags is []any → require len>=1, each kebab-case string
    // - Other types / empty array → error
    func EffectiveTags(e *Entity) ([]string, error)

    // tagMatched reports EffectiveTags ∩ requested ≠ ∅
    func tagMatched(effective, requested []string) bool

    // referencedCapabilityIDs returns uses_capabilities IDs from e.Raw (same parsing as validator).
    func referencedCapabilityIDs(e *Entity) []string

    // ApplyTagSelection sets Entity.Selected on all entities.
    // Steps:
    // 1. For !IsTaggableKind: Selected = true
    // 2. For taggable: compute EffectiveTags; TagMatched = intersection with requested
    // 3. Initial selected = TagMatched for taggable
    // 4. mode include: BFS/DFS from selected taggable entities via uses_capabilities;
    //    mark referenced capabilities Selected=true even if not TagMatched
    // 5. mode strict: for each selected (TagMatched) entity, every uses_capabilities
    //    target must be TagMatched; else append ValidationError
    // 6. Returns validation errors (strict failures and EffectiveTags parse errors)
    func ApplyTagSelection(entities []*Entity, requested []string, mode string) []ValidationError
    ```
*   **Logic**:
    *   `EffectiveTags`: `_, ok := e.Raw["tags"]` で省略判定（値が `nil` でもキー有りなら明示扱い。ただし YAML で `tags:` のみは null になり得る → その場合はバリデーションエラー）。
    *   スカラー: `tags: test` → `Raw["tags"]` が `string`。
    *   配列: YAML デコード後 `[]any`。各要素を `fmt.Sprintf("%v", …)` ではなく型アサーション `string` で取得し、非 string はエラー。
    *   include 閉包: capability ID → Entity マップを構築し、キューで到達可能な capability を selected にする。既に selected な非 capability からの参照のみ起点（初期 TagMatched 集合）。
    *   strict: TagMatched なエンティティの参照先が TagMatched でない場合エラー（参照先が存在しない場合は既存 `ValidateReferences` が先に検出）。

#### [NEW] [tags_test.go](file://features/tt/internal/prompt/manifest/tags_test.go)
*   **Description**: TDD 用テーブル駆動テスト（実装前に失敗することを確認）。
*   **Technical Design** — 最低限のケース:
    | Test | Input | Expect |
    |---|---|---|
    | `TestEffectiveTags_Omitted` | Raw without tags | `[baseline]` |
    | `TestEffectiveTags_Scalar` | `tags: "test"` | `[test]` |
    | `TestEffectiveTags_Array` | `[baseline, test]` | both |
    | `TestEffectiveTags_EmptyArray` | `[]` | error |
    | `TestEffectiveTags_InvalidName` | `Foo` / `a_b` | error |
    | `TestNormalizeRequestedTags_TrimDedup` | `"baseline, test, baseline"` | tags=`[baseline,test]`, 1 warning |
    | `TestNormalizeTagRefsMode` | `""`→include, `strict`→strict, `foo`→err |
    | `TestApplyTagSelection_OR` | A baseline, B test; requested `[test]` | B selected, A not |
    | `TestApplyTagSelection_IncludeClosure` | P(baseline) uses X(test); requested baseline, include | P+X selected |
    | `TestApplyTagSelection_StrictError` | same; strict | error |
    | `TestApplyTagSelection_NonTaggableAlways` | guard/worker/bundle/target | always selected |
    | `TestIsTaggableKind` | policy/procedure/capability/skip true; others false |

### 1. Entity Selected Field

#### [MODIFY] [types.go](file://features/tt/internal/prompt/manifest/types.go)
*   **Description**: `Selected` を Entity に追加し、YAML roundtrip 対応。
*   **Technical Design**:
    ```go
    type Entity struct {
        APIVersion string         `yaml:"apiVersion"`
        Kind       string         `yaml:"kind"`
        ID         string         `yaml:"id"`
        Title      string         `yaml:"title"`
        Selected   bool           `yaml:"selected"`
        FilePath   string         `yaml:"-"`
        Raw        map[string]any `yaml:"-"`
    }
    ```
*   **Logic**:
    *   `UnmarshalYAML`: `e.Selected = aux.Selected` を追加（CheckDrift が resolved YAML を読むときに必要）。
    *   `MarshalYAML`: `raw["selected"] = e.Selected` を必ず書き出す（resolved 全件＋印）。
    *   ソース Manifest の frontmatter に `selected` は置かない（スキーマ `additionalProperties: false` により拒否される）。`Selected` はパイプラインが付与する実行時／resolved 専用フィールド。

#### [MODIFY] [types_test.go](file://features/tt/internal/prompt/manifest/types_test.go)
*   **Description**: Marshal/Unmarshal で `selected: true|false` が保持されることを検証。

### 2. Compile Pipeline Hook

#### [MODIFY] [compiler.go](file://features/tt/internal/prompt/compiler/compiler.go)
*   **Description**: Options に Tags/TagRefs を追加し、検証後〜Resolve 前に選択を適用。
*   **Technical Design**:
    ```go
    type CompileOptions struct {
        ProjectPath string
        Paths       *PathConfig
        DryRun      bool
        Target      string
        Apply       bool
        EmitMode    emitter.EmitMode
        EmitDryRun  bool
        Tags        []string // already normalized; empty means caller bug — Compile defaults to []string{manifest.BaselineTag}
        TagRefs     string   // include|strict; empty → include
    }
    ```
*   **Logic**（既存ステップ番号の直後）:
    1. `Tags` が nil/empty なら `[]string{manifest.BaselineTag}` に正規化。
    2. `TagRefs` が空なら `manifest.TagRefsInclude`。不正値は即エラー return。
    3. 既存の schema / ID / `ValidateReferences` の後、エラーが無ければ:
       `selErrs := manifest.ApplyTagSelection(entities, opts.Tags, mode)`
       `result.Errors` に append。エラーがあれば既存どおり Resolve せず return。
    4. その後 `Resolve` → Marshal（全エンティティ＋`selected`）→ Emit。

#### [MODIFY] [compiler_test.go](file://features/tt/internal/prompt/compiler/compiler_test.go)
*   **Description**: Tags 省略時は既存 fixture が selected になること、`--tags test` 相当で非マッチ policy が `selected: false` になること、resolved YAML に `selected` が含まれることを検証。必要なら fixture に `tags: [baseline]` を付与、または省略による暗黙 baseline に依存。

### 3. Digest / Deploy / Update

#### [MODIFY] [digest.go](file://features/tt/internal/prompt/compiler/digest.go)
*   **Description**: digest 入力に正規化済み Tags と TagRefs を含める。
*   **Technical Design**:
    ```go
    // ComputeSourceDigest hashes all source files plus selection context.
    // tags must be normalized (deduped); they are sorted before hashing for stability.
    // tagRefs must be include|strict.
    func ComputeSourceDigest(cfg *manifest.ProjectConfig, rootDir string, tags []string, tagRefs string) (string, error)
    ```
*   **Logic**:
    *   既存どおり全 sources glob の rel+content を hasher に書く。
    *   その後:
        ```go
        sorted := append([]string(nil), tags...)
        sort.Strings(sorted)
        hasher.Write([]byte("\n#tags\n"))
        hasher.Write([]byte(strings.Join(sorted, ",")))
        hasher.Write([]byte("\n#tag-refs\n"))
        hasher.Write([]byte(tagRefs))
        ```
    *   `DigestInfo` 構造は変更不要（digest 文字列自体にタグ差が反映される）。

#### [NEW/MODIFY] [digest_test.go](file://features/tt/internal/prompt/compiler/digest_test.go)
*   **Description**: 同一ソースで tags のみ変えた場合に digest が変わること、tag-refs のみ変えた場合も変わること、tags 順序が違っても同一 digest になること（ソートのため）。

#### [MODIFY] [deploy.go](file://features/tt/internal/prompt/compiler/deploy.go)
*   **Technical Design**:
    ```go
    type DeployOptions struct {
        ProjectPath string
        Paths       *PathConfig
        Target      string
        Force       bool
        DryRun      bool
        Mode        emitter.EmitMode
        Tags        []string
        TagRefs     string
    }
    ```
*   **Logic**:
    *   Tags/TagRefs を Compile と同様にデフォルト補完。
    *   `ComputeSourceDigest(cfg, rootDir, tags, tagRefs)` を digest チェック前後の両方で使用。
    *   `Compile(CompileOptions{..., Tags: tags, TagRefs: tagRefs})` へ伝播。

#### [MODIFY] [deploy_test.go](file://features/tt/internal/prompt/compiler/deploy_test.go)
*   **Description**:
    *   `TestDeploy_DigestIncludesTags`: baseline deploy → 成功保存 → `--tags test` 相当で再 deploy すると `Skipped==false`（ソース未変更でも）。
    *   既存 digest 一致スキップ系テストは `Tags`/`TagRefs` デフォルトで引き続き通ること。

#### [MODIFY] [update.go](file://features/tt/internal/prompt/compiler/update.go)
*   **Technical Design**:
    ```go
    type UpdateOptions struct {
        // existing fields...
        Tags    []string
        TagRefs string
    }
    ```
*   **Logic**: `Deploy(DeployOptions{..., Tags: opts.Tags, TagRefs: opts.TagRefs})` へ伝播。

#### [MODIFY] [update_test.go](file://features/tt/internal/prompt/compiler/update_test.go)
*   **Description**: Tags が Deploy に渡ること（digest 差または Compile 結果の selected）を 1 ケース以上で検証。

### 4. Emitters — Selected Guard

#### [MODIFY] 以下の 4 ファイル
*   [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)
*   [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)
*   [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go)
*   [codex.go](file://features/tt/internal/prompt/emitter/codex.go)

*   **Description**: policy / capability / procedure ループで `if !entity.Selected { continue }` を追加。
*   **Logic**:
    *   `resolved.Entities["target"]` および safety 系は常に selected=true 前提だが、ガードを付けても害はない（一貫性のため capability/policy/procedure に必須、target は任意）。
    *   `skip` は従来どおり非 emit（Selected に関わらず emit しない既存コメントを維持）。
    *   branch skills（`EmitBranchSkills`）はタグ対象外のまま常時処理。
*   **Technical Design**: 共通ヘルパを `emitter` に置いてもよい:
    ```go
    func skipUnselected(e *manifest.Entity) bool { return e == nil || !e.Selected }
    ```

#### [MODIFY] 対応する `*_test.go`（cursor/antigravity/claude_code/codex）
*   **Description**: `Selected=false` の capability が出力されないこと、`Selected=true` は出力されることを最小ケースで検証。

### 5. CLI

#### [MODIFY] [prompt.go](file://features/tt/cmd/prompt.go)
*   **Description**: `--tags` / `--tag-refs` を 3 サブコマンドに追加し、Options へ渡す。
*   **Technical Design**:
    ```go
    // package-level vars (or per-command — shared string vars OK like path flags)
    var (
        promptTags    string
        promptTagRefs string
    )

    // On compile/deploy/update FlagSets:
    cmd.Flags().StringVar(&promptTags, "tags", "",
        "Comma-separated selection tags (default: TT_TAGS or baseline)")
    cmd.Flags().StringVar(&promptTagRefs, "tag-refs", "",
        "Reference mode: include (default) or strict (default: TT_TAG_REFS or include)")

    func resolveTagsFlag(flagValue string, flagChanged bool) (tags []string, warnings []string, err error) {
        raw := flagValue
        if !flagChanged || raw == "" {
            // If flag not changed, prefer env; if changed to empty string, treat as invalid → baseline via env or default
            if !flagChanged {
                raw = os.Getenv(manifest.EnvKeyTags)
            }
        }
        if strings.TrimSpace(raw) == "" {
            return []string{manifest.BaselineTag}, nil, nil
        }
        return manifest.NormalizeRequestedTags(raw)
    }

    func resolveTagRefsFlag(flagValue string, flagChanged bool) (string, error) {
        raw := flagValue
        if !flagChanged || raw == "" {
            if !flagChanged {
                raw = os.Getenv(manifest.EnvKeyTagRefs)
            }
        }
        return manifest.NormalizeTagRefsMode(raw) // empty → include
    }
    ```
*   **Logic**:
    *   優先順位: CLI で `--tags` が Changed かつ非空 → その値。未指定 → `TT_TAGS`。どちらも空 → `baseline`。
    *   `--tag-refs` 同様: Changed → 値、否则 `TT_TAG_REFS`、空 → `include`。
    *   warnings は `cmd.PrintErrln` または `fmt.Fprintf(os.Stderr, "WARNING: ...\n")` で表示。
    *   `CompileOptions` / `DeployOptions` / `UpdateOptions` に解決結果を渡す。
    *   `resolveTargetFlag` と同様のスタイルを維持する（path flags の Changed パターンと混在可）。

#### [MODIFY] [common.go](file://features/tt/cmd/common.go)
*   **Description**: `knownEnvVars` に `{"TT_TAGS", "baseline"}`, `{"TT_TAG_REFS", "include"}` を追加。

#### [MODIFY] [prompt_test.go](file://features/tt/cmd/prompt_test.go)
*   **Description**:
    *   `resolveTagsFlag` / `resolveTagRefsFlag` の単体テスト（env 差し替えは `t.Setenv`）。
    *   ケース: 省略→baseline、`TT_TAGS=test`、CLI が env より優先、`"baseline, test"` トリム、重複 Warning、不正タグ名エラー、tag-refs デフォルト include。

### 6. JSON Schemas

対象プロパティ（仕様どおり）:

```json
"tags": {
  "description": "Selection tags for compile/deploy/update filtering. If omitted, the entity implicitly has the 'baseline' tag. Explicit tags replace the implicit baseline; include 'baseline' explicitly when needed. Empty arrays are forbidden.",
  "oneOf": [
    {
      "type": "string",
      "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
      "minLength": 1
    },
    {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "string",
        "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
        "minLength": 1
      }
    }
  ]
}
```

#### [MODIFY] プロジェクト本番スキーマ
*   [capability.schema.json](file://prompts/manifest/schemas/capability.schema.json) — `tags` 追加
*   [procedure.schema.json](file://prompts/manifest/schemas/procedure.schema.json) — `tags` 追加
*   [policy.schema.json](file://prompts/manifest/schemas/policy.schema.json) — 旧 Classification `tags` を上記へ置換
*   [skip.schema.json](file://prompts/manifest/schemas/skip.schema.json) — 同上

#### [MODIFY] カタログスキーマ（同名 4 ファイル）
*   [catalog/.../schemas/capability.schema.json](file://catalog/originals/axsh/go-standard-project/base/prompts/manifest/schemas/capability.schema.json)
*   procedure / policy / skip も同様

#### [MODIFY] テスト用スキーマ（存在する同名ファイルを本番と同一内容に同期）
*   `features/tt/internal/prompt/manifest/testdata/schemas/{capability,procedure,policy}.schema.json`（skip が無ければ追加）
*   `features/tt/internal/prompt/compiler/testdata/catalog_template/prompts/manifest/schemas/{capability,procedure,policy,skip}.schema.json`
*   `features/tt/internal/prompt/compiler/testdata/valid/prompts/manifest/schemas/policy.schema.json` 他、テストが参照する schema コピー

### 7. Manifest Frontmatter — Explicit `tags: [baseline]`

各ファイルの YAML frontmatter に `tags:\n  - baseline`（または `tags: [baseline]`）を追加。既存キー（apiVersion の直後など）と並べ、本文は変更しない。

#### [MODIFY] プロジェクト（19 files） under [prompts/manifest/code_content/](file://prompts/manifest/code_content/)
*   capabilities: `pre-push-knowledge-check.md`, `pre-sync-knowledge-compile.md`, `record-far-knowledge.md`
*   policies: `coding-rules.md`, `far-knowledge-memory.md`, `logging-rules.md`, `planning-rules.md`, `project-instructions.md`, `testing-rules.md`
*   procedures: `build-pipeline.md`, `create-implementation-plan.md`, `create-specification.md`, `execute-implementation-plan.md`, `investigate.md`, `review-point.md`, `run-all-tests.md`, `systematize-far-knowledge.md`, `test-generator.md`
*   refs: `vibe-coding-standard.md`

#### [MODIFY] カタログ（23 files） under [catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/](file://catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/)
*   上記に加え capabilities: `portable-file-references.md`, `prompt-manifest-update.md`, `workflow-blueprint-design-philosophy.md`
*   procedures: `create-pull-request.md`
*   refs: `vibe-coding-standard.md`

#### [MODIFY] compiler testdata 内の code_content md（valid / catalog_template 等）
*   テスト fixture の frontmatter にも `tags: [baseline]` を付与し、明示タグ経路を通す（省略経路は単体テストで別途カバー）。

### 8. Integration Tests

#### [NEW] [tests/tt/tt_prompt_tags_test.go](file://tests/tt/tt_prompt_tags_test.go)
*   **Description**: CLI 経由のタグ選択・参照モード・digest 差の統合テスト。
*   **Technical Design** (`//go:build integration`):
    *   既存 `copyCompilerValidFixture` / `runTTInDir` を再利用。
    *   必要なら temp fixture に procedure+capability（異なる tags）を書き足すヘルパ `writeTaggedFixture(t, root)` を同ファイルに定義。
*   **テストケース**:
    1. `TestPromptTags_DefaultBaselineSelectsUntagged` — `--tags` 無し compile dry-run 成功、resolved に selected true
    2. `TestPromptTags_TestOnlyExcludesBaseline` — baseline 明示と test のみの 2 capability を配置し `--tags test` で test 側のみ selected
    3. `TestPromptTags_IncludePullsReferencedCapability` — procedure(baseline)→capability(test)、`--tags baseline` で両方 selected
    4. `TestPromptTags_StrictFailsOnCrossTagRef` — 同上で `--tag-refs strict` が非ゼロ exit
    5. `TestPromptTags_TT_TAGSEnv` — `t.Setenv("TT_TAGS","test")` で CLI 省略時に test 選択
    6. `TestPromptDeploy_DigestDiffersByTags` — deploy baseline 後、同一ソースで `--tags test` がスキップされない

## Step-by-Step Implementation Guide

1. **TDD: tags core**
   *   `tags_test.go` を先に書き、`scripts/process/build.sh --skip-frontend --skip-etc` で失敗を確認。
   *   `tags.go` を実装して単体テストを通す。

2. **Entity.Selected**
   *   `types.go` / `types_test.go` を更新。

3. **Schemas**
   *   本番・カタログ・testdata の capability/procedure/policy/skip に `tags` oneOf を適用。

4. **Compile hook**
   *   `compiler.go` に Tags/TagRefs と `ApplyTagSelection` 呼び出しを追加。
   *   `compiler_test.go` を更新（失敗→実装）。

5. **Digest / Deploy / Update**
   *   `ComputeSourceDigest` シグネチャ変更と全呼び出し更新。
   *   deploy/update Options 伝播とテスト。

6. **Emitters**
   *   4 emitter に `Selected` ガードとテスト。

7. **CLI**
   *   `prompt.go` / `common.go` / `prompt_test.go`。

8. **Frontmatter baseline 明示**
   *   プロジェクト 19 + カタログ 23 + testdata md。

9. **Integration tests**
   *   `tests/tt/tt_prompt_tags_test.go` を追加。

10. **Verify**
    *   Verification Plan を実行し、総合判定を記録する。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Integration Tests（タグ機能）**:
   ```bash
   ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "tt" --specify "PromptTags|PromptDeploy_Digest|PromptPaths"
   ```
   *   **Log Verification**:
       *   `TestPromptTags_*` が全て PASS
       *   strict ケースで非ゼロ exit / エラーメッセージに参照タグ不一致が含まれる
       *   digest ケースで 2 回目 deploy が skip されない
       *   既存 `PromptPaths` リグレッションが PASS

3. **E2E / Integration 理由**:
   *   GUI E2E は不要（CLI / ファイルシステム機能のため）。
   *   `tests/tt/` 統合テストが「CLI サブコマンドとして利用者に提供」「ファイルシステム読み書き」の必須基準を満たす。

### テスト項目設計（§11）

#### ボトムアップ順序
1. `EffectiveTags` / `NormalizeRequestedTags` / `NormalizeTagRefsMode`（末端純ロジック）
2. `ApplyTagSelection`（参照閉包・strict）
3. `Entity` Marshal `selected`
4. `Compile` パイプライン統合（単体・tempdir）
5. `ComputeSourceDigest` タグ差
6. Emitters の Selected ガード
7. CLI resolve + env
8. `tests/tt` 統合（全体）

#### 観点チェックリスト
| # | 観点 | カバー |
|---|---|---|
| 1 | 正常系 | baseline 省略、明示、OR、include 閉包 |
| 2 | 異常系・境界 | 空配列、不正 kebab、strict エラー、不明 tag-refs |
| 3 | 外部連携 | schema ファイル、fixture FS、digest ファイル |
| 4 | データ一貫性 | resolved YAML roundtrip の selected |
| 5 | 状態遷移 | deploy 後タグ変更で再デプロイ |
| 6 | 設定反映 | TT_TAGS / TT_TAG_REFS / CLI 優先 |
| 7 | 副作用 | 非選択 entity が emit されない、memory は残る |

#### セルフレビュー結果（§11.4）
1. **網羅性**: 仕様の必須要件 1〜11 およびレビュー決定を単体＋統合でカバー。GUI 非対象は明示。
2. **証拠の十分性**: selected の YAML 内容、emit 有無、digest 不等号、exit code を直接アサート。
3. **迂回排除**: CLI→Options→ApplyTagSelection→Emit の経路を統合テストで踏む。
4. **依存関係**: 純ロジック → compiler → CLI/integration の順で計画。

### 総合判定プロセス（§12）

全テスト成功後、実装実行ワークフローにて以下を実施し記録する:
*   スキップ有無、WARN（重複タグ Warning は意図的）、primary path 確認
*   新規 `PromptTags` テスト欠落が無いか
*   判定フォーマット（✅/⚠️/❌）で総合判定を出力

## Documentation

`prompts/specifications/` ディレクトリは現時点で存在しない。本変更に対応する利用者向け仕様の正本は Source Specification（ideas）とする。

追加で更新が必要な場合:
*   将来 `prompts/specifications` に prompt compile 文書が追加されたら、`--tags` / `--tag-refs` / `baseline` を追記する（本計画のスコープ外）。
*   カタログ originals のスキーマ・code_content 更新自体が一般利用者向けドキュメント同期に相当する。
