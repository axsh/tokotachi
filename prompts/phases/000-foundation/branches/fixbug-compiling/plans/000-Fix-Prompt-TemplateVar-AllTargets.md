# 000-Fix-Prompt-TemplateVar-AllTargets

> **Source Specification**: `prompts/phases/000-foundation/branches/fixbug-compiling/ideas/000-Fix-Prompt-TemplateVar-AllTargets.md`

## Goal Description

Claude / Codex / Cursor の prompt emit でも `{{policy:…}}` / `{{procedure:…}}` 等をターゲット固有パスへ解決し、policy ソース内の `.agent/workflows/…` ハードコードをテンプレート変数（または中立表現）へ置き換える。これにより、再デプロイ後の全ターゲット成果物で参照パスが実ファイルと一致する。

## User Review Required

1. **ディレクトリ直書きの言い換え文言**（例: 「`.agent/workflows/` 配下に定義されたワークフローは…」）を、テンプレート変数ではなく中立表現にする方針でよいか（仕様 R3 の許容範囲内）。代替案として `{{target:…}}` 系の新変数追加は本計画では行わない。
2. 任意要件 **O1（`.agents/` 残骸掃除）** は本計画のスコープ外（仕様どおり先送り）でよいか。

それ以外は None（技術方針は仕様の表に従う）。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 全ターゲットで `ResolveTemplateVars` | Proposed Changes > claude_code.go / codex.go / cursor.go |
| R2: 解決パスと実配置の一致（拡張子・instructions リネーム） | Proposed Changes > template.go（`TemplateContext` 拡張） |
| R3: policy ハードコードを可搬化 | Proposed Changes > policies（workspace / catalog / compiler testdata） |
| R4: 再デプロイ後に未展開・誤パスが残らない | Verification Plan > Integration `TestPromptTemplateVars_*` + Update 後アサーション |
| R5: 全ターゲット展開の単体テスト固定 | Proposed Changes > template_test.go / *_test.go（各 emitter） |
| O1: `.agents/` 残骸 | 先送り（本計画では実装しない） |
| O2: 未知プレースホルダは現状維持 | template.go の既存挙動を変更しない |

## Proposed Changes

> TDD: 各コンポーネントで `_test.go` を先に記述・失敗確認してから実装する。

### emitter（テンプレート解決）

#### [MODIFY] [features/tt/internal/prompt/emitter/template_test.go](file://features/tt/internal/prompt/emitter/template_test.go)

*   **Description**: ターゲット別 policy 命名・拡張子のテーブル駆動テストを追加し、既存 Antigravity ケースを回帰固定する。
*   **Technical Design**:
    *   新規: `TestResolveTemplateVars_PerTargetPolicyNaming`
    *   既存 `TestResolveTemplateVars` の Antigravity 期待値（`instructions.md`）は、`RenameProjectInstructions: true` / `PolicyExt: ".md"` を明示したコンテキストで維持する。
*   **Logic**（追加ケースの期待値をそのまま固定）:

| TargetName | PolicyExt | RenameProjectInstructions | 入力 | 期待出力 |
| :--- | :--- | :--- | :--- | :--- |
| `claude-code` | `.md` | false | `{{policy:testing-rules}}` | `.claude/rules/testing-rules.md` |
| `codex` | `.md` | false | `{{policy:testing-rules}}` | `.codex/rules/testing-rules.md` |
| `cursor` | `.mdc` | false | `{{policy:testing-rules}}` | `.cursor/rules/testing-rules.mdc` |
| `cursor` | `.mdc` | false | `{{policy:project-instructions}}` | `.cursor/rules/project-instructions.mdc` |
| `claude-code` | `.md` | false | `{{policy:project-instructions}}` | `.claude/rules/project-instructions.md` |
| `antigravity` | `.md` | true | `{{policy:project-instructions}}` | `.agent/rules/instructions.md` |
| `claude-code` | `.md` | false | `{{procedure:build-pipeline}}` | `.claude/skills/build-pipeline/SKILL.md`（Workflows 空） |
| `antigravity` | `.md` | true | `{{procedure:build-pipeline}}` | `.agent/workflows/build-pipeline.md`（Workflows 設定あり） |
| any | — | — | `{{unknown:foo}}` | `{{unknown:foo}}`（未変更） |

