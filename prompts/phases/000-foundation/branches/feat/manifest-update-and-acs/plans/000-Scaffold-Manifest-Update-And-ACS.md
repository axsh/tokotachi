# 000-Scaffold-Manifest-Update-And-ACS

> **Source Specification**: [prompts/phases/000-foundation/branches/main/ideas/031-Scaffold-Manifest-Update-And-ACS.md](../../main/ideas/031-Scaffold-Manifest-Update-And-ACS.md)

## Goal Description

`tt scaffold … --update` によるマニフェスト更新ワークスペース（backup / latest / instruction / `doc-relations-NNN`）、frontmatter `uid`、Character n-gram Jaccard、および ACS B 案（`tt prompt tags`、非 baseline サンプル、ドキュメント、更新・マージ capability と `prompt-manifest-update` からの呼び出し）を実装する。

## User Review Required

None.（仕様レビュー済み。ユーザー指示により計画→実装を連続実行する）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| A1 `--update [workdir]` | Proposed Changes > scaffold CLI + RunOptions |
| A2 workdir レイアウト | `pkg/scaffold/update_workspace.go` |
| A3 stdout = instruction | `update_workspace.go` + CLI |
| A4 クリーン更新 | `IsCleanUpdate` + apply path |
| A5 instruction 英語方針 | `DefaultMergeInstruction()` 定数 |
| A6 `uid` | schemas + catalog/REPO frontmatter + validator |
| A7 `doc-relations-NNN` + n-gram Jaccard | `pkg/scaffold/similarity.go` + `doc_relations.go` |
| A8 履歴スキップしない／記録可 | apply/update 分岐 |
| A9 capability + prompt-manifest-update 呼び出し | BASE/REPO capabilities |
| B1–B4 ACS | tags CLI + samples + help/docs |
| n 選定フィクスチャ | `similarity_test.go` で n=3,4,5 比較し定数固定 |

## Proposed Changes

パス略称:

- `BASE` = `catalog/originals/axsh/go-standard-project/base/prompts/manifest`
- `REPO` = `prompts/manifest`

### Unit: similarity (TDD first)

#### [NEW] [pkg/scaffold/similarity.go](pkg/scaffold/similarity.go)

*   **Description**: Character n-gram Jaccard。
*   **Technical Design**:

```go
const DefaultNGramN = 4 // fixed after fixture comparison in tests

func NormalizeForSimilarity(s string) string // CRLF→LF, collapse runs of whitespace optionally keep simple: only CRLF→LF + trim
func CharNGrams(s string, n int) map[string]struct{}
func Jaccard(a, b map[string]struct{}) float64
func SimilarityScore(left, right string, n int) float64
```

*   **Logic**:
    *   rune 単位で長さ n の連続部分文字列を集合化（重複は集合なので1回）。
    *   Jaccard = \|A∩B\| / \|A∪B\|。空集合同士は 1.0、片方空は 0.0。
    *   `DefaultNGramN` はフィクスチャ比較後に固定（暫定実装中はテストが最良 n を選び、定数と一致させる）。

#### [NEW] [pkg/scaffold/similarity_test.go](pkg/scaffold/similarity_test.go)

*   **テストケース**:
    *   `TestJaccard_Basic`
    *   `TestSimilarity_FixtureDispersion` — 対: 軽微編集（高）、リネーム相当（高）、無関係（低）、定型 frontmatter のみ類似（誤って高すぎない）。n∈{3,4,5} でスコア表を計算し、**正対応の最小スコア > 誤対応の最大スコア** となる n を選び、`DefaultNGramN` と一致することを要求。

### Unit: doc-relations + instruction

#### [NEW] [pkg/scaffold/doc_relations.go](pkg/scaffold/doc_relations.go)

```go
type DocRelation struct {
    BackupRelPath string
    UID           string
    UIDMatchPath  string   // latest path if uid match
    SimilarTop3   []SimilarHit
}
type SimilarHit struct {
    LatestRelPath string
    Score         float64
}

func ParseFrontmatterUID(content []byte) (uid string, ok bool)
func BuildDocRelations(backupRoot, latestRoot string) ([]DocRelation, error)
func WriteDocRelationFiles(workdir string, rels []DocRelation, maxPerFile int) ([]string, error) // writes doc-relations-001.md ...
func ShouldSkipRelationPath(rel string) bool // __far-knowledge- in base name or id
```

*   **Logic**:
    *   backup 配下の `.md` を走査。`ShouldSkipRelationPath` で除外。
    *   uid マップを latest から構築。一致すれば `UIDMatchPath`。
    *   不一致時は latest 全件と SimilarityScore、上位 3。
    *   出力は常に `doc-relations-001.md` 始まり（`maxPerFile` 超過で連番継続）。連番なし `doc-relations.md` は書かない。

#### [NEW] [pkg/scaffold/merge_instruction.go](pkg/scaffold/merge_instruction.go)

