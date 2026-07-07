# 023-Fix-Workflow-Frontmatter-In-PromptUpdate

> **Source Specification**: prompts/phases/000-foundation/branches/main/ideas/022-Fix-Workflow-Frontmatter-In-PromptUpdate.md

## Goal Description

`tt prompt update --target ag` で `.agent/workflows/` に出力される procedure (Workflow) Markdown ファイルに、`name` と `description` を含む YAML frontmatter を付与する。
また、`target.schema.json` の `paths` プロパティに `workflows` が含まれていないためバリデーションエラーが発生する問題も併せて修正する。

## User Review Required

> [!IMPORTANT]
> **target.schema.json の修正について**: 仕様書 022 には明記されていませんが、`tt prompt update --target ag` を実行すると `target.schema.json` の `paths` に `workflows` が定義されていないことによるバリデーションエラーが発生します (仕様書 020 に記載済みの問題)。ローカルの `target.schema.json` がカタログ版より古い状態であるため、本計画ではカタログ版に同期する修正を含めています。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: Antigravity emitter で workflow frontmatter を生成 | Proposed Changes > Emitter > antigravity.go |
| R2: カタログ側 procedure に description を追加/整理 | Proposed Changes > Catalog Procedures > 各 .md ファイル |
| R3: procedure.schema.json に description を追加 | Proposed Changes > Schema > procedure.schema.json |
| (補足) target.schema.json に workflows を追加 | Proposed Changes > Schema > target.schema.json |

## Proposed Changes

### Schema

#### [MODIFY] [target.schema.json](file:///prompts/manifest/schemas/target.schema.json)
*   **Description**: `paths` オブジェクトに `workflows` プロパティを追加する。カタログ版 (`catalog/originals/axsh/go-standard-project/base/prompts/manifest/schemas/target.schema.json`) にはすでに存在しているが、ローカル版に反映されていなかった。
*   **Technical Design**:
    *   `paths.properties` に `"workflows": { "type": "string" }` を追加する。
*   **Logic**:
    *   `antigravity.yaml` は `paths.workflows: .agent/workflows/` を定義しているが、現在のローカルスキーマにはこのプロパティがないため `additionalProperties: false` によりバリデーションエラーが発生する。

変更後の `paths` セクション:
```json
"paths": {
  "type": "object",
  "properties": {
    "rules": { "type": "string" },
    "skills": { "type": "string" },
    "workflows": { "type": "string" }
  },
  "additionalProperties": false
}
```

---

#### [MODIFY] [procedure.schema.json](file:///prompts/manifest/schemas/procedure.schema.json)
*   **Description**: `description` プロパティを追加する。
*   **Technical Design**:
    *   `properties` に `"description"` を追加する。`additionalProperties: false` が設定されているため、追加しないとバリデーションエラーになる。
*   **Logic**:
    *   `approval` プロパティの直後に以下を追加:
    ```json
    "description": {
      "type": "string",
      "description": "Human-readable description of the procedure for display in agent UI."
    }
    ```

---

### Emitter

#### [MODIFY] [emitter_test.go](file:///features/tt/internal/prompt/emitter/emitter_test.go)
*   **Description**: 既存の `TestEmit_Antigravity` テストを更新し、workflow 出力に frontmatter が含まれることを検証する。また、`description` 付きの procedure テストデータを追加する。
*   **Technical Design**:
    *   `test-proc-body` の `Raw` に `"description": "A procedure for testing body"` を追加する。
    *   `test-proc-steps` の `Raw` に `"description": "A procedure for testing steps"` を追加する。
    *   `expectedDryRunFiles` の workflow テストケースの `contain` に frontmatter 文字列を追加する:
        *   `test-proc-body.md`: `"name: test-proc-body"`, `"description: A procedure for testing body"`
        *   `test-proc-steps.md`: `"name: test-proc-steps"`, `"description: A procedure for testing steps"`
