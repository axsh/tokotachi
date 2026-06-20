# 000-SeparateDeployPaths-AgyAndCodex

> **Source Specification**: [001-SeparateDeployPaths-AgyAndCodex.md](file://prompts/phases/000-foundation/branches/fix-compile-bug/ideas/001-SeparateDeployPaths-AgyAndCodex.md)

## Goal Description

Antigravity (agy) と Codex のデプロイパスを完全分離する。
- Antigravity: `.agents/` (複数形) から `.agent/` (単数形) へ移行。procedures を `workflows/` にフラットファイルとしてデプロイ
- Codex: `.agents/` から `.codex/` へ移行
- `.agents/` (複数形) は廃止

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: デプロイパスの分離 (agy=`.agent/`, codex=`.codex/`) | Proposed Changes > antigravity.yaml, codex.yaml, antigravity.go, codex.go |
| R2: procedures を workflows にデプロイ (フラットファイル `{id}.md`) | Proposed Changes > antigravity.go (Emit procedures section) |
| R3: Codex の AGENTS.md マーカーが `.codex/` を参照 | Proposed Changes > codex.go (generateMarkerContent) |
| R4: ターゲット設定ファイルの更新 | Proposed Changes > antigravity.yaml, codex.yaml |
| R5: buildDir 出力の整合性 | Proposed Changes > antigravity.go, codex.go (resolvePaths) |
| R6: Cursor / Claude Code は変更なし | 対象外 (変更不要) |
| R7: workflows frontmatter | Proposed Changes > antigravity.go (WorkflowFrontmatter struct) |

## Proposed Changes

### emitter パッケージ テスト (`features/tt/internal/prompt/emitter/`)

#### [MODIFY] [template_test.go](file://features/tt/internal/prompt/emitter/template_test.go)
*   **Description**: テンプレート変数解決テストを新しいパスと Workflows フィールドに合わせて更新
*   **Technical Design**:
    - `TargetPaths` の初期化に `Workflows` フィールドを追加
    - テストケースのパス期待値を `.agents/` から `.agent/` に更新
    - `procedure` の解決先を `workflows/` パスに変更
*   **Logic**:
    - `TargetPaths{Rules: ".agent/rules/", Skills: ".agent/skills/", Workflows: ".agent/workflows/"}` に変更
    - `"procedure reference resolves to workflows path"` テストケース追加: `{{procedure:build-pipeline}}` -> `.agent/workflows/build-pipeline.md`
    - `"capability reference resolves to skills path"` テストケースのパスを `.agent/skills/` に更新

テストケース変更:
```go
// 既存 procedure テスト → 更新
{
    name:  "procedure reference resolves to workflows path",
    input: "Run {{procedure:arch-correct}} when needed.",
    want:  "Run .agent/workflows/arch-correct.md when needed.",
},
// 既存 capability テスト → パス更新のみ
{
    name:  "capability reference resolves to skills path",
    input: "Use {{capability:architecture-maintainer}} skill.",
    want:  "Use .agent/skills/architecture-maintainer/SKILL.md skill.",
},
```

#### [MODIFY] [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go)
*   **Description**: Antigravity Emit テストを更新し、procedures が `workflows/` にフラットファイルで出力されることを検証
*   **Technical Design**:
    - テスト内のターゲット YAML パスを `.agent/` に更新
    - procedure の出力先として `workflows/{id}.md` を期待するアサーションを追加
    - capability の出力先として `skills/{id}/SKILL.md` を期待するアサーションを維持
*   **Logic**:
    - ターゲット YAML のパス設定: `rules: .agent/rules/`, `skills: .agent/skills/`, `workflows: .agent/workflows/`
    - procedure エンティティの出力検証: `filepath.Join(buildDir, "antigravity", ".agent/workflows/", proc.ID+".md")` が存在
    - capability エンティティの出力検証: `filepath.Join(buildDir, "antigravity", ".agent/skills/", cap.ID, "SKILL.md")` が存在

#### [MODIFY] [codex_test.go](file://features/tt/internal/prompt/emitter/codex_test.go)
*   **Description**: Codex Emit テストのパスを `.codex/` に更新し、AGENTS.md マーカーの参照パスを検証
*   **Technical Design**:
    - テスト内のターゲット YAML パスを `.codex/rules/`, `.codex/skills/` に更新
    - AGENTS.md マーカーテストで `.codex/` パスへの参照を検証
*   **Logic**:
    - ターゲット YAML: `"rules": ".codex/rules/"`, `"skills": ".codex/skills/"`
    - AGENTS.md マーカーに `.codex/rules/` と `.codex/skills/` が含まれることを検証

---

### emitter パッケージ 実装 (`features/tt/internal/prompt/emitter/`)

#### [MODIFY] [template.go](file://features/tt/internal/prompt/emitter/template.go)
*   **Description**: `TargetPaths` に `Workflows` フィールドを追加し、`procedure` の解決先を変更
*   **Technical Design**:
    ```go
    // TargetPaths holds the target-specific output paths.
    type TargetPaths struct {
        Rules     string // e.g., ".agent/rules/"
        Skills    string // e.g., ".agent/skills/"
        Workflows string // e.g., ".agent/workflows/"
    }
    ```
    - `resolveRef` の `procedure` case を `workflows/` パスに変更
    - workflows はフラットファイル形式 (`{id}.md`)
*   **Logic**:
    ```go
    case "procedure":
        if ctx.Paths.Workflows != "" {
            return ensureTrailingSlash(ctx.Paths.Workflows) + id + ".md"
        }
        // fallback: skills path (for targets without workflows support)
        return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
    case "capability":
        return ensureTrailingSlash(ctx.Paths.Skills) + id + "/SKILL.md"
    ```

#### [MODIFY] [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: デフォルトパスを `.agent/` に変更、resolvePaths を 3 パス返却に変更、procedures を workflows にフラットファイルでデプロイ
*   **Technical Design**:

    **WorkflowFrontmatter 構造体** (R7: 将来の分離に備え別構造体で定義):
    ```go
    // WorkflowFrontmatter defines the YAML frontmatter for workflow files.
    type WorkflowFrontmatter struct {
        Name        string `yaml:"name"`
        Description string `yaml:"description"`
    }
    ```

    **resolvePaths 変更** (3値返却):
    ```go
    func (a *AntigravityEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string, string) {
        rulesPath := ".agent/rules/"
        skillsPath := ".agent/skills/"
        workflowsPath := ".agent/workflows/"

        for _, target := range resolved.Entities["target"] {
            if target.ID == "antigravity" {
                if paths, ok := target.Raw["paths"].(map[string]any); ok {
                    if r, ok := paths["rules"].(string); ok { rulesPath = r }
                    if s, ok := paths["skills"].(string); ok { skillsPath = s }
                    if w, ok := paths["workflows"].(string); ok { workflowsPath = w }
                }
            }
        }

        if apply {
            return filepath.Join(a.RootDir, rulesPath),
                filepath.Join(a.RootDir, skillsPath),
                filepath.Join(a.RootDir, workflowsPath)
        }
        return filepath.Join(buildDir, "antigravity", rulesPath),
            filepath.Join(buildDir, "antigravity", skillsPath),
            filepath.Join(buildDir, "antigravity", workflowsPath)
    }
    ```

    **resolveTargetPaths 変更** (`Workflows` を含む):
    ```go
    func (a *AntigravityEmitter) resolveTargetPaths(resolved *manifest.ResolvedManifest) TargetPaths {
        tp := TargetPaths{
            Rules:     ".agent/rules/",
            Skills:    ".agent/skills/",
            Workflows: ".agent/workflows/",
        }
        // ... target overrides for "workflows" path ...
        return tp
    }
    ```

*   **Logic**:
    - **Emit() の呼び出し元修正**: `rulesDir, skillsDir, workflowsDir := a.resolvePaths(resolved, buildDir, apply)`
    - **Procedures Emit の変更** (Section 3):
        - 出力先を `skillsDir` から `workflowsDir` に変更
        - ファイル形式を `{id}/SKILL.md` から `{id}.md` (フラットファイル) に変更
        - frontmatter を `WorkflowFrontmatter` に変更
        ```go
        // 3. Emit Procedures (as Workflows - flat .md files)
        outputPath := filepath.Join(workflowsDir, proc.ID+".md")
        ```
    - **Capabilities Emit の変更** (Section 2): 変更なし。引き続き `skillsDir` に `{id}/SKILL.md` として出力
    - **CleanTargetDirs**: `workflowsDir` を追加
        ```go
        if err := CleanTargetDirs(filepath.Join(buildDir, "antigravity")); err != nil {
        ```
        (既に antigravity サブディレクトリ全体をクリーンアップしているため追加不要)
    - **EmitResult.TargetDirs**: `workflowsDir` を追加
        ```go
        return &EmitResult{
            EmittedFiles: emittedFiles,
            TargetDirs:   []string{rulesDir, skillsDir, workflowsDir},
        }, nil
        ```
    - **Check() の修正**: `liveDirs` に `workflowsDir` を追加
        ```go
        rulesDir, skillsDir, workflowsDir := a.resolvePaths(resolved, buildDir, true)
        liveDirs := map[string]string{
            "rules":     rulesDir,
            "skills":    skillsDir,
            "workflows": workflowsDir,
        }
        ```
    - **Check() の `folderName` デフォルト修正**: `.agents/` から `.agent/` に変更
        ```go
        folderName := ".agent/" + cat + "/"
        ```

#### [MODIFY] [codex.go](file://features/tt/internal/prompt/emitter/codex.go)
*   **Description**: デフォルトパスを `.codex/` に変更、AGENTS.md マーカーのパス参照を更新
*   **Technical Design**:

    **resolvePaths 変更**:
    ```go
    func (c *CodexEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string) {
        rulesPath := ".codex/rules/"
        skillsPath := ".codex/skills/"
        // ... (target overrides は既存のまま)
    }
    ```

    **generateMarkerContent 変更**:
    ```go
    func (c *CodexEmitter) generateMarkerContent(policies []*manifest.Entity, skillIDs []string) string {
        rulesPath := ".codex/rules/"
        skillsPath := ".codex/skills/"
        // ...
        sb.WriteString("This project follows structured rules and workflows managed under `.codex/`.\n\n")
        // ...
    }
    ```

    **Check() の修正**: `folderName` デフォルトを `.codex/` に更新
    ```go
    folderName := ".codex/" + cat + "/"
    ```

---

### ターゲット設定ファイル (`prompts/manifest/targets/`)

#### [MODIFY] [antigravity.yaml](file://prompts/manifest/targets/antigravity.yaml)
*   **Description**: パスを `.agent/` に変更し、`workflows` パスを追加
*   **Logic**:
    ```yaml
    apiVersion: agent.meta/v1
    kind: target
    id: antigravity
    includes:
      policy: true
      capability: true
      procedure: true
      subagent: false
    paths:
      rules: .agent/rules/
      skills: .agent/skills/
      workflows: .agent/workflows/
    limits:
      rules:
        max_file_size: 12000
        on_exceed: warn
    ```

#### [MODIFY] [codex.yaml](file://prompts/manifest/targets/codex.yaml)
*   **Description**: パスを `.codex/` に変更
*   **Logic**:
    ```yaml
    apiVersion: agent.meta/v1
    kind: target
    id: codex
    includes:
      policy: true
      capability: true
      procedure: true
      subagent: false
    paths:
      rules: .codex/rules/
      skills: .codex/skills/
    index_file: AGENTS.md
    ```

## Step-by-Step Implementation Guide

1.  **template.go にWorkflowsフィールドを追加**:
    - `TargetPaths` 構造体に `Workflows string` フィールドを追加
    - `resolveRef` の `procedure` case を `Workflows` パスに変更 (fallback 付き)

2.  **template_test.go のテストを更新**:
    - `TargetPaths` の初期化に `Workflows` を追加
    - テストケースのパス期待値を `.agent/` に更新
    - procedure の解決先を `workflows/` パスに変更

3.  **単体テスト実行 (template)**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

4.  **antigravity.yaml と codex.yaml を更新**:
    - antigravity.yaml: パスを `.agent/` に変更、`workflows` 追加
    - codex.yaml: パスを `.codex/` に変更

5.  **antigravity.go を修正**:
    - `WorkflowFrontmatter` 構造体を定義
    - `resolvePaths()` を 3 パス返却に変更、デフォルトを `.agent/` に
    - `resolveTargetPaths()` に `Workflows` を追加
    - `Emit()`: procedures の出力先を `workflowsDir` に変更、フラットファイル形式に
    - `Emit()`: `EmitResult.TargetDirs` に `workflowsDir` を追加
    - `Check()`: `liveDirs` に `workflows` を追加、`folderName` デフォルトを `.agent/` に

6.  **emitter_test.go を更新**:
    - antigravity テストのターゲット YAML パスを更新
    - procedure の出力検証を `workflows/{id}.md` に変更
    - capability の出力検証を `skills/{id}/SKILL.md` に維持

7.  **codex.go を修正**:
    - `resolvePaths()` のデフォルトを `.codex/` に変更
    - `generateMarkerContent()` のパス参照を `.codex/` に変更
    - `Check()` の `folderName` デフォルトを `.codex/` に変更

8.  **codex_test.go を更新**:
    - ターゲット YAML パスを `.codex/` に更新
    - AGENTS.md マーカーテストで `.codex/` を検証

9.  **全体ビルドと単体テスト実行**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

10. **Verification Plan を実行** (Step 11 以降)

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Full Build (最終検証)**:
    ```bash
    ./scripts/process/build.sh
    ```

3.  **E2E Tests**:
    E2E テストは不要。理由: 本変更は tt ツール (CLI) の内部ロジック変更であり、VSCode 拡張機能やフロントエンドに影響しない。全ての変更は単体テストでカバーされる。

### テスト項目設計

依存関係: `antigravity.go/codex.go` -> `template.go` -> `TargetPaths`

ボトムアップ順序:
1. **Step 1 (末端)**: `TargetPaths` と `ResolveTemplateVars` のテスト (template_test.go)
2. **Step 2 (中間)**: `AntigravityEmitter.Emit` / `CodexEmitter.Emit` のテスト (emitter_test.go, codex_test.go)
3. **Step 3 (結合)**: ビルド全体の成功

#### テスト項目一覧

| # | テスト対象 | テストケース | 検証観点 |
|---|:---|:---|:---|
| 1 | `ResolveTemplateVars` | `procedure` -> `.agent/workflows/{id}.md` | 正常系: procedure がフラットファイルパスに解決される |
| 2 | `ResolveTemplateVars` | `capability` -> `.agent/skills/{id}/SKILL.md` | 正常系: capability が skills パスに解決される |
| 3 | `ResolveTemplateVars` | `policy` -> `.agent/rules/{id}.md` | 正常系: policy が rules パスに解決される |
| 4 | `ResolveTemplateVars` | `Workflows` 未設定時の fallback | 異常系: Workflows が空文字列の場合 skills に fallback |
| 5 | `AntigravityEmitter.Emit` | procedures が `workflows/{id}.md` に出力 | 正常系: フラットファイルが生成される |
| 6 | `AntigravityEmitter.Emit` | capabilities が `skills/{id}/SKILL.md` に出力 | 正常系: フォルダ構造が維持される |
| 7 | `AntigravityEmitter.Emit` | buildDir 出力先が `.agent/` | 正常系: buildDir 下に `.agent/` 構成 |
| 8 | `CodexEmitter.Emit` | buildDir 出力先が `.codex/` | 正常系: buildDir 下に `.codex/` 構成 |
| 9 | `CodexEmitter.Emit` | AGENTS.md マーカーが `.codex/` を参照 | 正常系: マーカーセクション内パス |
| 10 | `AntigravityEmitter.Check` | workflows ディレクトリのドリフト検出 | 正常系: 3ディレクトリ全てを検査 |

#### セルフレビュー結果

1. **網羅性**: R1-R7 の全要件が少なくとも 1 つ以上のテスト項目でカバーされている。R6 は変更なしのため対象外。
2. **証拠の十分性**: 各テストはファイルの存在確認だけでなく、出力パスの正確性 (`.agent/` vs `.agents/`, フラットファイル vs フォルダ構造) を検証する。
3. **迂回排除**: procedure が `skills/` に出力されないこと、capability が `workflows/` に出力されないことを暗黙的に確認 (ファイル数の一致)。
4. **依存関係**: ボトムアップ順序に従い、template -> emitter の順でテストが設計されている。

## Documentation

#### [MODIFY] [001-SeparateDeployPaths-AgyAndCodex.md](file://prompts/phases/000-foundation/branches/fix-compile-bug/ideas/001-SeparateDeployPaths-AgyAndCodex.md)
*   **更新内容**: 実装完了後、仕様書の実現方針セクションに実装結果を反映する必要がある場合のみ更新
