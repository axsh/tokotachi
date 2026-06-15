---
apiVersion: agent.meta/v1
id: unify-skills-output
kind: plan
title: 007-Unify-Skills-Output
status: draft
---

# 007-Unify-Skills-Output

> **Source Specification**: [007-Unify-Skills-Output.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/007-Unify-Skills-Output.md)

## Goal Description

Antigravity emitter の workflows 出力を廃止し、procedure を skills に統合する。
また `capabilities` フラグを `includes` に改名してエンティティ種別ベースに変更し、
全 emitter でフィルタリングを機能させる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Procedure を skills/ に出力 | Proposed Changes > antigravity.go Emit() |
| disable-model-invocation フラグ設定 | Proposed Changes > antigravity.go Emit() |
| resolvePaths から workflows 除去 | Proposed Changes > antigravity.go resolvePaths() |
| TargetPaths.Workflows 除去 | Proposed Changes > template.go |
| ProcedureFrontmatter 廃止 | Proposed Changes > antigravity.go |
| EmitResult.TargetDirs から workflows 除去 | Proposed Changes > antigravity.go Emit() |
| Check() から workflows 除去 | Proposed Changes > antigravity.go Check() |
| CategoryLimit.Workflows 後方互換で残す | 変更なし (limits.go) |
| capabilities -> includes 改名 | Proposed Changes > target YAML x4 + includes.go |
| target schema 更新 | Proposed Changes > target.schema.json |
| immune モードでデプロイ | Step-by-Step > 最終ステップ |

## Proposed Changes

### Emitter 共通: includes ヘルパー

#### [NEW] [includes.go](file:///features/tt/internal/prompt/emitter/includes.go)

*   **Description**: `includes` フラグの読み取りヘルパー関数を新設
*   **Technical Design**:
    ```go
    // EntityIncludes holds entity-type inclusion flags for a target.
    type EntityIncludes struct {
        Policy     bool
        Capability bool
        Procedure  bool
        Subagent   bool
    }

    // ExtractIncludes reads includes flags from a target entity.
    // Returns defaults (all true) if the target is nil or has no includes.
    // Supports both "includes" (new) and "capabilities" (legacy) keys.
    func ExtractIncludes(target *manifest.Entity) EntityIncludes {
        defaults := EntityIncludes{
            Policy: true, Capability: true,
            Procedure: true, Subagent: false,
        }
        if target == nil { return defaults }

        // Try "includes" first, fallback to "capabilities" for backward compat
        raw, ok := target.Raw["includes"].(map[string]any)
        if !ok {
            raw, ok = target.Raw["capabilities"].(map[string]any)
            if !ok { return defaults }
        }

        result := defaults
        if v, ok := raw["policy"].(bool); ok { result.Policy = v }
        if v, ok := raw["capability"].(bool); ok { result.Capability = v }
        // Also check "capabilities" (plural) for legacy
        if v, ok := raw["capabilities"].(bool); ok { result.Capability = v }
        if v, ok := raw["procedure"].(bool); ok { result.Procedure = v }
        // Legacy: map old key names
        if v, ok := raw["rules"].(bool); ok { result.Policy = v }
        if v, ok := raw["skills"].(bool); ok { result.Capability = v }
        if v, ok := raw["workflows"].(bool); ok { result.Procedure = v }
        if v, ok := raw["subagent"].(bool); ok { result.Subagent = v }
        if v, ok := raw["subagents"].(bool); ok { result.Subagent = v }
        return result
    }
    ```
*   **Logic**: `includes` キーを優先し、存在しなければ旧 `capabilities` キーにフォールバック。各フラグが未定義の場合はデフォルト true (subagent のみ false)。旧キー名 (rules/skills/workflows/subagents) もマッピング。

#### [NEW] [includes_test.go](file:///features/tt/internal/prompt/emitter/includes_test.go)

*   **Description**: `ExtractIncludes` の単体テスト
*   **Logic**:
    - nil target -> デフォルト値 (全 true, subagent false)
    - 新 `includes` キーでの読み取り
    - 旧 `capabilities` キーでのフォールバック
    - 一部フラグのみ指定 -> 未指定はデフォルト

---

### Antigravity Emitter

#### [MODIFY] [antigravity.go](file:///features/tt/internal/prompt/emitter/antigravity.go)

