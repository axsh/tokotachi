# 024-Capability-Skill-Bundle

> **Source Specification**: [023-Capability-Skill-Bundle.md](file://prompts/phases/000-foundation/branches/main/ideas/023-Capability-Skill-Bundle.md)

## Goal Description

capability の `bundle`（`src`/`dest`）に列挙されたファイルを、Emit 時に各ターゲットのスキルフォルダへ物理同梱し、`SKILL.md` 本文中のワークスペース相対パスをスキル相対パスへ書き換える。`references` / `scripts` は案内用のまま同梱しない。スキーマ本体は Review Point で先行適用済みのため、本計画は emitter 共通ヘルパー・Check/orphan 対応・既存 capability 移行・単体/統合テストを実装する。

## User Review Required

None. ユーザーより実装計画作成 → 実装実行 → ツール/コンテンツリリースまでの明示指示あり。R1 スキーマは Review Point で適用済み。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: capability スキーマに `bundle` | 先行適用済み (`prompts/manifest/schemas/capability.schema.json` ほか)。本計画では再変更なし（検証のみ） |
| R2: Emit 時に bundle 同梱 | Proposed Changes > `bundle.go`, cursor/claude/codex/antigravity emitters |
| R3: SKILL.md パス書き換え | Proposed Changes > `RewriteBundlePaths` in `bundle.go` |
| R4: references/scripts 非同梱 | スキーマ description 済み + emitter が `bundle` のみ読む単体テスト |
| R5: Drift/Check companion | Proposed Changes > 各 emitter `Check` の拡張子フィルタ緩和 |
| R6: record-far-knowledge 移行 | Proposed Changes > capability MD（prompts + catalog + testdata） |
| O1: Branch skills 丸ごとコピー | Proposed Changes > `branch_skills.go`（本計画で実装） |
| O2/O3 | 先送り（dest 必須維持 / ソースフォルダ化は非スコープ） |
| S1–S7 | Verification Plan + `bundle_test.go` / `tt_prompt_bundle_test.go` |

## Proposed Changes

### features/tt/internal/prompt/emitter（共通バンドル）

#### [NEW] [bundle_test.go](file://features/tt/internal/prompt/emitter/bundle_test.go)

*   **Description**: TDD 用。ヘルパーの Red → Green。
*   **Technical Design**:
    ```go
    func TestParseBundleEntries_OK(t *testing.T)
    func TestParseBundleEntries_MissingFields(t *testing.T)
    func TestParseBundleEntries_Nil(t *testing.T)
    func TestValidateSkillDest_RejectsTraversal(t *testing.T)
    func TestValidateSkillDest_RejectsAbsolute(t *testing.T)
    func TestRewriteBundlePaths_BacktickAndMarkdownLink(t *testing.T)
    func TestRewriteBundlePaths_LongestSrcFirst(t *testing.T)
    func TestRewriteBundlePaths_LeavesUnbundledPaths(t *testing.T)
    func TestEmitBundledFiles_CopiesAndRegisters(t *testing.T)
    func TestEmitBundledFiles_MissingSrcErrors(t *testing.T)
    func TestEmitBundledFiles_DestEscapeErrors(t *testing.T)
    ```
*   **Logic**:
    *   Parse: `Raw["bundle"]` が `[]any` of `map[string]any` のとき `src`/`dest` を string 抽出。欠落は error。
    *   Rewrite: backtick 内および markdown link `](path)` の path を置換。src は長い順に置換。
    *   Emit: workspace の src を読み、skillDir/dest へ書き、emitted map に登録。

#### [NEW] [bundle.go](file://features/tt/internal/prompt/emitter/bundle.go)

*   **Description**: bundle 解析・パス検証・本文書き換え・ファイル同梱。
*   **Technical Design**:
    ```go
    package emitter

    type BundleEntry struct {
        Src  string // workspace-relative, slash-normalized for matching
        Dest string // skill-root-relative
    }

    func ParseBundleEntries(raw any) ([]BundleEntry, error)
    func ValidateSkillDest(dest string) error
    func RewriteBundlePaths(body string, entries []BundleEntry) string
    func EmitBundledFiles(skillDir, workspaceRoot string, entries []BundleEntry, opts EmitOptions) (map[string]bool, error)
    func writeBytesWithMode(path string, data []byte, mode EmitMode) error
    ```
*   **Logic**:
    *   `ParseBundleEntries`: nil/absent → empty slice, nil error。各要素で `src`/`dest` 必須。空文字禁止。`filepath.ToSlash` で Src を正規化。
    *   `ValidateSkillDest`: `filepath.IsAbs(dest)` なら error。`Clean` 後に `..` で始まる、または `..` セグメントを含むなら error。空も error。
    *   `RewriteBundlePaths`: entries を `len(Src)` 降順ソート。各 src について:
        *   `` `src` `` → `` `dest` ``（dest は ToSlash）
        *   `](src)` → `](dest)` および `](./src)` → `](./dest)`（存在する場合）
      unbundled な `./scripts/...` 等は触らない。
    *   `EmitBundledFiles`: 各 entry で ValidateSkillDest。`srcPath = Join(workspaceRoot, FromSlash(src))`。ReadFile 失敗は `fmt.Errorf("bundle src not found: %s: %w", src, err)`。`out = Join(skillDir, FromSlash(dest))`。`Clean(out)` が `Clean(skillDir)` の子であることを再確認。`writeBytesWithMode`。emitted に Clean(out) を登録。
    *   `writeBytesWithMode`: EmitModeSkip なら存在時スキップ。それ以外は MkdirAll + WriteFile。

### features/tt/internal/prompt/emitter（各ターゲット）

#### [MODIFY] [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go) / [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go) / [codex.go](file://features/tt/internal/prompt/emitter/codex.go) / [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)

*   **Description**: capability emit で bundle 処理を挿入。Check の untracked 走査を非 md にも拡張。
*   **Logic**（capability ループ内、body 取得後）:
    1. `entries, err := ParseBundleEntries(skill.Raw["bundle"])` — 失敗なら return err
    2. `body = RewriteBundlePaths(body, entries)`（stripFrontmatter / ResolveTemplateVars の後、frontmatter 組み立て前）
    3. SKILL.md を既存どおり write
    4. `bundled, err := EmitBundledFiles(filepath.Join(skillsDir, skill.ID), c.RootDir, entries, opts)` — 失敗なら return err
    5. bundled を emittedFiles にマージ
*   **Check untracked**: `.md`/`.mdc` 限定をやめ、`README.md` / `.gitkeep` 以外はすべて untracked 候補とする（companion `.json` の余剰検出）。期待ファイル比較は既存の Walk（全ファイル）を維持。

#### [MODIFY] [cursor_test.go](file://features/tt/internal/prompt/emitter/cursor_test.go)（代表）および必要なら他 emitter_test

*   **Description**: bundle 付き capability の emit で companion と書き換えを検証するケースを追加。
*   **Logic**: tempDir に `fixtures/payload.json` を置き、capability Raw に
    `bundle: []any{map[string]any{"src":"fixtures/payload.json","dest":"references/payload.json"}}`、
    body に `` See `fixtures/payload.json` ``。Emit 後:
    *   skills/test-skill/references/payload.json 存在
    *   SKILL.md に `references/payload.json` を含み `fixtures/payload.json` を含まない

### features/tt/internal/prompt/emitter（branch skills O1）

#### [MODIFY] [branch_skills.go](file://features/tt/internal/prompt/emitter/branch_skills.go) / [branch_skills_test.go](file://features/tt/internal/prompt/emitter/branch_skills_test.go)

*   **Description**: スキルディレクトリ内の全ファイルをコピー（`SKILL.md` 以外含む）。
*   **Logic**:
    *   `EmitBranchSkills`: `srcDir := filepath.Dir(skill.Path)` を Walk。ファイルごとに rel を取り `Join(skillsDir, skill.ID, rel)` へ `writeBytesWithMode`。全て emitted 登録。
    *   テスト: companion `references/note.md` を含むスキルを Scan/Emit し、出力側に存在することを assert。

### capability コンテンツ移行（R6）

#### [MODIFY] [record-far-knowledge.md](file://prompts/manifest/code_content/capabilities/record-far-knowledge.md)

*   **Description**: schema を `bundle` へ移し、body にスキル相対参照を追加。
*   **Logic**:
    ```yaml
    # remove from references:
    # - "prompts/memory/schemas/agent-record-payload.schema.json"
    bundle:
      - src: prompts/memory/schemas/agent-record-payload.schema.json
        dest: references/agent-record-payload.schema.json
    scripts:
      - "scripts/code/agent/record.sh"   # unchanged, not bundled
    ```
    *   body の Constraints または Recording Tool 付近に次を追加:
        ``Payload schema: `references/agent-record-payload.schema.json` ``
    *   同内容を catalog originals と compiler catalog_template の同名ファイルへ反映。

#### [MODIFY] catalog / testdata 同名 capability

*   `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/capabilities/record-far-knowledge.md`
*   `features/tt/internal/prompt/compiler/testdata/catalog_template/prompts/manifest/code_content/capabilities/record-far-knowledge.md`
*   pre-push / pre-sync の peer `references` と `scripts` は変更しない。

### tests/tt（統合 / E2E 相当）

#### [NEW] [tt_prompt_bundle_test.go](file://tests/tt/tt_prompt_bundle_test.go)

*   **Description**: `tt prompt deploy` 経由で bundle 同梱を検証（CLI + ファイルシステム）。
*   **テストケース**:
    *   `TestPromptDeploy_CapabilityBundle`: copyCompilerValidFixture ベース、capability を bundle 付きで追加、schema/fixture ファイル配置、`tt prompt deploy --target cursor`、出力スキル配下に companion と書き換え済み SKILL.md を確認。
    *   `TestPromptDeploy_ReferencesOnlyNoBundle`: references/scripts のみでは companion が増えない。
*   **検証ポイント**: S2, S5, S1 相当を CLI 経路でカバー。

## Step-by-Step Implementation Guide

1. **TDD: bundle helpers**:
    *   `bundle_test.go` を先に追加し、失敗することを確認（または実装と同時に Green）。
    *   `bundle.go` を実装し単体テストを通す。
2. **Wire emitters**:
    *   cursor / claude_code / codex / antigravity の capability ループに Parse → Rewrite → EmitBundledFiles を挿入。
    *   各 Check の untracked 拡張子フィルタを緩和。
    *   cursor_test に bundle ケース追加。
3. **Branch skills O1**:
    *   `EmitBranchSkills` をディレクトリコピーに変更しテスト追加。
4. **Migrate record-far-knowledge**:
    *   prompts / catalog / testdata の3箇所を更新。
5. **Integration test**:
    *   `tests/tt/tt_prompt_bundle_test.go` を追加。
6. **Verification Plan を実行**（下記）。
7. **Far-knowledge record → push → tool release → content release**。

## Verification Plan

### テスト項目設計（§11）とセルフレビュー

**ボトムアップ順序**:
1. C: `ParseBundleEntries` / `ValidateSkillDest` / `RewriteBundlePaths`（純ロジック）
2. B: `EmitBundledFiles` + cursor Emit（FS 副作用）
3. A: `tt prompt deploy` 統合テスト（CLI エンドツーエンド）

**観点チェックリスト**:
| # | 観点 | 対応テスト |
|---|------|-----------|
| 1 | 正常系 | Rewrite OK, Emit copies, Deploy bundle |
| 2 | 異常系 | missing src, dest traversal/absolute |
| 3 | 外部連携（FS） | EmitBundledFiles, Deploy |
| 4 | データ一貫性 | コピー内容一致、パス置換一致 |
| 5 | 状態遷移 | deploy 前後のスキルフォルダ |
| 6 | 設定反映 | Raw bundle のみが同梱対象 |
| 7 | 副作用 | references-only で companion 非生成 |

**セルフレビュー結果**: 網羅性は S1–S5 を単体+統合でカバー。証拠はファイル内容 assert。迂回は Raw に bundle を置かないケースで references 非同梱を明示。依存は helpers → emitter → CLI の順。十分と判断。

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Prompt|Bundle|Emit|Deploy|Compile|Tag"
    ```
    *   **Log Verification**: `TestPromptDeploy_CapabilityBundle` 成功、FAIL/SKIP なし。

3. **E2E Tests**:
    #### [NEW] [tt_prompt_bundle_test.go](file://tests/tt/tt_prompt_bundle_test.go)
    *   CLI 経由の deploy が本機能の利用者経路であるため、`tests/tt` 統合テストを E2E 相当とする（別途 agentservice E2E は不要）。
    *   理由: 変更は `tt prompt` サブシステムに閉じ、GUI なし。

### 総合判定プロセス（§12）

全テスト成功後、実装計画末尾または実行報告に総合判定結果を記載する（スキップ有無、WARN、新機能カバレッジ）。

## Documentation

#### [MODIFY] [023-Capability-Skill-Bundle.md](file://prompts/phases/000-foundation/branches/main/ideas/023-Capability-Skill-Bundle.md)
*   **更新内容**: Review Point でスキーマ先行適用済みである旨は反映済み。追加の仕様変更なし。

`prompts/specifications` 配下に該当ドキュメントなし（影響更新なし）。
