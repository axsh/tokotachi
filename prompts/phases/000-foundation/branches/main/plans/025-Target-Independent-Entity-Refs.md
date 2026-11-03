# 025-Target-Independent-Entity-Refs

> **Source Specification**: [024-Target-Independent-Entity-Refs.md](prompts/phases/000-foundation/branches/main/ideas/024-Target-Independent-Entity-Refs.md)

## Goal Description

マニフェスト本文の `{{kind:id}}` 参照を全ターゲット emitter（Cursor / Codex / Claude Code / Antigravity）で解決し、policy パス規約をターゲット実レイアウトに一致させる。あわせて `tt prompt instruction` とヘルプ案内でテンプレート仕様マニュアルを CLI から参照可能にする。

## User Review Required

None. ユーザーより create-implementation-plan → execute-implementation-plan の連続実行指示あり。任意要件 O1/O3 は先送り。O2 は TemplateContext 構築の共通化として本計画に含める。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: 全 emitter で ResolveTemplateVars | Proposed Changes > cursor/codex/claude_code/antigravity emitters |
| R2: policy/procedure/capability/target | Proposed Changes > `template.go` `resolveRef` |
| R3: ターゲット別パス規約 | Proposed Changes > `TemplateContext` フィールド + ヘルパー |
| R4: procedure 参照が成果物で解決 | emitter 単体テスト + 統合テスト `tt_prompt_template_refs_test.go` |
| R5: prompt-manifest-update 更新 | Proposed Changes > catalog/prompts capability MD |
| R6: tt / tt prompt ヘルプ案内 | Proposed Changes > `prompt.go` Short/Long |
| R7: `tt prompt instruction` | Proposed Changes > `TemplateRefManual` + `promptInstructionCmd` |
| O1: 未解決 warn | 先送り |
| O2: TemplateContext 共通化 | Proposed Changes > `BuildTemplateContext` |
| O3: instruction --target | 先送り |
| S1–S9 (検証シナリオ) | Verification Plan |

## Proposed Changes

### features/tt/internal/prompt/emitter（テンプレート解決）

#### [MODIFY] [template_test.go](features/tt/internal/prompt/emitter/template_test.go)

*   **Description**: TDD Red。ターゲット別コンテキストの期待パスを先に固定する。
*   **Technical Design**:
    ```go
    func TestResolveTemplateVars_CursorPolicyExt(t *testing.T)
    // {{policy:coding-rules}} → .cursor/rules/coding-rules.mdc
    // {{policy:project-instructions}} → .cursor/rules/project-instructions.mdc
    // {{procedure:test-generator}} → .cursor/skills/test-generator/SKILL.md
    // {{capability:portable-file-references}} → .cursor/skills/portable-file-references/SKILL.md

    func TestResolveTemplateVars_AntigravityPolicyRename(t *testing.T)
    // {{policy:project-instructions}} → .agent/rules/instructions.md
    // {{procedure:test-generator}} → .agent/workflows/test-generator.md（Workflows 設定時）

    func TestResolveTemplateVars_CodexAndClaude(t *testing.T)
    // policy → .codex/rules/coding-rules.md / .claude/rules/coding-rules.md
    // project-instructions は rename なし
    // procedure → skills/{id}/SKILL.md（Workflows 空）

    func TestTemplateRefManual_ContainsRequiredSections(t *testing.T)
    // TemplateRefManual() に {{kind:id}}, policy, procedure, capability, target,
    // emit, .mdc, instructions.md, unknown, memory が含まれること
    ```
*   **Logic**:
    *   既存 `TestResolveTemplateVars` の AG 寄りの期待（`instructions.md`）は、Antigravity 用コンテキストを明示して維持する。
    *   デフォルトフィクスチャが Cursor 相当なら期待パスを `.mdc` に更新する（破壊的変更をテストで先に表す）。

#### [MODIFY] [template.go](features/tt/internal/prompt/emitter/template.go)

