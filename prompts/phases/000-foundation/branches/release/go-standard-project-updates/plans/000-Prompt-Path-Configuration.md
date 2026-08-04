# 000-Prompt-Path-Configuration

> **Source Specification**: [000-Prompt-Path-Configuration.md](file://prompts/phases/000-foundation/branches/release/go-standard-project-updates/ideas/000-Prompt-Path-Configuration.md)

## Goal Description

`tt prompt compile` / `deploy` / `update` に、ワークスペース・プロンプトソース・ビルド出力・deploy 先を CLI / 環境変数で指定できるパス解決レイヤーを追加する。新フラグ未指定時は現行挙動（`prompts/` → `tmp/dist/` → 同一 workspace へ deploy）を維持する。あわせて `--help` のパス系フラグ説明をパス式記法で統一し、ラッパースクリプトから全フラグを透過する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1 `--workspace` / `TT_WORKSPACE` | Proposed Changes > paths.go, prompt.go, compiler/*.go |
| R2 `--prompts-dir` / `TT_PROMPTS_DIR` | Proposed Changes > paths.go, compiler.go, update.go, branch_skills.go |
| R3 `--build-dir` / `TT_BUILD_DIR` | Proposed Changes > paths.go, compiler.go, deploy.go |
| R3b `--resolved-manifest` / `TT_RESOLVED_MANIFEST` | Proposed Changes > paths.go, compiler.go |
| R4 `--deploy-root` / `TT_DEPLOY_ROOT` | Proposed Changes > paths.go, deploy.go, emitter/*.go |
| R5 優先順位 CLI > env > yaml > default | Proposed Changes > paths.go `mergePathValue` |
| R6 ラッパースクリプト透過 | Proposed Changes > scripts/code/prompt/*.sh |
| R7 後方互換 | Proposed Changes > paths.go `ResolvePaths` fallback, paths_test.go regression |
| R8 help パス式 | Proposed Changes > prompt.go help 定数, prompt_test.go |
| 任意: tt-user-manual 追記 | Documentation > docs/manual/tt-user-manual.md |
| 検証シナリオ 1–6 | Verification Plan > Automated Verification |
| パス式一覧（仕様書） | Proposed Changes > paths.go コメント + help 定数 |

## Proposed Changes

### features/tt/internal/prompt/compiler（パス解決コア）

#### [NEW] [paths_test.go](file://features/tt/internal/prompt/compiler/paths_test.go)

*   **Description**: `ResolvePaths` の TDD 用単体テスト。**実装前に作成し、Red → Green を確認する。**
*   **Technical Design**:
    ```go
    package compiler

    func TestResolvePaths_DefaultBackwardCompatible(t *testing.T)
    func TestResolvePaths_WorkspaceFlag(t *testing.T)
    func TestResolvePaths_PromptsDirFlag(t *testing.T)
    func TestResolvePaths_BuildDirFlag(t *testing.T)
    func TestResolvePaths_ResolvedManifestFlag(t *testing.T)
    func TestResolvePaths_DeployRootFlag(t *testing.T)
    func TestResolvePaths_PriorityFlagOverEnv(t *testing.T)
    func TestResolvePaths_PriorityEnvOverYAML(t *testing.T)
    func TestResolvePaths_AbsolutePathIgnoresAnchor(t *testing.T)
    func TestResolvePaths_ProjectRelativeToWorkspace(t *testing.T)
    func TestResolvePaths_SchemasDir(t *testing.T)
    func TestResolvePaths_GitWatchPaths(t *testing.T)
    func TestResolvePaths_InferredWorkspaceFromProject(t *testing.T)
    ```
*   **Logic**（テーブル駆動 `tests := []struct{...}`）:
    *   **DefaultBackwardCompatible**: `testdata/valid` の `project.yaml` のみ指定 → `Workspace` = 3 階層上推論、`ProjectYAML` = `{workspace}/prompts/manifest/project.yaml`、`BuildDirAbs` = `{workspace}/tmp/dist`、`SchemasDirAbs` = `{workspace}/prompts/manifest/schemas`、`DeployRootAbs` = `Workspace`
    *   **WorkspaceFlag**: `--workspace` = `.` + `--project prompts/manifest/project.yaml` → project が workspace 相対で解決
    *   **PromptsDirFlag**: `--prompts-dir catalog/prompts` → `SchemasDirAbs` = `{workspace}/catalog/prompts/manifest/schemas`、`GitWatchPaths()` = `catalog/prompts/manifest/`, `catalog/prompts/memory/`
    *   **BuildDirFlag**: `--build-dir .cache/tt-dist` → `BuildDirAbs` = `{workspace}/.cache/tt-dist`
    *   **ResolvedManifestFlag**: `--resolved-manifest out/custom.yaml` → 出力先が `{workspace}/out/custom.yaml`
    *   **BuildDirWithoutResolvedManifest**: `--build-dir .cache/tt-dist` のみ → `ResolvedManifestAbs` = `{workspace}/.cache/tt-dist/manifest.resolved.yaml`（R3 フォールバック）
    *   **DeployRootFlag**: `--deploy-root /other/ws` → `DeployRootAbs` = `/other/ws`（workspace とは独立）
    *   **PriorityFlagOverEnv**: env `TT_BUILD_DIR=env-dist` + flag `--build-dir cli-dist` → `BuildDirRel` = `cli-dist`
    *   **PriorityEnvOverYAML**: env `TT_BUILD_DIR=env-dist`、flag 未指定 → yaml `tmp/dist/` より env 優先
    *   **AbsolutePathIgnoresAnchor**: `--deploy-root /abs/path` → CWD 無関係に `/abs/path`
    *   **SchemasDir**: `{workspace} + {prompts-dir} + manifest/schemas` 式と一致

#### [NEW] [paths.go](file://features/tt/internal/prompt/compiler/paths.go)

*   **Description**: CLI / 環境変数 / project.yaml / 組み込みデフォルトをマージし、絶対パスを返す。
*   **Technical Design**:
    ```go
    package compiler

    import (
        "os"
        "path/filepath"
    )

    // PathOptions holds raw path inputs from CLI and environment.
    type PathOptions struct {
        CWD string

        WorkspaceFlag        string
        WorkspaceFlagSet     bool
        PromptsDirFlag       string
        PromptsDirFlagSet    bool
        ProjectFlag          string
        ProjectFlagSet       bool
        BuildDirFlag         string
        BuildDirFlagSet      bool
        ResolvedManifestFlag string
        ResolvedManifestFlagSet bool
        DeployRootFlag       string
        DeployRootFlagSet    bool
    }

    // PathConfig holds fully resolved absolute paths used by compile/deploy/update.
    type PathConfig struct {
        Workspace         string // 推論または --workspace（絶対パス）
        ProjectYAML       string // 解決済み project.yaml 絶対パス
        PromptsDir        string // workspace 相対（例: "prompts", "catalog/prompts"）
        BuildDir          string // workspace 相対（例: "tmp/dist/"）
        ResolvedManifest  string // workspace 相対（例: "tmp/dist/manifest.resolved.yaml"）
        DeployRoot        string // deploy apply 時のルート（絶対パス）
        SchemasDir        string // 絶対: {workspace} + {prompts-dir} + manifest/schemas
    }

    // GitWatchPaths returns workspace-relative paths for prompt update git monitoring.
    func (pc PathConfig) GitWatchPaths() []string {
        return []string{
            filepath.ToSlash(filepath.Join(pc.PromptsDir, "manifest")),
            filepath.ToSlash(filepath.Join(pc.PromptsDir, "memory")),
        }
    }

    func (pc PathConfig) BuildDirAbs() string {
        return filepath.Clean(filepath.Join(pc.Workspace, pc.BuildDir))
    }

    func (pc PathConfig) ResolvedManifestAbs() string {
        return filepath.Clean(filepath.Join(pc.Workspace, pc.ResolvedManifest))
    }

    func (pc PathConfig) PromptsDirAbs() string {
        return filepath.Clean(filepath.Join(pc.Workspace, pc.PromptsDir))
    }

    // ResolvePaths merges CLI, env, project.yaml, and built-in defaults.
    // Priority: CLI flag (Changed) > env > project.yaml > built-in default.
    func ResolvePaths(opts PathOptions) (*PathConfig, error)
    ```
*   **Logic** (`ResolvePaths` 手順):
    1.  `cwd := opts.CWD`（空なら `os.Getwd()`）
    2.  **project 入力のマージ**:
        ```go
        projectRel := mergePathValue(
            opts.ProjectFlag, opts.ProjectFlagSet,
            os.Getenv("TT_PROJECT"), // 予約（仕様に無いが将来用、今回は未使用）
            "prompts/manifest/project.yaml",
        )
        ```
    3.  **workspace 解決**:
        ```go
        workspaceRaw := mergePathValue(
            opts.WorkspaceFlag, opts.WorkspaceFlagSet,
            os.Getenv("TT_WORKSPACE"),
            "",
        )
        var workspace string
        if workspaceRaw != "" {
            workspace = resolvePath(cwd, workspaceRaw)
        } else {
            // 後方互換: project.yaml 位置から 3 階層上
            projectCandidate := resolveProjectCandidate(cwd, workspaceRaw, projectRel)
            inferred, err := ResolveProjectRoot(projectCandidate)
            if err != nil {
                return nil, err
            }
            workspace = inferred
        }
        ```
    4.  **project.yaml 絶対パス**:
        ```go
        projectCandidate := resolveProjectCandidate(cwd, workspace, projectRel)
        projectYAML, err := filepath.Abs(projectCandidate)
        ```
        `resolveProjectCandidate(cwd, workspace, rel)`:
        - `rel` が絶対 → そのまま
        - `workspace != ""` → `filepath.Join(workspace, rel)`
        - 否则 → `filepath.Join(cwd, rel)`
    5.  `cfg, err := LoadConfig(projectYAML)`
    6.  **prompts-dir**:
        ```go
        promptsDir := mergePathValue(
            opts.PromptsDirFlag, opts.PromptsDirFlagSet,
            os.Getenv("TT_PROMPTS_DIR"),
            "prompts",
        )
        ```
    7.  **project デフォルト再解決**（`--project` 未指定時）:
        - `ProjectFlagSet == false` かつ env も空 → 実効 project = `filepath.Join(promptsDir, "manifest", "project.yaml")` を workspace 相対として使用（既に LoadConfig 済みの場合は projectYAML が明示指定されていればスキップ）
    8.  **build-dir**:
        ```go
        yamlBuild := cfg.Defaults.BuildDir
        if yamlBuild == "" {
            yamlBuild = "tmp/dist/"
        }
        buildDir := mergePathValue(
            opts.BuildDirFlag, opts.BuildDirFlagSet,
            os.Getenv("TT_BUILD_DIR"),
            yamlBuild,
        )
        ```
    9.  **resolved-manifest**:
        ```go
        yamlResolved := cfg.Outputs.ResolvedManifest
        resolved := mergePathValue(
            opts.ResolvedManifestFlag, opts.ResolvedManifestFlagSet,
            os.Getenv("TT_RESOLVED_MANIFEST"),
            yamlResolved,
        )
        if resolved == "" {
            resolved = filepath.Join(buildDir, "manifest.resolved.yaml")
        }
        ```
    10. **deploy-root**:
        ```go
        deployRaw := mergePathValue(
            opts.DeployRootFlag, opts.DeployRootFlagSet,
            os.Getenv("TT_DEPLOY_ROOT"),
            "", // default applied below
        )
        deployRoot := workspace
        if deployRaw != "" {
            deployRoot = resolvePath(cwd, deployRaw)
        }
        ```
    11. **schemas**:
        ```go
        schemasDir := filepath.Join(workspace, promptsDir, "manifest", "schemas")
        ```
    12. `PathConfig` を返す

    **`mergePathValue(flag, flagSet, env, fallback string) string`**:
    ```go
    func mergePathValue(flag string, flagSet bool, env, fallback string) string {
        if flagSet {
            return flag
        }
        if env != "" {
            return env
        }
        return fallback
    }
    ```

    **`resolvePath(anchor, p string) string`**（絶対パス対応）:
    ```go
    func resolvePath(anchor, p string) string {
        if filepath.IsAbs(p) {
            return filepath.Clean(p)
        }
        return filepath.Clean(filepath.Join(anchor, p))
    }
    ```

    **パス式対応表**（コードコメントとして `paths.go` 先頭に転記）:

    | 用途 | パス式 |
    |---|---|
    | prompts ソースルート | `{workspace} + {prompts-dir}` |
    | project.yaml（デフォルト） | `{workspace} + {prompts-dir} + manifest/project.yaml` |
    | スキーマディレクトリ | `{workspace} + {prompts-dir} + manifest/schemas` |
    | update git 監視（manifest） | `{workspace} + {prompts-dir} + manifest/` |
    | update git 監視（memory） | `{workspace} + {prompts-dir} + memory/` |
    | build 出力ルート | `{workspace} + {build-dir}` |
    | resolved manifest（フォールバック） | `{workspace} + {build-dir} + manifest.resolved.yaml` |
    | digest ファイル | `{workspace} + {build-dir} + .compile-digest[-{target}]` |
    | deploy 先（apply） | `{deploy-root} + .cursor/rules/` 等 |
    | compile staging | `{workspace} + {build-dir} + cursor/.cursor/rules/` |

#### [MODIFY] [config.go](file://features/tt/internal/prompt/compiler/config.go)

*   **Description**: `ResolveProjectRoot` は後方互換のため維持。内部コメントで `ResolvePaths` への移行を明記。
*   **Logic**: 既存実装（`project.yaml` の 3 階層上）を変更しない。`paths.go` から呼び出す。

#### [MODIFY] [compiler.go](file://features/tt/internal/prompt/compiler/compiler.go)

*   **Description**: `CompileOptions` に `Paths *PathConfig` を追加し、ハードコードパスを置換。
*   **Technical Design**:
    ```go
    type CompileOptions struct {
        ProjectPath string      // 後方互換: PathConfig 未設定時のみ使用
        Paths       *PathConfig // 新: 解決済みパス（優先）
        DryRun      bool
        Target      string
        Apply       bool
        EmitMode    emitter.EmitMode
        EmitDryRun  bool
    }
    ```
*   **Logic**:
    1.  冒頭で `paths` を解決:
        ```go
        paths := opts.Paths
        if paths == nil {
            var err error
            paths, err = ResolvePaths(PathOptions{
                CWD: mustGetwd(),
                ProjectFlag: opts.ProjectPath,
                ProjectFlagSet: true,
            })
            if err != nil { return nil, err }
        }
        rootDir := paths.Workspace
        cfg, err := LoadConfig(paths.ProjectYAML)
        ```
    2.  スキーマ: `schemasDir := paths.SchemasDir`（`filepath.Join(rootDir, "prompts", "manifest", "schemas")` を置換）
    3.  resolved manifest 書き込み: `paths.ResolvedManifestAbs()`
    4.  emitter 生成:
        ```go
        emitObj = emitter.NewCursorEmitter(rootDir, paths.DeployRoot)
        ```
    5.  buildDir: `buildDir := paths.BuildDirAbs()`
    6.  `Apply` 時は emitter が `DeployRoot` を使用（後述）

#### [MODIFY] [deploy.go](file://features/tt/internal/prompt/compiler/deploy.go)

*   **Description**: `DeployOptions` に `Paths *PathConfig` を追加。digest / drift / compile 呼び出しを PathConfig 経由に。
*   **Technical Design**:
    ```go
    type DeployOptions struct {
        ProjectPath string
        Paths       *PathConfig
        Target      string
        Force       bool
        DryRun      bool
        Mode        emitter.EmitMode
    }
    ```
*   **Logic**:
    1.  `paths` 解決（compile と同パターン）
    2.  `ComputeSourceDigest(cfg, paths.Workspace)` — sources glob は workspace 相対のまま
    3.  `buildDir := paths.BuildDirAbs()`
    4.  `CheckDrift(paths.Workspace, paths.DeployRoot, opts.ProjectPath, target)` — drift 比較は deploy 先ファイルを `DeployRoot` 基準で
    5.  `Compile(CompileOptions{Paths: paths, ...})`

#### [MODIFY] [update.go](file://features/tt/internal/prompt/compiler/update.go)

*   **Description**: git 監視パスを `PathConfig.GitWatchPaths()` から生成。メタデータは workspace 基準のまま。
*   **Technical Design**:
    ```go
    type UpdateOptions struct {
        ProjectPath string
        Paths       *PathConfig
        Target      string
        Force       bool
        DryRun      bool
    }

    // CheckForChanges checks if source files have changed since the given time
    // by examining git log for paths under prompts-dir.
    func CheckForChanges(workspaceRoot, promptsDir string, since time.Time) (bool, error)
    ```
*   **Logic** (`CheckForChanges` 変更):
    ```go
    func CheckForChanges(workspaceRoot, promptsDir string, since time.Time) (bool, error) {
        sinceStr := since.Format(time.RFC3339)
        watchPaths := []string{
            filepath.ToSlash(filepath.Join(promptsDir, "manifest")),
            filepath.ToSlash(filepath.Join(promptsDir, "memory")),
        }
        args := append([]string{
            "log", "--since=" + sinceStr,
            "--name-only", "--pretty=format:", "--",
        }, watchPaths...)
        cmd := exec.Command("git", args...)
        cmd.Dir = workspaceRoot
        // ... 以下既存と同じ（git 不可時 return true, nil）
    }
    ```
    `Update()` 内: `paths, _ := resolve paths` → `CheckForChanges(paths.Workspace, paths.PromptsDir, updatedAt)` → `metaDir := filepath.Join(paths.Workspace, resolve.MetaDir(t))`

#### [MODIFY] [deploy_test.go](file://features/tt/internal/prompt/compiler/deploy_test.go)

*   **Description**: deploy-root / build-dir ケース追加（TDD: paths 実装後）。
*   **Logic**:
    *   `TestDeploy_CustomBuildDir`: `Paths` に `BuildDir: ".cache/tt-dist"` → digest が `.cache/tt-dist/.compile-digest` に作成、`tmp/dist/` には無い
    *   `TestDeploy_CustomDeployRoot`: 別 temp dir を `DeployRoot` に指定 → `.agent/rules/` が deploy 先側にのみ生成

#### [MODIFY] [compiler_test.go](file://features/tt/internal/prompt/compiler/compiler_test.go)

*   **Logic**:
    *   `TestCompile_CustomBuildDir`: resolved manifest が `{build-dir}/manifest.resolved.yaml` に書かれる
    *   既存 `TestCompile_Valid` は `Paths` 未指定でも pass（後方互換）

#### [MODIFY] [update_test.go](file://features/tt/internal/prompt/compiler/update_test.go)

*   **Logic**:
    *   `CheckForChanges` 呼び出しシグネチャ変更に合わせて修正
    *   `TestCheckForChanges_CustomPromptsDir`: `promptsDir = "custom/prompts"` で git log 引数が `custom/prompts/manifest`, `custom/prompts/memory` になること（`exec.Command` をラップするか、integration 的に git repo + paths で検証）

---

### features/tt/internal/prompt/emitter（deploy-root 分離）

#### [MODIFY] [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)（antigravity.go, claude_code.go, codex.go も同様）

*   **Description**: Emitter に `DeployRoot` を追加。`apply=true` 時のパス基準を `DeployRoot` に、`ScanBranchSkills` / drift 用は `RootDir`（workspace）のまま。
*   **Technical Design**:
    ```go
    type CursorEmitter struct {
        RootDir    string // workspace: branch skills scan, limits context
        DeployRoot string // apply=true output base; defaults to RootDir if empty
    }

    func NewCursorEmitter(workspaceRoot, deployRoot string) *CursorEmitter {
        if deployRoot == "" {
            deployRoot = workspaceRoot
        }
        return &CursorEmitter{RootDir: workspaceRoot, DeployRoot: deployRoot}
    }
    ```
*   **Logic** (`resolvePaths`):
    ```go
    if apply {
        return filepath.Join(c.DeployRoot, rulesPath),
            filepath.Join(c.DeployRoot, skillsPath)
    }
    return filepath.Join(buildDir, "cursor", rulesPath),
        filepath.Join(buildDir, "cursor", skillsPath)
    ```
    `Check()` の `livePath` も `DeployRoot` 基準に変更:
    ```go
    livePath = filepath.Join(c.DeployRoot, relPath)
    ```

#### [MODIFY] [branch_skills.go](file://features/tt/internal/prompt/emitter/branch_skills.go)

*   **Description**: `prompts-dir` 対応。
*   **Technical Design**:
    ```go
    func ScanBranchSkills(workspaceRoot, promptsDir string) ([]BranchSkill, error) {
        if promptsDir == "" {
            promptsDir = "prompts"
        }
        branchesDir := filepath.Join(workspaceRoot, promptsDir, "memory", "branches")
        // ... 以下既存
    }
    ```
*   **Logic**: 各 emitter の `Emit` 内 `ScanBranchSkills(c.RootDir)` → `ScanBranchSkills(c.RootDir, promptsDir)`。`promptsDir` は `compiler.Compile` から emitter に渡す（`Emit` に新引数を追加するか、emitter 構造体に `PromptsDir string` フィールド追加）。

    **採用**: `CursorEmitter` 等に `PromptsDir string` フィールドを追加し、`NewCursorEmitter(workspace, deployRoot, promptsDir)` とする。

#### [MODIFY] [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go), cursor_test.go 等

*   **Logic**: `New*Emitter(tempDir, tempDir, "prompts")` に呼び出し更新。deploy-root 分離テスト:
    ```go
    func TestCursorEmitter_DeployRootSeparate(t *testing.T) {
        workspace := t.TempDir()
        deployRoot := t.TempDir()
        emitter := NewCursorEmitter(workspace, deployRoot, "prompts")
        // apply=true emit → ファイルが deployRoot/.cursor/ 配下のみ
    }
    ```

---

### features/tt/cmd（CLI フラグ・help）

#### [NEW] [prompt_test.go](file://features/tt/cmd/prompt_test.go)

*   **Description**: help 定数とフラグ登録のテスト（R8）。
*   **Logic**:
    ```go
    func TestPromptPathHelpConstants_ContainPathExpressions(t *testing.T) {
        for _, s := range []string{helpWorkspace, helpPromptsDir, helpProject, helpBuildDir, helpResolvedManifest, helpDeployRoot} {
            if !strings.Contains(s, "Default:") { t.Errorf(...) }
        }
        if !strings.Contains(helpProject, "{workspace} + {prompts-dir}") { t.Errorf(...) }
    }
    ```

#### [MODIFY] [prompt.go](file://features/tt/cmd/prompt.go)

*   **Description**: パス系フラグ追加、共有 help 定数、PathOptions 構築、サブコマンド Long 追記。
*   **Technical Design** — help 定数（仕様書確定案をそのまま使用）:
    ```go
    const (
        helpWorkspace = "Workspace root for path resolution. Relative to CWD unless absolute. Default: inferred from --project (see path expr: {workspace}). Env: TT_WORKSPACE (flag wins)."
        helpPromptsDir = "Prompts source root (manifest and memory). Relative to workspace unless absolute. Default: {workspace} + {prompts-dir} (segment: prompts). Env: TT_PROMPTS_DIR (flag wins)."
        helpProject = "Path to project.yaml. Relative to workspace if set, else CWD unless absolute. Default: {workspace} + {prompts-dir} + manifest/project.yaml. Env: (none)."
        helpBuildDir = "Build output directory for staging and digests. Relative to workspace unless absolute. Default: {workspace} + {build-dir} (from project.yaml defaults.build_dir or tmp/dist/). Env: TT_BUILD_DIR (flag wins)."
        helpResolvedManifest = "Resolved manifest output path. Relative to workspace unless absolute. Default: {workspace} + {resolved-manifest} (from project.yaml outputs.resolved_manifest or {build-dir} + manifest.resolved.yaml). Env: TT_RESOLVED_MANIFEST (flag wins)."
        helpDeployRoot = "Root directory for editor config deployment in apply mode. Relative to CWD unless absolute. Default: {deploy-root} = {workspace}. Env: TT_DEPLOY_ROOT (flag wins)."
    )
    ```
*   **Logic**:
    1.  共有変数追加:
        ```go
        var (
            promptWorkspace, promptPromptsDir, promptProject string
            promptBuildDir, promptResolvedManifest, promptDeployRoot string
        )
        ```
    2.  `addPromptPathFlags(cmd *cobra.Command)` を定義し compile/deploy/update 各 init で呼ぶ
    3.  `buildPathOptions(cmd *cobra.Command) compiler.PathOptions`:
        ```go
        func buildPathOptions(cmd *cobra.Command) compiler.PathOptions {
            cwd, _ := os.Getwd()
            f := cmd.Flags()
            ws, _ := f.GetString("workspace")
            wsSet := f.Changed("workspace")
            // ... 各フラグで Changed() を取得
            return compiler.PathOptions{CWD: cwd, WorkspaceFlag: ws, WorkspaceFlagSet: wsSet, ...}
        }
        ```
    4.  `runPromptCompile` / `Deploy` / `Update`:
        ```go
        pathOpts := buildPathOptions(cmd)
        paths, err := compiler.ResolvePaths(pathOpts)
        result, err := compiler.Compile(compiler.CompileOptions{Paths: paths, ...})
        ```
    5.  サブコマンド `Long` 末尾追加:
        ```
        Path flags resolve in order: --workspace, --prompts-dir, --project, --build-dir,
        --resolved-manifest, --deploy-root. Paths compose as {workspace} + {prompts-dir} + ...
        See --help for defaults and TT_* env vars.
        ```

---

### scripts/code/prompt（ラッパー透過）

#### [MODIFY] [compile.sh](file://scripts/code/prompt/compile.sh), [deploy.sh](file://scripts/code/prompt/deploy.sh), [update.sh](file://scripts/code/prompt/update.sh)

*   **Description**: 全引数を `tt` に透過（R6）。
*   **Logic**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/../_resolve_tool.sh"
    exec "$TOOL" prompt compile "$@"
    ```
    deploy.sh → `prompt deploy "$@"`、update.sh → `prompt update "$@"`

---

### tests/tt（CLI 統合テスト）

#### [NEW] [tt_prompt_paths_test.go](file://tests/tt/tt_prompt_paths_test.go)

*   **Description**: 仕様書検証シナリオ 1–6 を自動化。CLI サブコマンド + ファイルシステム検証。
*   **Technical Design**:
    ```go
    //go:build integration

    package integration_test

    func TestPromptPaths_BackwardCompatible(t *testing.T)
    func TestPromptPaths_CustomBuildDir(t *testing.T)
    func TestPromptPaths_DeployRootSeparate(t *testing.T)
    func TestPromptPaths_WrapperScriptPassesFlags(t *testing.T)
    func TestPromptPaths_HelpShowsPathExpressions(t *testing.T)
    ```
*   **Logic**:
    *   **BackwardCompatible**: `testdata/valid` を temp に copy → `runTTInDir(tmp, "prompt", "compile", "--dry-run", "--target", "cursor")` → exit 0
    *   **CustomBuildDir**: `--build-dir .cache/tt-dist` + compile → `.cache/tt-dist/manifest.resolved.yaml` 存在、`tmp/dist/manifest.resolved.yaml` 非更新
    *   **DeployRootSeparate**: workspace temp + deployRoot 別 temp → `prompt deploy --force --target cursor --deploy-root <deployRoot>` → `<deployRoot>/.cursor/rules/` にファイル、workspace 側 `.cursor/` 無し
    *   **WrapperScriptPassesFlags**: `bash scripts/code/prompt/deploy.sh --help` または dry-run で `--deploy-root` が効くこと
    *   **HelpShowsPathExpressions**: `tt prompt compile --help` stdout に `{workspace} + {prompts-dir}` を含む

---

## Step-by-Step Implementation Guide

1.  **Path resolution tests (Red)**:
    *   Create `features/tt/internal/prompt/compiler/paths_test.go` with all test cases above.
    *   Run `./scripts/process/build.sh --skip-frontend --skip-etc` → compile fails (paths.go missing).

2.  **Path resolution implementation (Green)**:
    *   Create `paths.go` with `PathOptions`, `PathConfig`, `ResolvePaths`, helpers.
    *   Re-run build until `paths_test.go` passes.

3.  **Compiler / deploy / update integration of PathConfig**:
    *   Modify `CompileOptions`, `DeployOptions`, `UpdateOptions`.
    *   Update `compiler.go`, `deploy.go`, `update.go`, `CheckDrift` signature.
    *   Fix existing `compiler_test.go`, `deploy_test.go`, `update_test.go`.
    *   Add new test cases for custom build-dir / deploy-root.

4.  **Emitter deploy-root separation**:
    *   Update all four emitters + `branch_skills.go` + emitter tests.
    *   Pass `promptsDir` from `Compile()` into emitter constructors.

5.  **CLI flags and help (R8)**:
    *   Add help constants and flags to `prompt.go`.
    *   Create `prompt_test.go` for help string assertions.
    *   Wire `buildPathOptions` → `ResolvePaths` in run functions.

6.  **Shell wrapper passthrough**:
    *   Replace case loops in `scripts/code/prompt/*.sh` with `"$@"` exec.

7.  **Integration tests**:
    *   Create `tests/tt/tt_prompt_paths_test.go`.
    *   Run full verification (see below).

8.  **Documentation**:
    *   Update `docs/manual/tt-user-manual.md` with new flags and env vars.

9.  **Run Verification Plan** (Automated Verification section below).

---

## Verification Plan

### Test Item Design (§11 Self-Review)

**ボトムアップ順序**:
1. `paths_test.go` — パス解決（末端）
2. `compiler_test.go` / `deploy_test.go` / `update_test.go` / emitter tests — コンパイラ・エミッタ
3. `prompt_test.go` — CLI help
4. `tt_prompt_paths_test.go` — E2E CLI

**§11.3 観点チェックリスト**:

| # | 観点 | 対応テスト |
|---|---|---|
| 1 | 正常系 | DefaultBackwardCompatible, Deploy first run, compile valid |
| 2 | 異常系・境界値 | AbsolutePathIgnoresAnchor, 空 workspace 推論 |
| 3 | 外部連携 | CheckForChanges git log 引数、wrapper script exec |
| 4 | データ一貫性 | resolved manifest / digest 出力位置 |
| 5 | 状態遷移 | deploy-root 分離後 workspace 側 unchanged |
| 6 | 設定反映 | flag > env > yaml 優先 |
| 7 | 副作用 | build-dir 変更時 tmp/dist 非汚染 |

**§11.4 セルフレビュー結果**: 上記テスト群が成功すれば、仕様書シナリオ 1–6 および R1–R8 をカバーできる。git 監視の完全検証は `update_test` + 統合テストで git init リポジトリを用いる。迂回排除: emitter テストで `DeployRoot != RootDir` 時に出力先ディレクトリを明示 assert する。

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: 全 Go パッケージ（特に `features/tt/internal/prompt/compiler`, `features/tt/cmd`）のテストが PASS。`paths_test`, `prompt_test`, emitter テスト失敗が無いこと。

2.  **Integration Tests**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "tt" --specify "PromptPaths"
    ```
    *   **Log Verification**:
        - `TestPromptPaths_BackwardCompatible` PASS
        - `TestPromptPaths_CustomBuildDir` PASS
        - `TestPromptPaths_DeployRootSeparate` PASS
        - `TestPromptPaths_WrapperScriptPassesFlags` PASS
        - `TestPromptPaths_HelpShowsPathExpressions` PASS

3.  **E2E Tests (新規)**:
    #### [NEW] [tt_prompt_paths_test.go](file://tests/tt/tt_prompt_paths_test.go)
    *   **テストケース**: 上記 5 関数
    *   **検証ポイント**: 仕様書「検証シナリオ」1, 2, 3, 6 および R8 help 表示

    E2E が CLI + filesystem 操作のため必須。既存 `runTTInDir` / `copyTestdata` ヘルパーを再利用。

### 総合判定（§12 — 実装完了後に記録）

実装完了時、以下を walkthrough または PR コメントに記載:

```markdown
### 総合判定結果

**判定**: （実装者が記入）

#### テスト結果サマリ
- paths_test: N/N
- compiler/deploy/update/emitter tests: N/N
- tt_prompt_paths_test: N/N

#### チェック項目
| 1 | スキップ | ✅/⚠️/❌ |
| 2 | 部分エラー | ✅/⚠️/❌ |
| ... | ... | ... |
```

---

## Documentation

#### [MODIFY] [tt-user-manual.md](file://docs/manual/tt-user-manual.md)

*   **更新内容**:
    *   `prompt compile` / `deploy` / `update` の「主なフラグ」に `--workspace`, `--prompts-dir`, `--build-dir`, `--resolved-manifest`, `--deploy-root` を追加
    *   環境変数表に `TT_WORKSPACE`, `TT_PROMPTS_DIR`, `TT_BUILD_DIR`, `TT_RESOLVED_MANIFEST`, `TT_DEPLOY_ROOT` を追加
    *   優先順位: `CLI フラグ > 環境変数 > project.yaml > 組み込みデフォルト`
    *   パス式の説明（仕様書「パス式」節の主要パス一覧を転記）
    *   使用例 4 件（仕様書「使用例」節と同一）
