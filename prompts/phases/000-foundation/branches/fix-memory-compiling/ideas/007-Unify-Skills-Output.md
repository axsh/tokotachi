---
apiVersion: agent.meta/v1
id: unify-skills-output
kind: idea
title: Unify Skills Output - Workflows廃止とSkills統合
status: draft
---

# Unify Skills Output - Antigravity Workflows 廃止と Skills 統合

## 背景 (Background)

現在、Antigravity emitter は procedure を `.agents/workflows/` と `.agents/skills/` の両方に出力している。
一方、Codex emitter と Claude Code emitter は procedure を `.agents/skills/` にのみ出力する。
この不整合により、Antigravity IDE でワークフロー候補が重複表示される問題が発生している。

### 現状の出力マッピング

| ソース kind | Codex | Claude Code | Antigravity |
|:---|:---|:---|:---|
| capability | `.agents/skills/{id}/SKILL.md` | `.claude/skills/{id}/SKILL.md` | `.agents/skills/{id}/SKILL.md` |
| procedure | `.agents/skills/{id}/SKILL.md` | `.claude/skills/{id}/SKILL.md` | `.agents/workflows/{id}.md` + `.agents/skills/{id}/SKILL.md` |

### 問題点

1. Antigravity IDE で procedure が skills と workflows の両方に表示され、候補が重複する
2. 3ツール間で出力構造が統一されていない
3. `capabilities` フラグ (`workflows: true/false`) がコードで参照されていない
4. `capabilities` フラグが出力ディレクトリベース (rules/skills/workflows) になっており、ソースエンティティの種別 (policy/capability/procedure) と不整合

## 要件 (Requirements)

### 必須要件

1. **Antigravity emitter が procedure を `.agents/workflows/` に出力する処理を廃止する**
   - procedure は capability と同じく `.agents/skills/{id}/SKILL.md` に出力する
   - Codex emitter の procedure-as-skill 出力と同じ `SkillFrontmatter` を使用する

2. **`disable-model-invocation` フラグを適切に設定する**
   - capability: `disable-model-invocation: false` (自動起動可能)
   - procedure: `disable-model-invocation: false` (デフォルト、`trigger.manual_only` がなければ)
   - ソースの `trigger.manual_only: true` が指定されている場合は `disable-model-invocation: true`

3. **`resolvePaths` から workflows パスを除去する**
   - 戻り値を `(rulesDir, skillsDir)` の2値に変更
   - `workflowsDir` 関連の変数、ロジックを全て削除

4. **`TargetPaths` 構造体から `Workflows` フィールドを除去する**
   - template.go の `TargetPaths.Workflows` を削除
   - `{{procedure:id}}` テンプレート変数の解決先を `skills/{id}/SKILL.md` に変更

5. **`ProcedureFrontmatter` 構造体を廃止する**
   - procedure も `SkillFrontmatter` を使用するため不要になる

6. **`EmitResult.TargetDirs` から workflows ディレクトリを除去する**
   - orphan cleanup 対象から workflows を除外

7. **`Check()` メソッドから workflows 関連のロジックを除去する**
   - `liveDirs` マップから `"workflows"` エントリを削除

8. **`CategoryLimit.Workflows` の扱い**
   - limits.go の `Workflows` フィールドは残してよい (後方互換)
   - ただし Antigravity emitter では procedure に skills の limit を適用する

9. **target YAML の `capabilities` を `includes` に改名し、エンティティ種別ベースに変更する**
   - `capabilities` キーを `includes` に改名
   - フラグをソースエンティティ種別に変更: `policy`, `capability`, `procedure`
   - `subagents` は `subagent` に改名 (エンティティ名に統一)
   - compiler または emitter がこのフラグを読み取り、`false` のエンティティをスキップする
   - 全 target YAML を更新:
     ```yaml
     # antigravity.yaml (変更後)
     includes:
       policy: true
       capability: true
       procedure: true
       subagent: false

     # codex.yaml (変更後)
     includes:
       policy: true
       capability: true
       procedure: true
       subagent: false

     # claude-code.yaml (変更後)
     includes:
       policy: true
       capability: true
       procedure: true
       subagent: false

     # cursor.yaml (変更後)
     includes:
       policy: true
       capability: true
       procedure: true
       subagent: false
     ```
   - `paths.workflows` を全 target から削除 (該当があれば)