*   **Description**: ターゲット別 policy 命名を `TemplateContext` に持たせ、`resolvePolicyPath` を更新する。マニュアル文字列とコンテキスト構築ヘルパーを追加する。
*   **Technical Design**:
    ```go
    type TemplateContext struct {
        Paths      TargetPaths
        MemBase    string
        TargetName string
        // PolicyExt is the file extension for policy outputs, including the dot.
        // Cursor: ".mdc"; others: ".md".
        PolicyExt string
        // RenameProjectInstructionsTo, when non-empty, replaces the filename
        // for id "project-instructions" (Antigravity: "instructions.md").
        RenameProjectInstructionsTo string
    }

    func BuildTemplateContext(targetID string, paths TargetPaths) *TemplateContext
    func resolvePolicyPath(id string, ctx *TemplateContext) string
    func TemplateRefManual() string
    ```
*   **Logic**:
    *   `BuildTemplateContext`:
        *   `paths` をそのまま設定。`MemBase = "prompts/memory"`。`TargetName = targetID`（`ag` エイリアスは呼び出し側で正規化済みの `antigravity` を渡す）。
        *   `targetID == "cursor"` → `PolicyExt = ".mdc"`, `RenameProjectInstructionsTo = ""`。
        *   `targetID == "antigravity"` → `PolicyExt = ".md"`, `RenameProjectInstructionsTo = "instructions.md"`。
        *   それ以外（`codex`, `claude-code` 含む）→ `PolicyExt = ".md"`, `RenameProjectInstructionsTo = ""`。
        *   `PolicyExt` が空のレガシー呼び出し互換: `".md"` 扱い。
    *   `resolvePolicyPath`:
        ```go
        filename := id + ctx.PolicyExt // PolicyExt 空なら ".md"
        if id == "project-instructions" && ctx.RenameProjectInstructionsTo != "" {
            filename = ctx.RenameProjectInstructionsTo
        }
        return ensureTrailingSlash(ctx.Paths.Rules) + filename
        ```
    *   `resolveRef` の `procedure` / `capability` / `target` / `default` は現行ロジック維持（Workflows 非空なら `{workflows}{id}.md`、否则 `{skills}{id}/SKILL.md`）。
    *   `TemplateRefManual()`: 英語または日英混在でよいが、仕様 R7 の必須項目をすべて含む固定文字列を返す（ネットワーク・FS 非依存）。内容は R2/R3 の表と一致させる。

### features/tt/internal/prompt/emitter（各ターゲット）

#### [MODIFY] [cursor_test.go](features/tt/internal/prompt/emitter/cursor_test.go)（および必要なら codex/claude/antigravity テスト）

*   **Description**: body にテンプレート参照を含む emit フィクスチャで解決を検証する（TDD）。
*   **Logic**:
    *   policy body に `See {{procedure:test-proc}} and {{policy:coding-rules}} and {{capability:test-skill}}.` を含める。
    *   Cursor Emit 後、rules の `.mdc` 本文に次が含まれること:
        *   `.cursor/skills/test-proc/SKILL.md`
        *   `.cursor/rules/coding-rules.mdc`
        *   `.cursor/skills/test-skill/SKILL.md`
    *   リテラル `{{procedure:test-proc}}` が残らないこと。
    *   Antigravity 既存テスト（`emitter_test.go`）は既に解決を期待しているため、`.mdc` 変更の影響が無いことを確認。期待パスが `instructions.md` のままであること。

#### [MODIFY] [cursor.go](features/tt/internal/prompt/emitter/cursor.go) / [codex.go](features/tt/internal/prompt/emitter/codex.go) / [claude_code.go](features/tt/internal/prompt/emitter/claude_code.go) / [antigravity.go](features/tt/internal/prompt/emitter/antigravity.go)

*   **Description**: Emit 開始時に `BuildTemplateContext` を構築し、policy / capability / procedure の body パイプラインで `ResolveTemplateVars` を呼ぶ。
*   **Logic**（各 kind ループ、共通）:
    1. Emit 冒頭: target YAML から `TargetPaths` を抽出（Cursor/Codex/Claude は現状 `resolvePaths` のみなので、相対パス文字列版の抽出を追加。Antigravity は既存 `resolveTargetPaths` を流用可）。
    2. `tmplCtx := BuildTemplateContext("<target-id>", paths)`。
    3. body 処理:
       ```
       body = stripFrontmatter(body)
       body = strings.ReplaceAll(body, "\r\n", "\n")
       body = ResolveTemplateVars(body, tmplCtx)
       // capability のみ:
       body = RewriteBundlePaths(body, bundleEntries)  // ResolveTemplateVars の後
       ```
    4. Antigravity: 既存の手組み `TemplateContext` を `BuildTemplateContext("antigravity", a.resolveTargetPaths(resolved))` に置き換え（挙動同等）。

