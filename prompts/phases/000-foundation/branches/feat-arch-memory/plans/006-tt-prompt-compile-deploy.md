# 006-tt-prompt-compile-deploy

> **Source Specification**: [006-tt-prompt-compile-deploy.md](file://prompts/phases/000-foundation/branches/feat-arch-memory/ideas/006-tt-prompt-compile-deploy.md)

## Goal Description

`agentctl` のプロンプトコンパイル・デプロイ機能を `tt` コマンドのサブコマンド（`tt prompt compile`, `tt prompt deploy`, `tt prompt update`）として統合する。agentctl の内部パッケージ（compiler, manifest, memory, emitter）を `features/tt/internal/prompt/` に移植し、ターゲット名称解決の共通化、テンプレート変数の拡張、スクリプトの書き換え、プロシージャの追加を行う。

本実装計画は規模が大きいため、以下の3パートに分割する:

- **Part 1（本計画書）**: 基盤構築 - ターゲット名称解決の共通化（R1）、パッケージ移植（R6）、go.mod 更新
- **Part 2**: コマンド実装 - `tt prompt compile/deploy/update`（R2, R3, R4）、テンプレート変数拡張（R7）
- **Part 3**: スクリプト書き換え（R5）、architecture-maintainer 簡潔化（R8）、prompt-update プロシージャ（R9）、ドキュメント更新

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1. ターゲット名称解決の共通化 | Proposed Changes > pkg/resolve > target.go |
| R6. agentctl の内部パッケージの移植 | Proposed Changes > features/tt/internal/prompt/* |
| go.mod の依存追加 | Proposed Changes > features/tt > go.mod |
| R2-R4 (compile/deploy/update) | Part 2 で対応 |
| R5 (スクリプト書き換え) | Part 3 で対応 |
| R7 (テンプレート変数拡張) | Part 2 で対応 |
| R8 (architecture-maintainer 簡潔化) | Part 3 で対応 |
| R9 (prompt-update プロシージャ) | Part 3 で対応 |

## Proposed Changes

### pkg/resolve（ターゲット名称解決の共通化）

#### [NEW] [target_test.go](file://pkg/resolve/target_test.go)
*   **Description**: ターゲット名称解決の単体テスト。TDD により先に作成する。
*   **Technical Design**:
    ```go
    package resolve

    func TestResolveTarget_ExactMatch(t *testing.T)
    func TestResolveTarget_AliasMatch(t *testing.T)
    func TestResolveTarget_PrefixMatch(t *testing.T)
    func TestResolveTarget_AmbiguousError(t *testing.T)
    func TestResolveTarget_AllowAll(t *testing.T)
    func TestResolveTarget_DisallowAll(t *testing.T)
    func TestResolveTarget_UnknownInput(t *testing.T)
    func TestResolveTarget_EmptyInput(t *testing.T)
    func TestResolveTargets_All(t *testing.T)
    func TestResolveTargets_Single(t *testing.T)
    ```
*   **Logic**: テーブル駆動テストで以下のケースを検証:
    - 完全一致: `"antigravity"` -> `"antigravity"`, `"cursor"` -> `"cursor"`, `"claude-code"` -> `"claude-code"`, `"codex"` -> `"codex"`, `"all"` -> `"all"`
    - エイリアス一致: `"ag"` -> `"antigravity"`, `"agy"` -> `"antigravity"`, `"claude"` -> `"claude-code"`, `"vscode"` -> `"code"`(エディタ向け)
    - 前方部分一致: `"anti"` -> `"antigravity"`, `"cur"` -> `"cursor"`, `"cl"` -> `"claude-code"`, `"co"` -> `"codex"`, `"al"` -> `"all"`
    - 曖昧エラー: `"a"` -> エラー（候補: `all`, `antigravity`）、`"c"` -> エラー（候補: `claude-code`, `codex`, `cursor`）
    - `allowAll=false` 時: `"all"` -> エラー
    - 空文字列: エラー
    - 不明な入力: `"xyz"` -> エラー
    - `ResolveTargets("all")` -> `["antigravity", "claude-code", "codex", "cursor"]`（ソート済み）
    - `ResolveTargets("anti")` -> `["antigravity"]`

#### [NEW] [target.go](file://pkg/resolve/target.go)
*   **Description**: ターゲット名称解決の共通ロジック。前方部分一致、エイリアス、曖昧性エラーに対応。
*   **Technical Design**:
    ```go
    package resolve

    // EnvKeyTarget is the environment variable name for the default target.
    const EnvKeyTarget = "TT_TARGET"

    // KnownTargets is the list of canonical target names (sorted).
    var KnownTargets = []string{"antigravity", "claude-code", "codex", "cursor"}

    // TargetAliases maps alias names to canonical target names.
    var TargetAliases = map[string]string{
        "ag":    "antigravity",
        "agy":   "antigravity",
        "claude": "claude-code",
    }

    // AllTarget is the special value representing all targets.
    const AllTarget = "all"

    // ResolveTarget resolves a target name using prefix matching.
    // Returns an error if the input is ambiguous (multiple matches).
    // allowAll controls whether "all" is a valid target name.
    func ResolveTarget(input string, allowAll bool) (string, error)

    // ResolveTargets resolves a target name and returns the list of
    // concrete target names. When input is "all", returns all known targets.
    func ResolveTargets(input string) ([]string, error)

    // MetaDir returns the metadata directory path for a given target.
    func MetaDir(target string) string
    ```
*   **Logic**:
    1. 入力が空なら `fmt.Errorf("target name cannot be empty")` を返す
    2. `TargetAliases` に完全一致があれば、対応する正規名に変換する
    3. 正規名（`KnownTargets` + `"all"`）に対して完全一致チェック
    4. 完全一致がなければ前方部分一致を行う:
       - 対象リスト: `KnownTargets` + (allowAll なら `"all"` も含む)
       - `strings.HasPrefix(canonical, input)` で候補を収集
    5. 候補数が 0 → エラー `"unknown target %q"`
    6. 候補数が 1 → その候補を返す
    7. 候補数が 2以上 → エラー `"ambiguous target %q: matches %v"`
    8. 結果が `"all"` かつ `allowAll=false` → エラー `"target \"all\" is not allowed in this context"`
    9. `ResolveTargets` は `ResolveTarget(input, true)` を呼び、結果が `"all"` なら `KnownTargets` のコピーを返す。それ以外は `[]string{result}` を返す
    10. `MetaDir` は以下のマッピングで返す:

    ```go
    var targetMetaDirs = map[string]string{
        "antigravity": ".agent/.meta/",
        "cursor":      ".cursor/.meta/",
        "claude-code": ".claude/.meta/",
        "codex":       ".agents/.meta/",
    }
    ```

---

### pkg/detect（既存エディタ解決のリファクタリング）

#### [MODIFY] [editor_test.go](file://pkg/detect/editor_test.go)
*   **Description**: `ParseEditor` のテストを、`resolve.ResolveTarget` との統合に合わせて更新する。
*   **Technical Design**: 既存テストに `"all"` が拒否されるケースを追加。
*   **Logic**: `ParseEditor("all")` はエラーを返すことを検証。

#### [MODIFY] [editor.go](file://pkg/detect/editor.go)
*   **Description**: `ParseEditor` を `resolve.ResolveTarget` を内部で利用する形にリファクタリングする。
*   **Technical Design**:
    ```go
    func ParseEditor(s string) (Editor, error) {
        // "all" is not valid for editors
        resolved, err := resolve.ResolveTarget(s, false)
        if err != nil {
            return "", err
        }
        return Editor(resolved), nil
    }
    ```
*   **Logic**:
    - `resolve.ResolveTarget(s, false)` を呼ぶ（`allowAll=false`）
    - 既存のエイリアス `"vscode"` → `"code"` は `TargetAliases` に追加する必要があるが、エディタ名 `"code"` は `KnownTargets` に含まれていない。これについては以下で対処する:
    - エディタ固有のエイリアス（`"vscode"` -> `"code"`）は `pkg/detect/editor.go` 内で `ResolveTarget` 呼び出し前に前処理する。`"code"` はエディタ専用名のため、`ResolveTarget` の対象外とし、従来の `ParseEditor` のロジック（switch文）で直接返す。`ResolveTarget` にフォールスルーするのは `KnownTargets` に含まれる名前のみ。

    具体的な実装:
    ```go
    func ParseEditor(s string) (Editor, error) {
        // Handle editor-only names that are not in KnownTargets
        switch s {
        case "code", "vscode":
            return EditorVSCode, nil
        }
        // Delegate to shared target resolution (all is not allowed for editors)
        resolved, err := resolve.ResolveTarget(s, false)
        if err != nil {
            // Allow custom editor names as before
            return Editor(s), nil
        }
        return Editor(resolved), nil
    }
    ```

    > Note: `"code"` / `"vscode"` はエディタ固有のため、`resolve.ResolveTarget` の `KnownTargets` には含めない。これにより、`--target code` は不正なターゲットとして正しくエラーになる。

---

### features/tt/internal/prompt/（パッケージ移植）

agentctl の `internal/` 配下のパッケージを `features/tt/internal/prompt/` に移植する。import パスの書き換えが主な変更点であり、ロジック自体はそのまま移植する。

#### manifest パッケージ

##### [NEW] [types.go](file://features/tt/internal/prompt/manifest/types.go)
*   **Description**: `agentctl/internal/manifest/types.go` からの移植。
*   **Technical Design**: import パスを `github.com/axsh/tokotachi/features/agentctl/...` から `github.com/axsh/tokotachi/features/tt/internal/prompt/...` に変更。
*   **Logic**: `Entity`, `MemoryDoc`, `ProjectConfig`, `OutputConfig`, `DefaultConfig`, `ValidationError`, `ValidKinds`, `ValidStatuses` をそのまま移植。コメントは英語に統一（INV-008）。

##### [NEW] [parser.go](file://features/tt/internal/prompt/manifest/parser.go)
*   **Description**: `agentctl/internal/manifest/parser.go` からの移植。
*   **Logic**: `ParseAllEntities`, `ExpandGlob`, `parseEntityFile` をそのまま移植。import パス変更のみ。

##### [NEW] [resolver.go](file://features/tt/internal/prompt/manifest/resolver.go)
*   **Description**: `agentctl/internal/manifest/resolver.go` からの移植。
*   **Logic**: `Resolve`, `ResolvedManifest`, `MarshalResolvedManifest` をそのまま移植。

##### [NEW] [validator.go](file://features/tt/internal/prompt/manifest/validator.go)
*   **Description**: `agentctl/internal/manifest/validator.go` からの移植。
*   **Logic**: `NewValidator`, `ValidateSchema`, `ValidateIDUniqueness`, `ValidateReferences` をそのまま移植。

##### テストファイル群
*   [NEW] [types_test.go](file://features/tt/internal/prompt/manifest/types_test.go)
*   [NEW] [parser_test.go](file://features/tt/internal/prompt/manifest/parser_test.go)
*   [NEW] [resolver_test.go](file://features/tt/internal/prompt/manifest/resolver_test.go)
*   [NEW] [validator_test.go](file://features/tt/internal/prompt/manifest/validator_test.go)
*   各テストはそのまま移植し、import パスを更新する。

##### [NEW] testdata/
*   `agentctl/internal/manifest/testdata/` のテストデータをそのまま移植する。

---

#### memory パッケージ

##### [NEW] [frontmatter.go](file://features/tt/internal/prompt/memory/frontmatter.go)
*   **Description**: `agentctl/internal/memory/frontmatter.go` からの移植。
*   **Logic**: `ParseFrontmatter`, `ParseAllMemoryDocs` をそのまま移植。import パス変更のみ。

##### [NEW] [indexer.go](file://features/tt/internal/prompt/memory/indexer.go)
*   **Description**: `agentctl/internal/memory/indexer.go` からの移植。
*   **Logic**: `GenerateIndex` をそのまま移植。

##### テストファイル群
*   [NEW] [frontmatter_test.go](file://features/tt/internal/prompt/memory/frontmatter_test.go)
*   [NEW] [indexer_test.go](file://features/tt/internal/prompt/memory/indexer_test.go)

##### [NEW] testdata/
*   `agentctl/internal/memory/testdata/` のテストデータをそのまま移植する。

---

#### emitter パッケージ

##### [NEW] [emitter.go](file://features/tt/internal/prompt/emitter/emitter.go)
*   **Description**: `Emitter` インターフェース、`EmitMode` 定義の移植。
*   **Logic**:
    ```go
    type EmitMode string
    const (
        EmitModeOverwrite EmitMode = "overwrite"
        EmitModeImmune    EmitMode = "immune"
        EmitModeSkip      EmitMode = "skip"
    )
    type EmitOptions struct {
        Mode   EmitMode
        DryRun bool
    }
    type Emitter interface {
        Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) error
        Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error)
    }
    ```

##### [NEW] [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: `AntigravityEmitter` の移植。
*   **Logic**: そのまま移植。import パス変更のみ。

##### [NEW] [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)
*   **Description**: `CursorEmitter` の移植。

##### [NEW] [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go)
*   **Description**: `ClaudeCodeEmitter` の移植。

##### [NEW] [codex.go](file://features/tt/internal/prompt/emitter/codex.go)
*   **Description**: `CodexEmitter` の移植。

##### その他のユーティリティ
*   [NEW] [emit_mode.go](file://features/tt/internal/prompt/emitter/emit_mode.go) - `writeFileWithMode`, `CleanOrphanFiles`
*   [NEW] [template.go](file://features/tt/internal/prompt/emitter/template.go) - `ResolveTemplateVars`
*   [NEW] [marker.go](file://features/tt/internal/prompt/emitter/marker.go) - `stripFrontmatter`
*   [NEW] [limits.go](file://features/tt/internal/prompt/emitter/limits.go) - `ExtractLimits`, `CheckAndApplyLimit`

##### テストファイル群
*   [NEW] [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go)
*   [NEW] [claude_code_test.go](file://features/tt/internal/prompt/emitter/claude_code_test.go)
*   [NEW] [codex_test.go](file://features/tt/internal/prompt/emitter/codex_test.go)
*   [NEW] [cursor_test.go](file://features/tt/internal/prompt/emitter/cursor_test.go)
*   [NEW] [emit_mode_test.go](file://features/tt/internal/prompt/emitter/emit_mode_test.go)
*   [NEW] [template_test.go](file://features/tt/internal/prompt/emitter/template_test.go)
*   [NEW] [marker_test.go](file://features/tt/internal/prompt/emitter/marker_test.go)
*   [NEW] [limits_test.go](file://features/tt/internal/prompt/emitter/limits_test.go)

---

#### compiler パッケージ

##### [NEW] [config.go](file://features/tt/internal/prompt/compiler/config.go)
*   **Description**: `LoadConfig`, `ResolveProjectRoot` の移植。
*   **Logic**: そのまま移植。import パス変更のみ。コメント英語化。

##### [NEW] [compiler.go](file://features/tt/internal/prompt/compiler/compiler.go)
*   **Description**: `Compile` パイプラインの移植。
*   **Logic**: そのまま移植。ただし `switch opts.Target` 部分は、`"all"` 対応を追加する。`"all"` の場合は全エミッターを順に実行する:
    ```go
    if opts.Target == "all" {
        targets := resolve.KnownTargets
        for _, t := range targets {
            emitObj := newEmitter(t, rootDir)
            if err := emitObj.Emit(resolved, buildDir, apply, emitOpts); err != nil {
                return nil, fmt.Errorf("failed to emit target %s: %w", t, err)
            }
        }
    } else {
        emitObj := newEmitter(opts.Target, rootDir)
        if err := emitObj.Emit(resolved, buildDir, apply, emitOpts); err != nil {
            return nil, fmt.Errorf("failed to emit target %s: %w", opts.Target, err)
        }
    }
    ```

##### [NEW] [digest.go](file://features/tt/internal/prompt/compiler/digest.go)
*   **Description**: `ComputeSourceDigest`, `LoadDigest`, `SaveDigest`, `DigestPath` の移植。
*   **Logic**: そのまま移植。コメントヘッダーを `"tt prompt deploy"` に変更。

##### [NEW] [deploy.go](file://features/tt/internal/prompt/compiler/deploy.go)
*   **Description**: `Deploy` パイプラインの移植。
*   **Logic**: そのまま移植。ただしデフォルトターゲットを `"all"` に変更:
    ```go
    target := opts.Target
    if target == "" {
        target = "all"
    }
    ```

---

### features/tt（go.mod 更新）

#### [MODIFY] [go.mod](file://features/tt/go.mod)
*   **Description**: agentctl で使用している依存を追加。
*   **Technical Design**: 以下を `require` に追加:
    ```
    github.com/santhosh-tekuri/jsonschema/v6 v6.x.x
    github.com/yuin/goldmark v1.x.x
    github.com/yuin/goldmark-meta v1.x.x
    ```
*   **Logic**: `go get` で追加し、`go mod tidy` で整理する。

## Step-by-Step Implementation Guide

### Phase 1: ターゲット名称解決（R1）

1.  **テスト作成**: `pkg/resolve/target_test.go` を作成。テーブル駆動テストで全パターンを定義。
2.  **実装**: `pkg/resolve/target.go` を作成。`ResolveTarget`, `ResolveTargets`, `MetaDir` を実装。
3.  **テスト実行**: ビルドスクリプトで単体テストを確認。
4.  **Git コミット**: `feat: add target name resolution with prefix matching`

### Phase 2: エディタ解決のリファクタリング

5.  **テスト更新**: `pkg/detect/editor_test.go` に `"all"` 拒否ケースを追加。
6.  **実装更新**: `pkg/detect/editor.go` の `ParseEditor` を `resolve.ResolveTarget` を利用する形にリファクタリング。
7.  **テスト実行**: 既存テストが全て通ることを確認。
8.  **Git コミット**: `refactor: integrate editor resolution with shared target resolver`

### Phase 3: パッケージ移植

9.  **go.mod 更新**: `features/tt/go.mod` に依存を追加。
    ```bash
    cd features/tt && go get github.com/santhosh-tekuri/jsonschema/v6 github.com/yuin/goldmark github.com/yuin/goldmark-meta && go mod tidy
    ```
10. **Git コミット**: `chore: add jsonschema, goldmark dependencies to tt module`

11. **manifest パッケージ移植**: `features/tt/internal/prompt/manifest/` を作成し、types.go, parser.go, resolver.go, validator.go とテストファイル、testdata を移植。import パスを書き換える。
12. **テスト実行**: manifest パッケージの単体テストが全て通ることを確認。
13. **Git コミット**: `feat: port manifest package from agentctl to tt`

14. **memory パッケージ移植**: `features/tt/internal/prompt/memory/` を作成し、frontmatter.go, indexer.go とテストファイル、testdata を移植。
15. **テスト実行**: memory パッケージの単体テストが全て通ることを確認。
16. **Git コミット**: `feat: port memory package from agentctl to tt`

17. **emitter パッケージ移植**: `features/tt/internal/prompt/emitter/` を作成し、全ファイルを移植。
18. **テスト実行**: emitter パッケージの単体テストが全て通ることを確認。
19. **Git コミット**: `feat: port emitter package from agentctl to tt`

20. **compiler パッケージ移植**: `features/tt/internal/prompt/compiler/` を作成し、config.go, compiler.go, deploy.go, digest.go を移植。`"all"` ターゲット対応を追加。
21. **テスト実行**: compiler パッケージの単体テストが全て通ることを確認。
22. **Git コミット**: `feat: port compiler package from agentctl to tt with all-target support`

### Phase 4: ビルド検証

23. **全体ビルド + 単体テスト**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests** (共通機能のリグレッション確認):
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```
    *   **Log Verification**: 既存の `tt` コマンド（`up`, `down`, `open` 等）が正常動作することを確認。`pkg/detect/editor.go` のリファクタリングが既存挙動を壊していないことを検証する。

### テスト項目のセルフレビュー

1.  **網羅性の検証**: Part 1 の単体テストは、ターゲット名称解決のロジック（完全一致、エイリアス、前方一致、曖昧エラー、all制御）と、移植した各パッケージの既存テストスイートで構成される。パッケージ移植のテストは元リポジトリから import パスのみ変更して移植するため、元のテスト網羅性を維持する。
2.  **証拠の十分性**: 各テストは期待値の直接比較で検証し、「エラーが出ない」だけの確認は行わない。
3.  **迂回排除**: `ParseEditor` のリファクタリングでは、既存テストケースを全て通過させることで迂回を排除する。
4.  **ボトムアップ順序**: `pkg/resolve/target.go` (末端) -> `pkg/detect/editor.go` (利用側) -> `manifest` -> `memory` -> `emitter` -> `compiler` の順で依存関係に沿って実装・テストする。

## Documentation

#### [MODIFY] [tt-user-manual.md](file://docs/manual/tt-user-manual.md)
*   **更新内容**: `tt prompt` サブコマンドグループの説明を追加（Part 2 の実装完了後に詳細を記載）。本 Part では「Coming Soon」として概要だけ追加する。

## 継続計画について

本計画書は Part 1 である。Part 2 および Part 3 は別ファイルとして同時に作成する:

- **Part 2** ([006-tt-prompt-compile-deploy-part2.md](file://prompts/phases/000-foundation/branches/feat-arch-memory/plans/006-tt-prompt-compile-deploy-part2.md)): cobra コマンド実装（compile, deploy, update）、テンプレート変数拡張
- **Part 3** ([006-tt-prompt-compile-deploy-part3.md](file://prompts/phases/000-foundation/branches/feat-arch-memory/plans/006-tt-prompt-compile-deploy-part3.md)): スクリプト書き換え、スキル簡潔化、プロシージャ追加、ドキュメント更新