10. **emitter が `includes` フラグを参照してエンティティをフィルタリングする**
    - `includes.policy: false` なら policy エンティティをスキップ
    - `includes.capability: false` なら capability エンティティをスキップ
    - `includes.procedure: false` なら procedure エンティティをスキップ
    - フラグが未定義の場合はデフォルト `true` (後方互換)

### 任意要件

- capabilities と procedures のソースマニフェスト上の分離は維持する
  (役割の整理として有用であり、`disable-model-invocation` の違いを明確にする)

## 実現方針 (Implementation Approach)

### 変更対象ファイル

#### Go ソースコード

1. **`features/tt/internal/prompt/emitter/antigravity.go`**
   - `resolvePaths()`: 戻り値を `(string, string)` に変更、`workflowsPath` 削除
   - `resolveTargetPaths()`: `TargetPaths.Workflows` 削除
   - `Emit()`: procedure 出力ロジックを skills 方式 (SKILL.md + SkillFrontmatter) に変更
   - `Check()`: `liveDirs` から `"workflows"` エントリ削除
   - `ProcedureFrontmatter` 構造体を削除

2. **`features/tt/internal/prompt/emitter/template.go`**
   - `TargetPaths.Workflows` フィールド削除
   - `resolveRef()` の `"procedure"` ケースを `skills/{id}/SKILL.md` に変更

3. **`features/tt/internal/prompt/emitter/template_test.go`**
   - `TargetPaths` 初期化から `Workflows` 削除
   - procedure 解決テストケースを `skills/` パスに更新

4. **`features/tt/internal/prompt/emitter/emitter_test.go`**
   - target config の `"workflows"` エントリ削除
   - procedure 出力先のアサーションを `skills/` パスに更新

#### 設定ファイル

5. **`prompts/manifest/targets/antigravity.yaml`**
   - `capabilities` -> `includes` に改名、エンティティ種別ベースに変更
   - `paths.workflows` 行を削除

6. **`prompts/manifest/targets/codex.yaml`**
   - `capabilities` -> `includes` に改名、エンティティ種別ベースに変更

7. **`prompts/manifest/targets/claude-code.yaml`**
   - `capabilities` -> `includes` に改名、エンティティ種別ベースに変更

8. **`prompts/manifest/targets/cursor.yaml`**
   - `capabilities` -> `includes` に改名、エンティティ種別ベースに変更

#### emitter フィルタリング

9. **`features/tt/internal/prompt/emitter/antigravity.go`** (追加変更)
   - `Emit()` 冒頭で `includes` フラグを読み取り、各エンティティループの前にスキップ判定を追加

10. **`features/tt/internal/prompt/emitter/codex.go`** (追加変更)
    - 同様に `includes` フラグによるフィルタリングを追加

### 変更しないもの

- Claude Code emitter -- Codex emitter を共有しているため、codex.go の変更で対応
- Cursor emitter -- Claude Code と同じ構造のため同様
- capability / procedure のソースマニフェスト構造 -- 維持
- `limits.go` の `Workflows` フィールド -- 後方互換のため残す

## 検証シナリオ (Verification Scenarios)

1. `tt prompt compile --apply` を実行する
2. `.agents/workflows/` ディレクトリが**生成されない**ことを確認する
3. `.agents/skills/` に全ての procedure が SKILL.md として存在することを確認する
4. SKILL.md の frontmatter に `name`, `description`, `disable-model-invocation` が含まれることを確認する
5. `{{procedure:create-specification}}` テンプレート変数が `.agents/skills/create-specification/SKILL.md` に解決されることを確認する
6. Antigravity IDE でワークフロー候補の重複が解消されることを確認する
7. target YAML で `includes.procedure: false` を設定した場合、procedure が出力されないことを確認する

## テスト項目 (Testing for the Requirements)

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh --backend-only
   ```

2. 統合テスト (template カテゴリ):
   ```
   scripts/process/integration_test.sh --categories "template"
   ```

### 単体テスト (変更対象パッケージ)

```
go test ./features/tt/internal/prompt/emitter/... -v -run "TestAntigravity|TestEmit|TestResolveTemplateVars"
```

### 手動検証 (IDE側)

- Antigravity IDE でスラッシュコマンド候補を表示し、重複がないことを確認 (自動化不可)