```go
func DefaultMergeInstruction(workdir string, relationFiles []string) string
```

*   **Logic**: 仕様 A5 の英語方針 1–6 を本文に含め、workdir・backup・latest・各 `doc-relations-NNN.md` へのパスを列挙。

#### [NEW] [pkg/scaffold/update_workspace.go](pkg/scaffold/update_workspace.go)

```go
type UpdateResult struct {
    WorkDir        string
    CleanApplied   bool
    Instruction    string
    RelationFiles  []string
}

func DefaultUpdateWorkDir(repoRoot, category, name string) string
func PrepareUpdateWorkspace(opts UpdateWorkspaceOptions) (*UpdateResult, error)

type UpdateWorkspaceOptions struct {
    RepoRoot       string
    WorkDir        string
    LocalManifest  string // repoRoot/prompts/manifest
    LatestFiles    map[string][]byte // rel path under prompts/manifest → content
    Stdout         io.Writer
}
```

*   **Logic**:
    1. workdir 作成。`backup/` に LocalManifest を再帰コピー。
    2. `latest/` に LatestFiles を書き出し（`prompts/manifest` 相当の相対パス）。
    3. `IsCleanUpdate(backup, latest)`: 対象ファイル集合が一致し全バイト一致なら true。不一致や片側のみファイルがあれば false（安全側）。
    4. clean なら LocalManifest を latest 内容で置換（余剰削除含む）、`CleanApplied=true`、履歴更新は呼び出し側。
    5. 非 clean なら relations + instruction を書き、Stdout とファイルに同じ instruction、保存先を補足。

#### [NEW] 対応 `*_test.go`

フィクスチャ一時ディレクトリで clean / dirty / doc-relations 連番 / far-knowledge 除外を検証。

### Scaffold CLI + RunOptions

#### [MODIFY] [pkg/scaffold/scaffold.go](pkg/scaffold/scaffold.go)

*   `RunOptions` に `Update bool`, `UpdateWorkDir string` を追加。
*   `Update && Force` はエラー（同時指定禁止）。
*   `Apply` / apply 経路: `Update` 時は履歴スキップしない。テンプレート取得後、`prompts/manifest` 配下（パスに `prompts/manifest/` を含む、または placement 後の相対がそれに相当）を `LatestFiles` に集め `PrepareUpdateWorkspace`。clean でなければ通常 ApplyFiles はしない。依存チェーン時は project 本体（manifest を含むエントリ）に対して update を実行；実装は「最終エントリまたは manifest を含むテンプレート」を対象とし、計画では **project スキャフォールド単体＋`--skip-deps` でも動く**ことを最低保証、依存チェーンは可能なら同様に manifest 抽出。

#### [MODIFY] [features/tt/cmd/scaffold.go](features/tt/cmd/scaffold.go)

```go
scaffoldCmd.Flags().BoolVar(&scaffoldFlagUpdate, "update", false, "...")
scaffoldCmd.Flags().StringVar(&scaffoldFlagUpdateDir, "update-dir", "", "...")
```

*   位置引数の第3解釈は避け、workdir は `--update` のオプション引数または `--update-dir`。仕様例 `tt scaffold project axsh-go-standard --update tmp/update/` に合わせ、`--update` を `StringVar` にして空ならデフォルト workdir、値がパス、と解釈する:

```go
scaffoldCmd.Flags().StringVar(&scaffoldFlagUpdate, "update", "", "Prepare manifest update workspace (optional workdir; empty value with flag set → default tmp/scaffold-update/...)")
// Use NoOptDefVal so --update alone works:
scaffoldCmd.Flags().Lookup("update").NoOptDefVal = "__default__"
```

*   `Changed("update")` で Update モード。値が `__default__` なら DefaultUpdateWorkDir。

### uid schema + frontmatter

#### [MODIFY] schemas（REPO + BASE）

`capability.schema.json` / `policy.schema.json` / `procedure.schema.json` / `skip.schema.json` に:

```json
"uid": {
  "type": "string",
  "format": "uuid",
  "description": "Stable document identity across renames. RFC 4122 UUID. Do not change once published in catalog."
}
```

（任意プロパティ。catalog 正本では全 code_content に付与）

#### [MODIFY] BASE + REPO の全 taggable `.md`

各 frontmatter に一意の `uid: <uuid>` を追加（一度発行したら固定）。実装時は決定的でないランダム UUID でよいが、BASE=REPO 共有ファイルは同一 uid。

### ACS: tt prompt tags

#### [NEW] [features/tt/cmd/prompt_tags_list.go](features/tt/cmd/prompt_tags_list.go)（または prompt.go 内）

```go
var promptTagsListCmd = &cobra.Command{
  Use:   "tags",
  Short: "List selection tags and member entities (ACS discovery)",
  RunE:  runPromptTagsList,
}
```