*   **Description**: workflows 出力廃止、procedure を skills に統合、includes フィルタリング追加
*   **Technical Design**:

    1. `ProcedureFrontmatter` 構造体 (L29-31) を削除

    2. `resolvePaths()` (L33-65):
       - 戻り値を `(string, string)` に変更
       - `workflowsPath` 変数と `paths["workflows"]` 読み取りを削除
       ```go
       func (a *AntigravityEmitter) resolvePaths(resolved *manifest.ResolvedManifest,
           buildDir string, apply bool) (string, string) {
           rulesPath := ".agents/rules/"
           skillsPath := ".agents/skills/"
           for _, target := range resolved.Entities["target"] {
               if target.ID == "antigravity" {
                   if paths, ok := target.Raw["paths"].(map[string]any); ok {
                       if r, ok := paths["rules"].(string); ok { rulesPath = r }
                       if s, ok := paths["skills"].(string); ok { skillsPath = s }
                   }
               }
           }
           if apply {
               return filepath.Join(a.RootDir, rulesPath),
                   filepath.Join(a.RootDir, skillsPath)
           }
           return filepath.Join(buildDir, "antigravity", rulesPath),
               filepath.Join(buildDir, "antigravity", skillsPath)
       }
       ```

    3. `Emit()` (L67-260):
       - 呼び出し: `rulesDir, skillsDir := a.resolvePaths(...)` (2値)
       - `includes` フィルタリング追加:
         ```go
         inc := ExtractIncludes(FindTarget(resolved, "antigravity"))
         ```
       - Policy ループの前: `if !inc.Policy { ... skip ... }`
       - Capability ループの前: `if !inc.Capability { ... skip ... }`
       - Procedure ループ (L193-241): 完全に書き換え
         - `if !inc.Procedure { ... skip ... }`
         - `ProcedureFrontmatter` -> `SkillFrontmatter` を使用
         - `trigger.manual_only` から `DisableModelInvocation` を設定
         - 出力先: `filepath.Join(skillsDir, proc.ID, "SKILL.md")`
         - limit チェック: `limits.Workflows` -> `limits.Skills`
       - `EmitResult.TargetDirs`: `workflowsDir` を除去 -> `[]string{rulesDir, skillsDir}`

    4. `resolveTargetPaths()` (L263-286):
       - `TargetPaths.Workflows` 設定を削除
       - `paths["workflows"]` 読み取りを削除

    5. `Check()` (L293-440):
       - `resolvePaths` の戻り値を2値に変更
       - `liveDirs` から `"workflows"` エントリを削除

#### [MODIFY] [template.go](file:///features/tt/internal/prompt/emitter/template.go)

*   **Description**: `TargetPaths.Workflows` を削除、`{{procedure:id}}` の解決先を変更
*   **Technical Design**:
    - L26 `Workflows string` フィールドを削除
    - L54-55 の `case "procedure"` を変更:
      ```go
      case "procedure":
          return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
      ```

---

### Codex / Claude Code / Cursor Emitter

#### [MODIFY] [codex.go](file:///features/tt/internal/prompt/emitter/codex.go)

*   **Description**: includes フィルタリング追加
*   **Logic**: `Emit()` 冒頭に以下を追加:
    ```go
    inc := ExtractIncludes(FindTarget(resolved, "codex"))
    ```
    各エンティティループの前に `if !inc.Policy/Capability/Procedure` チェック追加。

#### [MODIFY] [claude_code.go](file:///features/tt/internal/prompt/emitter/claude_code.go)

*   **Description**: 同様に includes フィルタリング追加 (target ID = "claude-code")

#### [MODIFY] [cursor.go](file:///features/tt/internal/prompt/emitter/cursor.go)

*   **Description**: 同様に includes フィルタリング追加 (target ID = "cursor")

---

### Target YAML

#### [MODIFY] [antigravity.yaml](file:///prompts/manifest/targets/antigravity.yaml)

*   **変更内容**:
    ```yaml
    # Before
    capabilities:
      rules: true
      skills: true
      workflows: true
      subagents: false
    paths:
      rules: .agents/rules/
      skills: .agents/skills/
      workflows: .agents/workflows/

    # After
    includes:
      policy: true
      capability: true
      procedure: true
      subagent: false
    paths:
      rules: .agents/rules/
      skills: .agents/skills/
    ```

#### [MODIFY] [codex.yaml](file:///prompts/manifest/targets/codex.yaml)

*   **変更内容**: `capabilities` -> `includes` に改名、エンティティ種別ベースに変更

#### [MODIFY] [claude-code.yaml](file:///prompts/manifest/targets/claude-code.yaml)

*   **変更内容**: 同上

#### [MODIFY] [cursor.yaml](file:///prompts/manifest/targets/cursor.yaml)

