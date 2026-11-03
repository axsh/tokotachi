# 026-Template-Ref-Backslash-Escape

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/025-Template-Ref-Backslash-Escape.md

## Goal Description

`ResolveTemplateVars` にバックスラッシュ・エスケープ `\{{kind:id}}` → リテラル `{{kind:id}}` を追加し、ドキュメント例示が emit 時にパス置換されないようにする。マニュアルと `prompt-manifest-update` も更新する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1 `\{{kind:id}}` → リテラル | Proposed Changes > template.go / template_test.go |
| R2 エスケープを解決より先に確定 | template.go 単一パス regex（グループ1が `\`） |
| R3 `\\{{...}}` → `\` + 解決パス | template_test.go ケース + regex の「直前1つの `\`」 |
| R4 既存解決の維持 | 既存テーブルケース維持 |
| R5 マニュアル・ドキュメント | TemplateRefManual / prompt-manifest-update.md |
| R6 回帰テスト | template_test.go + tt_prompt_template_refs_test.go |
| O1 二重バックスラッシュのマニュアル例 | TemplateRefManual Escape 節に1行 |

## Proposed Changes

### emitter (template resolution)

#### [MODIFY] [template_test.go](file://features/tt/internal/prompt/emitter/template_test.go)

*   **Description**: TDD — エスケープ用例を既存 `TestResolveTemplateVars` 系テーブルに追加し、先に失敗させる。マニュアル断言を拡張。
*   **Technical Design**:
    *   Antigravity 既定コンテキスト（既存テーブル先頭の ctx）および Cursor コンテキスト双方でエスケープを検証する。
*   **Logic**:
    *   追加ケース例（Go 文字列リテラル）:
        *   `input: "Docs: \\{{policy:coding-rules}}."` → `want: "Docs: {{policy:coding-rules}}."`
        *   `input: "\\{{unknown:foo}}"` → `want: "{{unknown:foo}}"`
        *   混在: `"Esc \\{{policy:coding-rules}} live {{procedure:arch-correct}}."` → `"Esc {{policy:coding-rules}} live .agents/skills/arch-correct/SKILL.md."`（既存 AG paths に合わせる）
        *   二重バックスラッシュ: `input: "\\\\{{policy:coding-rules}}"` → `want: "\\.agents/rules/coding-rules.md"`（既存 AG rename なしの coding-rules）
    *   Cursor 専用サブテストまたは既存 Cursor テーブルへ:
        *   `"\\{{policy:coding-rules}}"` → `"{{policy:coding-rules}}"`
        *   `"\\\\{{policy:coding-rules}}"` → `"\\.cursor/rules/coding-rules.mdc"`
    *   `TestTemplateRefManual_ContainsRequiredSections` の `required` に `"\\{{"` と `"Escape"`（またはマニュアル見出しに使う固定語 `escape`）を追加。

#### [MODIFY] [template.go](file://features/tt/internal/prompt/emitter/template.go)

*   **Description**: 単一パスでエスケープと解決を処理。マニュアルに Escape 節を追加。
*   **Technical Design**:
    *   既存 `templateVarRegex`（`\{\{(\w+):([\w][\w-]*)\}\}`）は Pass 内の「素の参照」検出に維持する。
    *   `ResolveTemplateVars` は **左から右へ 1 バイトずつ走査**し、次の優先順位で消費する（R2/R3）:
        ```go
        func ResolveTemplateVars(text string, ctx *TemplateContext) string {
            if ctx == nil {
                return text
            }
            var b strings.Builder
            b.Grow(len(text))
            i := 0
            for i < len(text) {
                // 1) escaped backslash: \\  -> emit one '\'
                if text[i] == '\\' && i+1 < len(text) && text[i+1] == '\\' {
                    b.WriteByte('\\')
                    i += 2
                    continue
                }
                // 2) escaped ref: \{{kind:id}} -> emit {{kind:id}}
                if text[i] == '\\' {
                    if loc := templateVarRegex.FindStringSubmatchIndex(text[i+1:]); loc != nil && loc[0] == 0 {
                        // text[i+1:] starts with {{kind:id}}
                        b.WriteString(text[i+1 : i+1+loc[1]]) // write {{kind:id}} without '\'
                        i += 1 + loc[1]
                        continue
                    }
                }
                // 3) plain ref: {{kind:id}} -> resolve or leave
                if loc := templateVarRegex.FindStringSubmatchIndex(text[i:]); loc != nil && loc[0] == 0 {
                    match := text[i : i+loc[1]]
                    subs := templateVarRegex.FindStringSubmatch(match)
                    kind, id := subs[1], subs[2]
                    if resolved := resolveRef(kind, id, ctx); resolved != "" {
                        b.WriteString(resolved)
                    } else {
                        b.WriteString(match)
                    }
                    i += loc[1]
                    continue
                }
                b.WriteByte(text[i])
                i++
            }
            return b.String()
        }
        ```
    *   **Logic (R2/R3)**:
        *   `\{{policy:x}}` → 枝 2 → `{{policy:x}}`
        *   `\\{{policy:x}}` → 枝 1 で `\` を1つ出力し、残り `{{policy:x}}` を枝 3 で解決 → `\` + path（R3）
        *   `\\\{{policy:x}}` → 枝 1 で `\`、残り `\{{policy:x}}` を枝 2 → `\{{policy:x}}`
        *   任意グループ `(\\?)` の一発 ReplaceAll は R3 を満たせないため採用しない。
    *   `TemplateRefManual` に次の節を Format の直後へ挿入:
        ```
        Escape (documentation)
        ----------------------
          \{{kind:id}}     -> remains {{kind:id}} after emit (not resolved)
          \\{{kind:id}}    -> one literal backslash + resolved path

        Use a single backslash before {{ when showing the template syntax
        itself in manifest bodies (e.g. prompt-manifest-update).
        ```

#### [MODIFY] [tt_prompt_template_refs_test.go](file://tests/tt/tt_prompt_template_refs_test.go)

*   **Description**: emit 経路でエスケープがリテラルとして残る統合テストを追加。既存 `NotContains "{{policy:"` を壊さないよう **別テスト**にする。
*   **Logic**:
    *   新規 `TestPromptTemplateRefs_EscapedRefRemainsLiteral`:
        1. `writeTemplateRefFixture` を呼ぶか、同等フィクスチャを用意。
        2. procedure `demo-proc` 本文を上書き、または専用 procedure を追加:
           ```
           Escaped: \{{policy:coding-rules}}
           Live: {{policy:coding-rules}}
           ```
        3. `tt prompt deploy --target cursor --force`
        4. 成果物に `Escaped: {{policy:coding-rules}}` と `Live: .cursor/rules/coding-rules.mdc` が含まれること。
        5. `Escaped:` 行がパスに置換されていないこと。
    *   `TestPromptInstruction_PrintsManual` に `assert.Contains(t, stdout, `\{{`)` または `"Escape"` を追加。

#### [MODIFY] [prompt-manifest-update.md](file://catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/capabilities/prompt-manifest-update.md)

*   **Description**: エンティティ参照の例示をエスケープし、エスケープ規則を一文追加。
*   **Logic**:
    *   例示行を例えば次へ:
        ```
        - **エンティティ参照 (`\{{kind:id}}`)**: ...
          - 参照の書き方例: `\{{policy:coding-rules}}` / `\{{procedure:test-generator}}` / ...
          - 実際にパスへ展開したい箇所はバックスラッシュ無しの `{{policy:...}}` を使う
          - 記法そのものを本文に残すときは `\{{kind:id}}` とエスケープする
        ```
    *   作業ツリー `prompts/` に同ファイルが無いためカタログ原本のみ更新（現状 Glob 結果どおり）。

## Step-by-Step Implementation Guide

1. **[x] Red — unit tests**: Edit `features/tt/internal/prompt/emitter/template_test.go` にエスケープ / 二重バックスラッシュ / 混在 / マニュアル断言を追加。`go test` で失敗を確認。
2. **[x] Green — ResolveTemplateVars**: Edit `template.go` に左から右への走査実装を入れる。単体テストを通す。
3. **[x] Manual text**: `TemplateRefManual` に Escape 節を追加。マニュアル・テストを通す。
4. **[x] Docs**: `prompt-manifest-update.md` の例示をエスケープ対応に更新。
5. **[x] Integration test**: `tests/tt/tt_prompt_template_refs_test.go` にエスケープ用テストと instruction 断言を追加。
6. **[x] Verify**: Verification Plan を実行し、総合判定 (§12) を行う。

### 総合判定結果

**判定**: 動作確認完了

#### テスト結果サマリ
- `./scripts/process/build.sh`: PASS
- emitter / cmd 単体: PASS（エスケープ・二重 `\`・混在・マニュアル）
- `./scripts/process/integration_test.sh --categories "tt" --specify "TemplateRef|PromptInstruction"`: PASS（4 tests）
- スキップ / WARN による偽成功なし。エスケープ成果物は `Escaped: {{policy:...}}` と Live パスを直接断言。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Focused unit tests**:
    ```bash
    go test ./features/tt/internal/prompt/emitter/... ./features/tt/cmd/... -count=1
    ```

3. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "common" --specify "TemplateRef|PromptInstruction"
    ```
    *   **Log Verification**: `TestPromptTemplateRefs_EscapedRefRemainsLiteral` および既存 TemplateRefs / PromptInstruction が PASS。

4. **E2E Tests**:
    *   本変更は `tt prompt` の文字列変換とドキュメント更新のみ。VSCode GUI E2E は不要。
    *   CLI 経路の検証は `tests/tt/tt_prompt_template_refs_test.go`（既存の tt 統合テストインフラ）で実施する。追加の `tests/agentservice_e2e_test.go` 系は不要（理由: IDE/GUI 非関与）。

### テスト項目設計（§11）

ボトムアップ順序:

1. **C**: `ResolveTemplateVars` 単体（エスケープ / 解決 / 二重 `\` / 混在 / 未知 kind）
2. **B**: `TemplateRefManual` 文字列
3. **A**: `tt prompt deploy` / `tt prompt instruction` 統合

観点チェックリスト: 1 正常系 ✓ / 2 境界（未知 kind エスケープ、二重 `\`）✓ / 3 外部（ファイル emit）✓ / 4 データ一貫（リテラル往復）✓ / 5–7 本変更では状態機械・設定切替・リークは非該当（N/A）

### セルフレビュー結果（§11.4）

1. **網羅性**: エスケープ・非エスケープ・混在・二重 `\`・マニュアル・emit 成果物をカバーすれば「記法説明がパスに化けない」と言い切れる。
2. **証拠**: `Contains` / 完全一致 `want` で期待文字列を直接検証する。
3. **迂回排除**: 統合テストは実バイナリ `tt prompt deploy` 経由。単体は `ResolveTemplateVars` 直接。
4. **依存**: 単体成功後に統合を実行する計画。

### 総合判定プロセス（§12）

全テスト成功後、計画実行エージェントは testing-rules §12 のチェック項目（スキップ、WARN、偽成功、カバレッジ）を実施し、実装計画またはコミットメッセージに判定を残す。

## Documentation

#### [MODIFY] [prompt-manifest-update.md](file://catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/capabilities/prompt-manifest-update.md)
*   **更新内容**: エンティティ参照の例示を `\{{...}}` に変更し、エスケープ規則を追記。

#### [MODIFY] [template.go TemplateRefManual](file://features/tt/internal/prompt/emitter/template.go)
*   **更新内容**: Escape 節を追加（CLI マニュアルの単一の説明源）。
)