*   compile と同様に paths 解決→マニフェスト読込→各 taggable の EffectiveTags を集計。
*   出力: タグ一覧、続けてタグごとに kind/id/path。暗黙 baseline の説明を1行。

#### [MODIFY] prompt help Short/Long

ACS 呼称と `--tags` / `tt prompt tags` を追記。

### ACS samples + docs

#### [NEW] `BASE|REPO/capabilities/acs-external-agent-sample.md`

*   `tags: [external-agent]`（baseline を含めない）
*   短い説明のみ。B4 の見本。

#### [MODIFY] `prompt-manifest-update` および ACS 説明

ヘルプ／capability に ACS 短文。B2。

### A9 capabilities

#### [NEW] `BASE|REPO/capabilities/scaffold-manifest-update.md`

*   id: `scaffold-manifest-update`
*   本文（日本語）: `--update`、workdir、clean vs merge、instruction / doc-relations-NNN、uid、マージ方針。
*   `tags: [baseline]` + `uid`

#### [MODIFY] `BASE|REPO/capabilities/prompt-manifest-update.md`

*   `uses_capabilities: [scaffold-manifest-update]`
*   上流取り込みの詳細は削除し `\{{capability:scaffold-manifest-update}}` を呼ぶ。

### Integration tests

#### [NEW] [tests/tt/tt_scaffold_manifest_update_test.go](tests/tt/tt_scaffold_manifest_update_test.go)

*   ネットワーク不要のユニット寄りの統合: 一時 repo に manifest を置き、`PrepareUpdateWorkspace` 相当を直接、または `tt` の update をモック可能な経路で検証。
*   CLI フラグ: `tt scaffold --help` に update が出ること。
*   可能なら既存 fixture zip / ローカルファイルで latest を差し込み。

#### [NEW] [tests/tt/tt_prompt_tags_list_test.go](tests/tt/tt_prompt_tags_list_test.go)

*   一時ツリーで `tt prompt tags` が baseline / external-agent 等を列挙。

#### [NEW/MODIFY] [tests/tt/tt_scaffold_manifest_update_suite_test.go](tests/tt/tt_scaffold_manifest_update_suite_test.go)

*   BASE/REPO に `scaffold-manifest-update` 存在、`prompt-manifest-update` が参照、uid 付き、サンプル `external-agent`、連番関係の文書キーワード。

## Step-by-Step Implementation Guide

1. **Similarity TDD**: `similarity_test.go` を先に書きレッド → `similarity.go` 実装 → フィクスチャで `DefaultNGramN` 固定。
2. **doc_relations + instruction TDD**: テスト→実装。
3. **update_workspace TDD**: clean/dirty → 実装。
4. **CLI `--update`**: RunOptions + scaffold.go 配線。
5. **uid schemas + 一括付与**（BASE/REPO）。
6. **`tt prompt tags`** + テスト。
7. **ACS sample + help**。
8. **scaffold-manifest-update capability + prompt-manifest-update 改訂**（BASE=REPO）。
9. **統合テスト**追加。
10. **Verification Plan** 実行 + 総合判定。
11. backlog `ae7030f2-...` を n 固定後に resolve。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Integration Tests**:
   ```bash
   ./scripts/process/integration_test.sh --categories "tt" --specify "Scaffold|PromptTags|ManifestUpdate|ScaffoldManifest|DocRelations|Uid|PromptTagsList"
   ```

3. **E2E Tests**:
   GUI なし。CLI 統合（`tests/tt`）で代替。E2E カテゴリ追加は不要（理由: 画面変更なし、CLI は `tests/tt` が実バイナリを実行）。

### テスト項目設計（ボトムアップ）

| Step | 対象 | 観点 |
| :--- | :--- | :--- |
| 1 | Jaccard / n-gram | 正常・境界（空）、分散 |
| 2 | doc-relations | uid 一致、top3、連番、far-knowledge 除外 |
| 3 | update workspace | clean 上書き、dirty 非破壊、instruction 出力 |
| 4 | CLI flags | update モード、force 併用エラー |
| 5 | prompt tags | 一覧・所属 |
| 6 | catalog suite | A9・uid・sample・参照 |

#### セルフレビュー（testing-rules §8.4）

1. **網羅性**: 仕様 A1–A9 / B1–B4 を上表と Traceability でカバー。n はフィクスチャで固定。
2. **証拠**: ファイル存在・内容部分一致・スコア不等式・CLI 出力文字列。
3. **迂回排除**: clean パスはバイト一致を要求。tags は EffectiveTags 経由。
4. **依存**: 類似度→relations→workspace→CLI の順。

### 総合判定

Verification 完了後、testing-rules §9 の表で総合判定結果を計画末尾または実行ログに残す。

## Documentation

#### [MODIFY] 仕様 031 の実現方針

*   採用した `DefaultNGramN` の具体値を追記（実装後）。

#### [MODIFY] `tt` help / ACS 短文

*   prompt / scaffold help。