#### [MODIFY] [features/tt/internal/prompt/emitter/template.go](file://features/tt/internal/prompt/emitter/template.go)

*   **Description**: `TemplateContext` にターゲット別 policy 命名フィールドを追加し、`resolvePolicyPath` が実ファイル名と一致するパスを返すようにする。
*   **Technical Design**:

```go
// TemplateContext holds the information needed to resolve template variables.
type TemplateContext struct {
	Paths                     TargetPaths
	MemBase                   string // e.g., "prompts/memory"
	TargetName                string // e.g., "antigravity"
	PolicyExt                 string // ".md" or ".mdc"; empty defaults to ".md"
	RenameProjectInstructions bool   // true => project-instructions -> instructions{ext}
}

// NewTemplateContext builds a TemplateContext with target-specific naming defaults.
func NewTemplateContext(targetName string, paths TargetPaths) *TemplateContext {
	ctx := &TemplateContext{
		Paths:      paths,
		MemBase:    "prompts/memory",
		TargetName: targetName,
		PolicyExt:  ".md",
	}
	switch targetName {
	case "cursor":
		ctx.PolicyExt = ".mdc"
	case "antigravity":
		ctx.RenameProjectInstructions = true
	}
	return ctx
}
```

*   **Logic**（`resolvePolicyPath` 置換後）:
    1. `ext := ctx.PolicyExt`; 空なら `".md"`。
    2. `filename := id + ext`。
    3. `id == "project-instructions"` かつ `ctx.RenameProjectInstructions` のときだけ `filename = "instructions" + ext`。
    4. `return ensureTrailingSlash(ctx.Paths.Rules) + filename`。
    5. `resolveRef` の procedure / capability / target 分岐は現行ロジックを維持（Workflows 非空なら `{workflows}{id}.md`、空なら `{skills}{id}/SKILL.md`）。

#### [MODIFY] TargetPaths 抽出ヘルパー（`features/tt/internal/prompt/emitter/` 内の既存共通ファイル、または `paths_helpers.go` を NEW）

*   **Description**: ターゲット entity の `paths` マップから `TargetPaths` を抽出する共通ヘルパーを用意し、各 emitter の重複を減らす。
*   **Technical Design**:

```go
// ExtractTargetPaths merges defaults with optional overrides from target.Raw["paths"].
func ExtractTargetPaths(target *manifest.Entity, defaults TargetPaths) TargetPaths {
	tp := defaults
	if target == nil {
		return normalizeTargetPaths(tp)
	}
	paths, ok := target.Raw["paths"].(map[string]any)
	if !ok {
		return normalizeTargetPaths(tp)
	}
	if r, ok := paths["rules"].(string); ok {
		tp.Rules = r
	}
	if s, ok := paths["skills"].(string); ok {
		tp.Skills = s
	}
	if w, ok := paths["workflows"].(string); ok {
		tp.Workflows = w
	}
	return normalizeTargetPaths(tp)
}

func normalizeTargetPaths(tp TargetPaths) TargetPaths {
	if tp.Rules != "" {
		tp.Rules = ensureTrailingSlash(tp.Rules)
	}
	if tp.Skills != "" {
		tp.Skills = ensureTrailingSlash(tp.Skills)
	}
	if tp.Workflows != "" {
		tp.Workflows = ensureTrailingSlash(tp.Workflows)
	}
	return tp
}
```

*   **Logic**: Antigravity の `resolveTargetPaths` は本ヘルパー呼び出しに置き換える（デフォルト `.agent/rules|skills|workflows/`）。Claude/Codex/Cursor は Workflows デフォルト空文字のまま。

### emitter（各ターゲット Emit）

#### [MODIFY] [features/tt/internal/prompt/emitter/claude_code_test.go](file://features/tt/internal/prompt/emitter/claude_code_test.go)

*   **Description**: Emit 後の body にテンプレート変数が残らず、Claude パスへ展開されることを断言するテストを追加（TDD: 実装前は失敗）。
*   **Technical Design**:
    *   新規: `TestClaudeCodeEmitter_ResolvesTemplateVars`
    *   fixture: selected な policy `project-instructions` の body に `See {{policy:testing-rules}} and {{procedure:build-pipeline}}` と、selected な procedure `build-pipeline` / policy `testing-rules` を最低限含める。