*   **変更内容**: 同上

---

### Schema

#### [MODIFY] [target.schema.json](file:///prompts/manifest/schemas/target.schema.json)

*   **Description**: `capabilities` -> `includes` への改名をスキーマに反映
*   **Technical Design**:
    - `required` から `"capabilities"` を削除、`"includes"` は任意 (後方互換のため required にしない)
    - `capabilities` プロパティ定義を削除
    - `includes` プロパティ定義を追加:
      ```json
      "includes": {
        "type": "object",
        "properties": {
          "policy": { "type": "boolean" },
          "capability": { "type": "boolean" },
          "procedure": { "type": "boolean" },
          "subagent": { "type": "boolean" }
        },
        "additionalProperties": false
      }
      ```
    - `paths.workflows` プロパティを削除
    - `limits.workflows` はそのまま残す (後方互換)

### テスト

#### [MODIFY] [template_test.go](file:///features/tt/internal/prompt/emitter/template_test.go)

*   **Description**: `TargetPaths` から `Workflows` 削除、procedure テストケースを skills パスに更新

#### [MODIFY] [emitter_test.go](file:///features/tt/internal/prompt/emitter/emitter_test.go)

*   **Description**: target config の `"workflows"` 削除、procedure 出力先を skills パスに更新

## Step-by-Step Implementation Guide

1.  **includes ヘルパーの作成**:
    - `features/tt/internal/prompt/emitter/includes.go` を新規作成
    - `ExtractIncludes` 関数を実装
    - `features/tt/internal/prompt/emitter/includes_test.go` を新規作成

2.  **template.go の修正**:
    - `TargetPaths.Workflows` フィールドを削除
    - `resolveRef` の `"procedure"` ケースを `skills/{id}/SKILL.md` に変更

3.  **antigravity.go の修正**:
    - `ProcedureFrontmatter` 構造体を削除
    - `resolvePaths()` を 2値返却に変更
    - `resolveTargetPaths()` から Workflows を削除
    - `Emit()` の procedure ループを skills 方式に書き換え
    - `Emit()` に includes フィルタリングを追加
    - `EmitResult.TargetDirs` から workflows を除去
    - `Check()` から workflows を除去

4.  **codex.go / claude_code.go / cursor.go の修正**:
    - 各 `Emit()` に includes フィルタリングを追加

5.  **target YAML の更新**:
    - 4ファイルの `capabilities` -> `includes` 改名

6.  **target.schema.json の更新**:
    - `capabilities` -> `includes` に改名
    - `paths.workflows` を削除

7.  **テストの修正**:
    - `template_test.go`: Workflows 削除、procedure テスト更新
    - `emitter_test.go`: workflows 削除、procedure 出力先更新

8.  **ビルド + テスト実行**:
    ```bash
    scripts/process/build.sh --backend-only
    ```

9.  **immune モードでデプロイ**:
    ```bash
    ./bin/tt.exe prompt deploy --force --mode immune
    ```

10. **デプロイ結果の検証**:
    - `.agents/workflows/` が存在しないことを確認
    - `.agents/skills/` に全 procedure が SKILL.md として存在することを確認
    - `{{procedure:*}}` テンプレート変数が skills パスに解決されることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --backend-only
    ```

2.  **emitter パッケージの単体テスト**:
    ```bash
    go test ./features/tt/internal/prompt/emitter/... -v
    ```

3.  **統合テスト (template カテゴリ)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "template"
    ```
    - 存在しない場合はスキップ。

4.  **immune デプロイ後の検証コマンド**:
    ```bash
    # workflows ディレクトリが存在しないこと
    test ! -d .agents/workflows && echo "OK: no workflows dir"

    # 全 procedure が skills に存在すること
    for proc in build-pipeline create-implementation-plan create-specification \
                execute-implementation-plan investigate review-point \
                run-all-tests systematize-far-knowledge test-generator; do
        test -f ".agents/skills/$proc/SKILL.md" && echo "OK: $proc" || echo "FAIL: $proc"
    done

    # SKILL.md に disable-model-invocation が含まれること
    grep -l "disable-model-invocation" .agents/skills/*/SKILL.md | wc -l
    ```

### Manual Verification

- Antigravity IDE でスラッシュコマンド候補を表示し、重複がないことを確認 (自動化不可)

## Documentation

#### [MODIFY] [MEMORY.md](file:///prompts/MEMORY.md)

*   **更新内容**: メモリシステムの説明は変更なし (本改修の範囲外)
