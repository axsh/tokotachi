# 000-Restructure-Agent-Target-Paths

> **Source Specification**: [030-Restructure-Agent-Target-Paths.md](file:///prompts/phases/000-foundation/branches/fix-workflows-folder-for-agy/ideas/030-Restructure-Agent-Target-Paths.md)

## Goal Description

Antigravity ターゲットにおいて廃止されていた `workflows` ディレクトリを復元し、 `procedures` をそこに展開するように戻します。また、AG と Codex のターゲットディレクトリが `.agents/` で競合している問題を解決するため、デフォルトパスをそれぞれ `.agent/` および `.codex/` に変更します。これに伴い、テンプレート変数の解決ロジックやターゲット定義ファイルも更新します。

## User Review Required

None.

## Requirement Traceability

> **Traceability Check**:
> 仕様書(Specification)の要件・決定事項をリストアップし、この計画書のどこで対応するかをマッピングしてください。

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| AGのデフォルトベースディレクトリを `.agent/` に変更 | Proposed Changes > `antigravity.go` |
| AGの `workflows` フォルダ出力を復元し、 `procedures` を展開 | Proposed Changes > `antigravity.go` |
| Codexのデフォルトベースディレクトリを `.codex/` に変更 | Proposed Changes > `codex.go` |
| `{{procedure:id}}` テンプレート変数の解決パス修正 | Proposed Changes > `template.go` |
| ターゲットマニフェスト (`antigravity.yaml`, `codex.yaml`) の更新 | Proposed Changes > `antigravity.yaml`, `codex.yaml` |

## Proposed Changes

### tt prompt compiler / emitter

#### [MODIFY] [template.go](file:///features/tt/internal/prompt/emitter/template.go)
*   **Description**: `TargetPaths` 構造体に `Workflows` を追加し、テンプレート変数解決ロジックを更新します。
*   **Technical Design**:
    ```go
    type TargetPaths struct {
        Rules     string
        Skills    string
        Workflows string // 追加
    }

    func resolveRef(kind, id string, ctx *TemplateContext) string {
        switch kind {
        case "policy":
            return resolvePolicyPath(id, ctx)
        case "procedure":
            // Workflows パスが設定されている場合はそれを使用
            if ctx.Paths.Workflows != "" {
                return ensureTrailingSlash(ctx.Paths.Workflows) + id + ".md"
            }
            return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
        case "capability":
            return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
        // ...
    }
    ```

#### [MODIFY] [antigravity.go](file:///features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: デフォルトパスの変更と `workflows` 出力ロジックの復元を行います。
*   **Technical Design**:
    *   `resolvePaths`: デフォルト値を `.agent/rules/`, `.agent/skills/` に変更。 `workflowsPath` (`.agent/workflows/`) を追加で返すように修正。
    *   `Emit`: `inc.Procedure` ブロック内で、 `procedures` を `workflowsDir` に `{id}.md` として書き出す。
    *   `resolveTargetPaths`: `Workflows` フィールドを解決するように修正。
    *   `Check`: `workflows` ディレクトリをドリフトチェックの対象に含める。

#### [MODIFY] [codex.go](file:///features/tt/internal/prompt/emitter/codex.go)
*   **Description**: デフォルトパスの変更とインデックスファイル内の説明文を更新します。
*   **Technical Design**:
    *   `resolvePaths`: デフォルト値を `.codex/rules/`, `.codex/skills/` に変更。
    *   `generateMarkerContent`: テキスト内の `.agents/` を `.codex/` に、 `rulesPath` / `skillsPath` のデフォルト値を修正。

### ターゲットマニフェスト

#### [MODIFY] [antigravity.yaml](file:///prompts/manifest/targets/antigravity.yaml)
*   **Description**: `paths` を新しいデフォルトに合わせます。
*   **Technical Design**:
    ```yaml
    paths:
      rules: .agent/rules/
      skills: .agent/skills/
      workflows: .agent/workflows/ # 追加
    ```

#### [MODIFY] [codex.yaml](file:///prompts/manifest/targets/codex.yaml)
*   **Description**: `paths` および `index_file` を新しいデフォルトに合わせます。
*   **Technical Design**:
    ```yaml
    paths:
      rules: .codex/rules/
      skills: .codex/skills/
    index_file: CODEX.md # AGENTS.md から変更
    ```

### テストコードの更新 (TDD)

#### [MODIFY] [template_test.go](file:///features/tt/internal/prompt/emitter/template_test.go)
*   **Description**: `Workflows` パスがある場合の解決テストを追加します。

#### [MODIFY] [emitter_test.go](file:///features/tt/internal/prompt/emitter/emitter_test.go)
*   **Description**: `TestEmit_Antigravity` を新しいパス構造に合わせて更新し、 `workflows` 出力の検証を追加します。

#### [MODIFY] [codex_test.go](file:///features/tt/internal/prompt/emitter/codex_test.go)
*   **Description**: 新しいパス（ `.codex/` ）に合わせて期待値を更新します。

## Step-by-Step Implementation Guide

1.  **Modify `template.go`**:
    *   Add `Workflows` to `TargetPaths` struct.
    *   Update `resolveRef` to handle `procedure` with `Workflows` path.
2.  **Update `template_test.go`**:
    *   Update existing tests or add new ones to verify `Workflows` path resolution.
3.  **Modify `antigravity.go`**:
    *   Update `resolvePaths` defaults and return values.
    *   Modify `Emit` to output procedures to `workflows` folder.
    *   Update `resolveTargetPaths` to populate `Workflows`.
    *   Update `Check` to include workflows directory.
4.  **Modify `codex.go`**:
    *   Update `resolvePaths` defaults.
    *   Update `generateMarkerContent` string literals and defaults.
5.  **Update `emitter_test.go` and `codex_test.go`**:
    *   Update expected paths and contents in emitter tests.
6.  **Update target manifests**:
    *   Modify `antigravity.yaml` and `codex.yaml`.
7.  **Run Verification**:
    *   Execute build and integration tests.

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "template"
    ```
    *   **Log Verification**: `tt prompt update` の実行結果として、新しいパス（ `.agent/` , `.codex/` ）が正しく出力されていること、および `workflows` が生成されていることを確認します。

### 総合判定プロセスの計画

全テスト完了後、以下の項目を確認します。
1.  `.agents/` ディレクトリが新しく作成されていないこと。
2.  `{{procedure:id}}` が AG ターゲットにおいて `.agent/workflows/id.md` に解決されていること。
3.  `CODEX.md` (旧 AGENTS.md) 内のパスが `.codex/` になっていること。