*   **Logic**:
    1. `Emit(..., apply=false, ...)` 後、該当 rules ファイルを読む。
    2. `assert.NotContains(t, content, "{{policy:testing-rules}}")`
    3. `assert.Contains(t, content, ".claude/rules/testing-rules.md")`
    4. `assert.Contains(t, content, ".claude/skills/build-pipeline/SKILL.md")`
    5. procedure skill 側 body も同様に置換されていること。

#### [MODIFY] [features/tt/internal/prompt/emitter/claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go)

*   **Description**: Emit 開始時に `TemplateContext` を構築し、policy / capability / procedure の各 body に `ResolveTemplateVars` を適用する。
*   **Technical Design**:
    *   `Emit` 冒頭（`inc := ExtractIncludes(...)` の直後付近）で:

```go
tmplCtx := NewTemplateContext("claude-code", ExtractTargetPaths(claudeTarget, TargetPaths{
	Rules:  ".claude/rules/",
	Skills: ".claude/skills/",
}))
```

*   **Logic**:
    1. policy ループ: `stripFrontmatter` → CRLF 正規化 → `body = ResolveTemplateVars(body, tmplCtx)` → frontmatter 付与。
    2. capability / procedure ループも同じ順で `ResolveTemplateVars` を入れる。
    3. Antigravity と同じ位置（正規化直後）に揃える。

#### [MODIFY] [features/tt/internal/prompt/emitter/codex_test.go](file://features/tt/internal/prompt/emitter/codex_test.go) / [codex.go](file://features/tt/internal/prompt/emitter/codex.go)

*   **Description / Logic**: Claude と同型。期待パスは `.codex/rules/...` / `.codex/skills/.../SKILL.md`。`NewTemplateContext("codex", ...)`。

#### [MODIFY] [features/tt/internal/prompt/emitter/cursor_test.go](file://features/tt/internal/prompt/emitter/cursor_test.go) / [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)

*   **Description / Logic**: 同型。期待パスは `.cursor/rules/testing-rules.mdc` / `.cursor/skills/build-pipeline/SKILL.md`。`NewTemplateContext("cursor", ...)`（`PolicyExt=.mdc`）。

#### [MODIFY] [features/tt/internal/prompt/emitter/antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)

*   **Description**: 既存 `tmplCtx` 構築を `NewTemplateContext("antigravity", ExtractTargetPaths(...))` に寄せ、回帰を防ぐ。
*   **Logic**: `RenameProjectInstructions=true` により既存の `instructions.md` 解決を維持。`emitter_test.go` の既存テンプレート展開アサーションが通ることを確認。

### prompt ソース可搬化（R3）

#### [MODIFY] policies（3系統を同一内容で同期）

対象ファイル（各ツリーで同名 4 ファイル）:

1. `prompts/manifest/code_content/policies/`
2. `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/policies/`
3. `features/tt/internal/prompt/compiler/testdata/catalog_template/prompts/manifest/code_content/policies/`

##### project-instructions.md

| 変更前 | 変更後 |
| :--- | :--- |
| 「`.agent/workflows/` 配下に定義されたワークフローは、以下の順序で連携して動作します:」 | 「プロジェクトのワークフロー（procedure）は、以下の順序で連携して動作します:」 |
| `[create-specification.md](.agent/workflows/create-specification.md)` | `[{{procedure:create-specification}}]({{procedure:create-specification}})` |
| create-implementation-plan / execute-implementation-plan / build-pipeline / investigate の同様リンク | 対応する `{{procedure:...}}` |

（既存の `{{policy:coding-rules}}` / `{{policy:testing-rules}}` はそのまま維持）

##### coding-rules.md

| 変更前 | 変更後 |
| :--- | :--- |
| 「一連の作業手順…は `.agent/workflows/` に定義し、定型化する。」 | 「一連の作業手順…は procedure（ワークフロー）として定義し、定型化する。」 |
| 「`.agent/workflows/build-pipeline.md` に相当するフロー」 | 「`{{procedure:build-pipeline}}` に相当するフロー」 |