*   **TargetPaths 抽出（Cursor 例）**:
    ```go
    func (c *CursorEmitter) resolveTargetPaths(resolved *manifest.ResolvedManifest) TargetPaths {
        tp := TargetPaths{Rules: ".cursor/rules/", Skills: ".cursor/skills/"}
        // target.ID == "cursor" の paths.rules / paths.skills で上書き（ensureTrailingSlash）
        return tp
    }
    ```
    Codex / Claude Code も同様（デフォルト `.codex/...`, `.claude/...`）。Workflows は空のまま（procedure → skills）。

### features/tt/cmd（CLI）

#### [NEW] [prompt_instruction_test.go](features/tt/cmd/prompt_instruction_test.go)

*   **Description**: ヘルプ案内と instruction 出力の単体テスト。
*   **Technical Design**:
    ```go
    func TestPromptCmd_HelpMentionsInstruction(t *testing.T)
    // promptCmd.Short / Long に "instruction" と "{{kind:id}}" 相当案内が含まれる

    func TestPromptInstructionCmd_PrintsManual(t *testing.T)
    // promptInstructionCmd を SetOut して Execute、stdout に必須キーワード
    ```

#### [MODIFY] [prompt.go](features/tt/cmd/prompt.go)

*   **Description**: Short/Long 更新と `instruction` サブコマンド追加。
*   **Technical Design**:
    ```go
    var promptCmd = &cobra.Command{
        Use:   "prompt",
        Short: "Manage prompt manifests; see 'tt prompt instruction' for {{kind:id}} template refs",
        Long: `Manage prompt manifest compilation and deployment.

Manifest bodies may use target-independent template references of the form {{kind:id}}
(policy, procedure, capability, target). They are resolved to agent-specific paths at emit time.
For the full template reference manual, run:  tt prompt instruction`,
    }

    var promptInstructionCmd = &cobra.Command{
        Use:   "instruction",
        Short: "Print the {{kind:id}} template reference manual",
        Long:  "Print the target-independent {{kind:id}} template reference manual to stdout.",
        RunE:  runPromptInstruction,
        Args:  cobra.NoArgs,
    }

    func runPromptInstruction(cmd *cobra.Command, args []string) error {
        _, err := fmt.Fprint(cmd.OutOrStdout(), emitter.TemplateRefManual())
        return err
    }
    ```
*   **Logic**:
    *   `init()` で `promptCmd.AddCommand(promptInstructionCmd)`（path/tag フラグは付けない）。
    *   Short は `tt --help` の一行に収まる長さを意識しつつ、`instruction` 誘導を必ず含める。

### tests/tt（統合）

#### [NEW] [tt_prompt_template_refs_test.go](tests/tt/tt_prompt_template_refs_test.go)

*   **Description**: CLI 経由で deploy 後にテンプレートが解決されること、および `instruction` / help を検証する。
*   **Logic**:
    1. temp workspace に最小 manifest（policy + procedure + capability + cursor/antigravity targets + schemas）を配置。
    2. procedure body に `Read {{policy:coding-rules}} then {{capability:demo-cap}} and {{procedure:demo-proc}}.` を含める。
    3. `runTTInDir(..., "prompt", "deploy", "--target", "cursor", "--force")` → skills/demo-proc/SKILL.md に `.cursor/rules/coding-rules.mdc` 等が含まれ、`{{policy:` が残らない。
    4. `runTTInDir(..., "prompt", "deploy", "--target", "ag", "--force")` → workflows または skills 側で `.agent/rules/...` に解決。
    5. `runTT(t, "prompt", "instruction")` exit 0、stdout に必須セクション。
    6. `runTT(t, "--help")` / `runTT(t, "prompt", "--help")` に `instruction` が含まれる。

### ドキュメント

#### [MODIFY] catalog / prompts の `prompt-manifest-update.md`