*   **Logic**:
    *   テストデータ (L86-L109):
    ```go
    "procedure": {
        {
            APIVersion: "agent.meta/v1",
            Kind:       "procedure",
            ID:         "test-proc-body",
            Title:      "Procedure with body",
            FilePath:   filepath.Join(tempDir, "proc1.yaml"),
            Raw: map[string]any{
                "body":        "This is body content.\n",
                "description": "A procedure for testing body",
            },
        },
        {
            APIVersion: "agent.meta/v1",
            Kind:       "procedure",
            ID:         "test-proc-steps",
            Title:      "Procedure with steps",
            FilePath:   filepath.Join(tempDir, "proc2.yaml"),
            Raw: map[string]any{
                "steps": []any{
                    "step-one",
                    "step-two",
                },
                "description": "A procedure for testing steps",
            },
        },
    },
    ```
    *   期待値 (L150-L157):
    ```go
    {
        path:    filepath.Join(buildDir, "antigravity", "workflows_dir", "test-proc-body.md"),
        contain: []string{"name: test-proc-body", "description: A procedure for testing body", "This is body content."},
    },
    {
        path:    filepath.Join(buildDir, "antigravity", "workflows_dir", "test-proc-steps.md"),
        contain: []string{"name: test-proc-steps", "description: A procedure for testing steps", "1. step-one", "2. step-two"},
    },
    ```

---

#### [MODIFY] [antigravity.go](file:///features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: `Emit Procedures` セクション (L196-L237) を修正し、capability emit と同様に frontmatter を生成する。
*   **Technical Design**:
    *   `proc.Raw["description"]` から description を取得する。
    *   `SkillFrontmatter` 構造体を再利用し、`Name` と `Description` のみを設定する (`Paths` と `DisableModelInvocation` は空/false のままで `omitempty` により省略される)。
    *   `content := body` の代わりに `content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)` を使用する。
*   **Logic**:
    *   L214-L216 を以下のように変更:
    ```go
    body = ResolveTemplateVars(body, tmplCtx)

    desc, _ := proc.Raw["description"].(string)
    fm := SkillFrontmatter{
        Name:        proc.ID,
        Description: desc,
    }
    fmBytes, err := yaml.Marshal(fm)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal workflow frontmatter for %s: %w", proc.ID, err)
    }
    content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)
    ```

---

### Catalog Procedures

カタログ内の各 procedure Markdown ファイルの frontmatter に `description` フィールドを追加する。
一部のファイル (`create-specification.md`, `investigate.md`, `systematize-far-knowledge.md`, `create-pull-request.md`) には 2つ目の frontmatter ブロック内に `description` が存在するが、パーサー (`parseMarkdownWithFrontmatter`) は最初の frontmatter しか解析しないため `Raw` に反映されない。これらは最初の frontmatter に移動する。

対象ディレクトリ: `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/`

#### [MODIFY] [build-pipeline.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/build-pipeline.md)
*   **変更内容**: frontmatter に `description: Run the full build, unit test, and integration test pipeline to verify code changes.` を追加。

#### [MODIFY] [create-implementation-plan.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/create-implementation-plan.md)
*   **変更内容**: frontmatter に `description: Create a detailed implementation plan document with proposed changes, verification steps, and user review.` を追加。

#### [MODIFY] [create-specification.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/create-specification.md)
*   **変更内容**: 2つ目の frontmatter ブロック (`---\ndescription: Create Specification\n---`) を削除し、最初の frontmatter に `description: Create a structured specification document capturing background, requirements, and verification scenarios.` を追加。

#### [MODIFY] [create-pull-request.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/create-pull-request.md)
*   **変更内容**: 2つ目の frontmatter ブロックを削除し、最初の frontmatter に `description: Create a pull request from committed code changes and manage post-merge revisions.` を追加。

#### [MODIFY] [execute-implementation-plan.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/execute-implementation-plan.md)
*   **変更内容**: frontmatter に `description: Execute an approved implementation plan following TDD workflow with incremental commits and verification.` を追加。

#### [MODIFY] [investigate.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/investigate.md)
*   **変更内容**: 2つ目の frontmatter ブロックを削除し、最初の frontmatter に `description: Investigate a codebase, bug, or technical question and produce a structured investigation report.` を追加。

#### [MODIFY] [review-point.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/review-point.md)
*   **変更内容**: frontmatter に `description: Pause the current workflow to request user review and feedback before proceeding.` を追加。

#### [MODIFY] [run-all-tests.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/run-all-tests.md)
*   **変更内容**: frontmatter に `description: Run all project tests including build, unit, integration, and E2E tests with a repair loop for failures.` を追加。