##### planning-rules.md

| 変更前 | 変更後 |
| :--- | :--- |
| 「本プロジェクトは `.agent/workflows/` および `scripts/` を活用した…」 | 「本プロジェクトはワークフロー（procedure）および `scripts/` を活用した…」 |

##### testing-rules.md

| 変更前 | 変更後 |
| :--- | :--- |
| 表セル「`.agent/workflows/build-pipeline.md`」 | 「`{{procedure:build-pipeline}}`」 |

*   **完了条件（ソース）**: 上記 3 ツリーの policies に対し `grep '\.agent/workflows'` が 0 件。

### 統合テスト（CLI / ファイルシステム）

#### [NEW] [tests/tt/tt_prompt_template_vars_test.go](file://tests/tt/tt_prompt_template_vars_test.go)

*   **Description**: `tt prompt update --force --target <name>` 後の成果物を読み、仕様の検証シナリオ 3〜7 を自動化する（本機能の E2E 相当。GUI E2E は対象外）。
*   **Technical Design**:
    *   `//go:build integration` + 既存 `runTTInDir` / catalog fixture 流用。
    *   fixture は `features/tt/internal/prompt/compiler/testdata/catalog_template` をコピー（ハードコード除去後のソースが入っていること）。
*   **テストケース**:

| 関数 | 検証内容 |
| :--- | :--- |
| `TestPromptTemplateVars_ClaudeResolvesPolicyAndProcedure` | update `--target claude-code` 後、`.claude/skills/build-pipeline/SKILL.md` に `.claude/rules/testing-rules.md` があり `{{policy:` が無い。`.claude/rules/project-instructions.md` に `.agent/workflows` が無く `.claude/skills/build-pipeline/SKILL.md` を含む |
| `TestPromptTemplateVars_CodexResolves` | 同上（`.codex/...`） |
| `TestPromptTemplateVars_CursorResolvesMdc` | `.cursor/skills/build-pipeline/SKILL.md` が `.cursor/rules/testing-rules.mdc` を参照 |
| `TestPromptTemplateVars_AntigravityRegression` | `.agent/workflows/build-pipeline.md` が `.agent/rules/testing-rules.md` を参照し、`instructions.md` 命名が維持される |

#### [MODIFY] [features/tt/internal/prompt/compiler/update_test.go](file://features/tt/internal/prompt/compiler/update_test.go)（任意強化）

*   **Description**: 既存 `TestUpdate_CatalogTemplate_AllTargets` に、少なくとも 1 ターゲット分の「未展開 `{{policy:` が無い」アサーションを追加して早期検知する。

## Step-by-Step Implementation Guide

1. **Failing unit tests for template naming (TDD)**:
    *   Edit `template_test.go` に R2/R5 のテーブルを追加。
    *   `./scripts/process/build.sh --skip-frontend --skip-etc` で失敗することを確認。
2. **Implement TemplateContext + resolvePolicyPath**:
    *   Edit `template.go` にフィールド / `NewTemplateContext` / `resolvePolicyPath` 更新。
    *   同ビルドで `template` 関連テストが通ることを確認。
3. **Shared ExtractTargetPaths**:
    *   ヘルパー追加し、Antigravity の `resolveTargetPaths` を置換。
4. **Failing emitter tests (TDD)**:
    *   `claude_code_test.go` / `codex_test.go` / `cursor_test.go` に `ResolvesTemplateVars` を追加し、失敗を確認。
5. **Wire ResolveTemplateVars into Emit**:
    *   `claude_code.go` / `codex.go` / `cursor.go`（および antigravity の ctx 構築統一）を実装。
    *   emitter 単体テストが全て通ることを確認。
6. **Portable policy sources**:
    *   workspace / catalog originals / compiler `catalog_template` の 4 policy を仕様どおり置換。
    *   各 policies ディレクトリで `.agent/workflows` が 0 件であることを確認。
7. **Integration / E2E (tt CLI)**:
    *   `tests/tt/tt_prompt_template_vars_test.go` を追加。
    *   （任意）`update_test.go` に未展開チェックを追加。
8. **Run Verification Plan**（下記）を実行し、§12 総合判定まで完了する。