*   **更新内容**: 「policy 間参照」を「エンティティ参照 (`{{kind:id}}`)」に拡張。`policy` / `procedure` / `capability` / `target` を列挙し、emit 時解決・`tt prompt instruction` への誘導を追記。作業ツリーにファイルが無い場合は catalog 原本を更新し、prompts 側が catalog 由来なら同様に揃える。

## Step-by-Step Implementation Guide

1. **[/] Red: template_test をターゲット別に更新/追加**
   *   Edit `features/tt/internal/prompt/emitter/template_test.go`。
2. **[/] Green: TemplateContext / BuildTemplateContext / resolvePolicyPath / TemplateRefManual**
   *   Edit `features/tt/internal/prompt/emitter/template.go`。
3. **[/] Red: cursor（代表）emitter テストにテンプレート解決ケース追加**
4. **[/] Green: 全 emitter に tmplCtx + ResolveTemplateVars（capability は RewriteBundlePaths の前）**
5. **[/] Red/Green: prompt_instruction_test + prompt.go Short/Long/instruction**
6. **[/] 統合テスト `tests/tt/tt_prompt_template_refs_test.go`**
7. **[/] prompt-manifest-update.md 更新（catalog + 必要なら prompts）**
8. **[ ] Verification Plan を実行し、総合判定を記録**

各ステップ完了ごとに `git add` → `git commit`。

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Integration Tests**（CLI サブコマンド・deploy 解決の実バイナリ検証）:
   ```bash
   ./scripts/process/integration_test.sh --categories "tt" --specify "PromptTemplate|PromptInstruction|PromptBundle|PromptTags|PromptPaths"
   ```
   *   **Log Verification**: `TestPromptTemplateRefs` / `TestPromptInstruction` 相当が PASS。deploy 成果物に `{{policy:` が残らないアサーションが通ること。

3. **E2E Tests**:
   *   本変更は IDE GUI を伴わない CLI/emitter ロジックのため、VS Code E2E は不要。
   *   CLI 利用者向け検証は `tests/tt/` 統合テストで代替する（planning-rules §2.1「CLI サブコマンド」）。

### テスト項目設計（§11）

#### ボトムアップ順序

1. **C**: `ResolveTemplateVars` / `BuildTemplateContext` / `TemplateRefManual`（単体）
2. **B**: 各 emitter Emit 後のファイル内容（単体）
3. **A**: `tt prompt deploy` / `tt prompt instruction` / help（統合・実バイナリ）

#### 観点チェックリスト

| # | 観点 | 本計画での扱い |
|---|------|----------------|
| 1 | 正常系 | Cursor/AG/Codex/Claude の解決パス |
| 2 | 異常系・境界 | 未知 kind は未置換で残る |
| 3 | 外部連携 | 統合テストで実 FS へ deploy |
| 4 | データ一貫性 | 解決後パスが実ファイル命名と一致 |
| 5 | 状態遷移 | N/A（ステートレス変換） |
| 6 | 設定反映 | target `paths` 上書きを TemplateContext に反映 |
| 7 | 副作用 | instruction が書き込みしない（stdout のみ） |

#### §11.4 セルフレビュー結果

*   網羅性: R1–R7 を単体＋統合でカバー。O1/O3 は明示先送り。
*   証拠の十分性: 「エラーなし」ではなく解決後文字列と help キーワードをアサート。
*   迂回排除: emitter 単体と `tt` バイナリ統合の二層。
*   依存関係: template → emitter → CLI/統合の順。

### 総合判定プロセス（§12）

Verification 完了後、計画書末尾またはコミットメッセージ／通知に §12.3 フォーマットで総合判定を記載する。

## Documentation

#### [MODIFY] [prompt-manifest-update.md](catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/capabilities/prompt-manifest-update.md)

*   **更新内容**: エンティティ参照節を R5/R7 に合わせて更新し、`tt prompt instruction` を案内する。

#### [MODIFY] [024-Target-Independent-Entity-Refs.md](prompts/phases/000-foundation/branches/main/ideas/024-Target-Independent-Entity-Refs.md)

*   **更新内容**: 実装完了後に必要なら status 追記は任意（本計画では必須としない）。
