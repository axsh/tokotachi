---
apiVersion: agent.meta/v1
id: Add-Description-To-Procedures-And-Workflows
kind: idea
---

# カタログ Procedure に description を追加し、Antigravity Workflow に frontmatter として転記する

## 背景 (Background)

Antigravity (ag) ターゲットでは、procedure エンティティが `.agent/workflows/` ディレクトリに Markdown ファイルとして出力される。
しかし、現在の出力には YAML frontmatter が付与されておらず、`description` フィールドも含まれていない。

Antigravity IDE は workflow ファイルに `description` を含む frontmatter を期待しており、
これが不在の場合、ワークフロー一覧での説明表示やコマンドパレットでの補完が正しく機能しない。

一方、他のターゲット (cursor, claude-code, codex) は既にスキル/プロシージャの emit 時に
`description` を frontmatter に含めて出力している。

### 現状の問題点

1. **カタログ側**: 9個の procedure Markdown に `description` フィールドが存在しない
2. **スキーマ側**: `procedure.schema.json` に `description` プロパティが定義されていない
3. **エミッター側**: `antigravity.go` の procedure emit で frontmatter を生成していない (`content := body` のみ)

## 要件 (Requirements)

### 必須要件

1. **カタログ procedure に description を追加**: `catalog/originals/axsh/go-standard-project/base/prompts/manifest/code_content/procedures/` 配下の全9ファイルの frontmatter に `description` フィールドを英語で追加する
2. **procedure.schema.json に description を追加**: `description` プロパティを optional として追加する
3. **Antigravity emitter で workflow frontmatter を生成**: `antigravity.go` の procedure emit 部分で、`description` を含む YAML frontmatter を付与して出力する
4. **出力形式**: 以下の形式で frontmatter を生成する
   ```yaml
   ---
   name: <procedure-id>
   description: <description from frontmatter>
   ---
   ```

### 対象ファイルと description 値

| ファイル | id | description (案) |
|---|---|---|
| build-pipeline.md | build-pipeline | Run the full build, unit test, and integration test pipeline to verify code changes. |
| create-implementation-plan.md | create-implementation-plan | Create a detailed implementation plan document with proposed changes, verification steps, and user review. |
| create-specification.md | create-specification | Create a structured specification document capturing background, requirements, and verification scenarios. |
| execute-implementation-plan.md | execute-implementation-plan | Execute an approved implementation plan following TDD workflow with incremental commits and verification. |
| investigate.md | investigate | Investigate a codebase, bug, or technical question and produce a structured investigation report. |
| review-point.md | review-point | Pause the current workflow to request user review and feedback before proceeding. |
| run-all-tests.md | run-all-tests | Run all project tests (build, unit, integration, E2E) and fix any failures in a repair loop. |
| systematize-far-knowledge.md | systematize-far-knowledge | Process pending intake events and compile far-knowledge into agent-readable skills. |
| test-generator.md | test-generator | Generate test implementation plans based on specifications and existing test infrastructure. |

## 実現方針 (Implementation Approach)

### 1. カタログ procedure frontmatter 修正

各 procedure の YAML frontmatter に `description:` フィールドを追加する。
既存の frontmatter 構造は維持し、`trigger` の後に `description` を追加する。

```yaml
---
apiVersion: agent.meta/v1
id: build-pipeline
kind: procedure
title: Build, Test, and Verify Pipeline
trigger:
    command: build-pipeline
description: Run the full build, unit test, and integration test pipeline to verify code changes.
---
```

### 2. procedure.schema.json 修正

`properties` に `description` を追加:

```json
"description": {
  "type": "string",
  "description": "Brief description of the procedure for display in UI and command palettes."
}
```

### 3. Antigravity emitter 修正

`antigravity.go` の procedure emit 部分 (L196-L237) を修正:

- `proc.Raw["description"]` から description を取得
- `SkillFrontmatter` と同様の struct (または同じ struct) を使って frontmatter を生成
- `content := body` を `content := frontmatter + body` に変更

```go
// Before (L216)
content := body

// After
desc, _ := proc.Raw["description"].(string)
fm := SkillFrontmatter{
    Name:        proc.ID,
    Description: desc,
}
fmBytes, _ := yaml.Marshal(fm)
content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)
```

## 検証シナリオ (Verification Scenarios)

1. カタログの全9 procedure に `description` が追加されていること
2. `tt prompt update --target ag` を実行し、`.agent/workflows/` 以下のファイルに frontmatter (`name`, `description`) が付与されていること
3. 他のターゲット (cursor, codex, claude-code) の出力に影響がないこと
4. スキーマバリデーションが通ること

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

### 単体テスト

- `TestUpdate_CatalogTemplate_AllTargets`: カタログテンプレートからの全ターゲット出力検証 (既存)
- 新規テスト (antigravity emitter): workflow 出力に frontmatter が含まれることを検証

### 統合テスト

- カタログテンプレートの `TestUpdate_CatalogTemplate_AllTargets` で `.agent/workflows/` ファイルの内容を検証 (frontmatter 存在チェック追加)