## Verification Plan

### テスト項目設計（§11）

#### ボトムアップ順序

| Step | 層 | テスト | 確認すること |
| :---: | :--- | :--- | :--- |
| 1 | C: `ResolveTemplateVars` | `template_test.go` | ターゲット別パス・拡張子・リネーム・未知 kind 放置 |
| 2 | B: 各 `Emit` | `claude_code_test` / `codex_test` / `cursor_test` / antigravity 回帰 | body 書き出し時に解決が入り、実ファイル内容に期待パスが現れる |
| 3 | A: CLI update | `tests/tt/tt_prompt_template_vars_test.go` | `tt prompt update` 経由で FS 上の成果物が仕様シナリオどおり |

#### 観点チェックリスト（§11.3）

| # | 観点 | 本計画でのカバー |
| :---: | :--- | :--- |
| 1 | 正常系 | 4 ターゲットの policy/procedure 解決アサーション |
| 2 | 異常系・境界 | 未知 `{{unknown:foo}}` は残す（既存） |
| 3 | 外部連携 | 統合テストで実 FS へ update しファイル読取 |
| 4 | データ一貫性 | 解決後パス上に対応ファイルが存在する（Emit で生成） |
| 5 | 状態遷移 | update 前後で未展開 → 展開済みへ変化 |
| 6 | 設定・構成 | target YAML の paths / デフォルト paths が反映 |
| 7 | 副作用 | 他ターゲットへ誤パスを埋め込まない（`--target` 別実行） |

#### セルフレビュー結果（§11.4）

1. **網羅性**: Emit 未呼び出し（本バグの直接原因）とパス命名不一致、ソースハードコードの 3 系統をそれぞれテスト層で押さえている → 十分。
2. **証拠の十分性**: 「エラー無し」ではなく Contains/NotContains で具体パス文字列を検証。
3. **迂回排除**: Claude テストは `.claude/` パスを要求し、Antigravity パス混入を NotContains で拒否。
4. **依存関係**: template → emitter → CLI の順で積み上げ。

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests（tt カテゴリに限定）**:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "PromptTemplateVars|PromptPaths|PromptTags"
    ```
    *   **Log Verification**:
        *   `TestPromptTemplateVars_*` が全て PASS。
        *   ログに `SKIP` / 予期せぬ `WARN` が無いこと。
        *   失敗時は該当ターゲットの成果物パスと期待文字列が require メッセージに出ること。

3. **E2E Tests**:
    *   GUI E2E は不要（本変更は prompt コンパイラ / CLI のみ。画面・拡張機能に触れない）。
    *   CLI レベルの E2E は `tests/tt/tt_prompt_template_vars_test.go` で代替する（上記 Integration に含む）。

### 総合判定プロセス（§12）

全自動テスト成功後、実装実行エージェントは次を実施し、walkthrough または計画の進捗コメントに **総合判定結果** を記録する:

| # | チェック項目 | 確認内容 |
| :---: | :--- | :--- |
| 1 | スキップ | 新規テストが build tag / 環境条件で実質スキップされていないか |
| 2 | 部分エラー | update ログに validation 握りつぶし等がないか |
| 3 | 迂回 | Antigravity だけ通って Claude が未検証のままになっていないか（4 ターゲット分のテスト有無） |
| 4 | 設定誤適用 | Cursor で `.md` のままになっていないか（`.mdc` アサーション） |
| 5 | 順序依存 | `PromptTemplateVars` を単独 `--specify` でも通るか |
| 6 | カバレッジ | R3 の 3 ツリー同期漏れが無いか（catalog / testdata） |
| 7 | 外部状態 | 一時ディレクトリ fixture のみ使用し、ワークスペース汚染が無いか |

判定フォーマットは `testing-rules` §12.3 に従う。

## Documentation

#### [MODIFY] （該当する場合のみ）既存フェーズ docs

*   `prompts/specifications` 配下に本機能の独立仕様が無ければ追加更新は不要。
*   本変更の正本は Source Specification（ideas/000-…）と本計画とする。
*   実装完了後、必要なら `030-Restructure-Agent-Target-Paths.md` 側に「全 emitter で解決済み」の実装メモを追記してよい（任意）。