#### [MODIFY] [systematize-far-knowledge.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/systematize-far-knowledge.md)
*   **変更内容**: 2つ目の frontmatter ブロックを削除し、最初の frontmatter に `description: Process pending intake events and compile far-knowledge into agent-readable skills.` を追加。

#### [MODIFY] [test-generator.md](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/test-generator.md)
*   **変更内容**: frontmatter に `description: Generate test implementation plans based on specifications and existing test infrastructure.` を追加。

---

### Local Procedures (prompts/manifest)

ローカルの procedure ファイルにも同様の修正を適用する。

対象ディレクトリ: `prompts/manifest/code_content/procedures/`

同じ10ファイルについて、カタログ側と同一の修正を行う。

## Step-by-Step Implementation Guide

1.  **テストの更新 (TDD: Red)**:
    *   `features/tt/internal/prompt/emitter/emitter_test.go` の `TestEmit_Antigravity` テストデータに `description` フィールドを追加する。
    *   期待値に frontmatter 文字列チェックを追加する。
    *   `./scripts/process/build.sh` を実行し、テストが **失敗する** ことを確認する。

2.  **Emitter の修正 (TDD: Green)**:
    *   `features/tt/internal/prompt/emitter/antigravity.go` の `Emit Procedures` セクション (L214-L216) を修正し、frontmatter 生成ロジックを追加する。
    *   `./scripts/process/build.sh` を実行し、テストが **成功する** ことを確認する。

3.  **Schema の修正**:
    *   `prompts/manifest/schemas/target.schema.json` の `paths.properties` に `"workflows": { "type": "string" }` を追加する。
    *   `prompts/manifest/schemas/procedure.schema.json` の `properties` に `"description"` を追加する。
    *   カタログ側のスキーマ (`catalog/.../schemas/procedure.schema.json`) にも同一の修正を適用する。

4.  **カタログ Procedure の修正**:
    *   `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/` 配下の全10ファイルに `description` を追加する。
    *   2つ目の frontmatter ブロックがあるファイル (`create-specification.md`, `investigate.md`, `systematize-far-knowledge.md`, `create-pull-request.md`) は、2つ目のブロックを削除し、最初の frontmatter に `description` を移動する。

5.  **ローカル Procedure の修正**:
    *   `prompts/manifest/code_content/procedures/` 配下の全ファイルに、カタログと同一の修正を適用する。

6.  **Verification Plan の実行**:
    *   Step 6-1 から Step 6-3 を順に実施する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: `TestEmit_Antigravity` が PASS すること。workflow 出力に `name:` と `description:` を含む frontmatter が検証される。

2.  **Integration Tests (common)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```
    *   **Log Verification**: スキーマバリデーションを含む既存の common テストが PASS すること。

3.  **手動検証 (E2E 代替)**:
    ```bash
    tt prompt update --target ag --force
    ```
    *   `.agents/workflows/` 配下のファイルに `---\nname: ...\ndescription: ...\n---` 形式の frontmatter が存在することを確認する。
    *   **E2E テストが不要な理由**: 本変更は `tt` CLI ツール内部の emitter ロジック修正であり、既存の単体テスト (`TestEmit_Antigravity`) で frontmatter 生成ロジックを網羅的に検証している。`tt` CLI 自体の E2E テストインフラは本リポジトリの `tests/` 配下には存在しないため、手動検証で補完する。

### セルフレビュー結果 (SS11.4)

*   **網羅性**: 仕様書の3要件 (R1-R3) すべてが Requirement Traceability で対応付けられている。追加で発見した target.schema.json の問題も修正対象に含めた。
*   **証拠の十分性**: 単体テスト (`TestEmit_Antigravity`) で frontmatter 生成を検証。common カテゴリの統合テストでスキーマ整合性を検証。
*   **迂回排除**: 手動検証は E2E テストインフラがない部分のみ。ロジックは単体テストでカバー。
*   **依存関係**: スキーマ修正 -> カタログ修正 -> emitter 修正の順序で依存関係あり。ただし TDD の観点からテストを先に書く。

## Documentation

本変更はツール内部の emitter ロジック修正であり、ユーザー向けドキュメントの更新は不要です。仕様書 022 がそのまま変更の記録として機能します。
