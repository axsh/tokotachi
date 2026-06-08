# 000-Unified-Agents-Deploy

> **Source Specification**: prompts/phases/000-foundation/branches/fix-agy-agent-files/ideas/000-Unified-Agents-Deploy.md

## Goal Description

Antigravity と Codex のデプロイ先を `.agents/` に統合する。immune モードで `--target all` を指定した際に、一方のターゲットのファイルが他方の orphan クリーンアップで削除されないよう、Deploy パイプラインの処理順序を変更する。

## User Review Required

> [!IMPORTANT]
> **Emitter インターフェースの戻り値変更**: `Emit()` メソッドの戻り値を `error` から `(*EmitResult, error)` に変更する。これにより全 Emitter（Antigravity, Codex, Cursor, ClaudeCode）および全テストファイルに影響がある。

> [!IMPORTANT]
> **CLIフラグの追加**: `tt prompt deploy` コマンドに `--mode` フラグを追加する。これにより既存の deploy シェルスクリプトを `--mode` 付きで呼び出せるようになる。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: デプロイ先統合（Agy を .agents/ に） | Proposed Changes > Config > antigravity.yaml, target.go |
| R2: immune モードの安全な処理順序 | Proposed Changes > Deploy Pipeline > deploy.go |
| R3: ユーザー追加ファイルの適切な扱い | R2 の実装で自動的に満たされる |
| R4: 単独ターゲット指定時の動作 | deploy.go の共有ディレクトリ検出ロジック |
| R5: Emitter の分離維持 | 変更なし（Emitter クラスは維持） |
| R6: AGENTS.md のインデックス統合（任意） | 本計画では対象外。別途仕様策定が必要 |

## Proposed Changes

### Emitter インターフェース

#### [MODIFY] [emitter.go](file://features/tt/internal/prompt/emitter/emitter.go)
*   **Description**: `Emit()` の戻り値を拡張し、emittedFiles を返すようにする。immune モードの orphan クリーンアップを Emitter 内部から Deploy パイプラインに移動するための基盤。
*   **Technical Design**:
    ```go
    // EmitResult holds the result of an Emit operation.
    type EmitResult struct {
        // EmittedFiles maps absolute file paths to true for all files written during emit.
        EmittedFiles map[string]bool
        // TargetDirs lists the target directories that were written to.
        TargetDirs []string
    }

    type Emitter interface {
        // Emit generates target-specific files into buildDir or project paths.
        // Returns EmitResult with the list of emitted files for orphan cleanup coordination.
        Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) (*EmitResult, error)
        // Check verifies if generated files in project paths match the resolved manifest.
        Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error)
    }
    ```
*   **Logic**:
    *   `EmitResult` 構造体を新設。`EmittedFiles`（emitされたファイルの絶対パスマップ）と `TargetDirs`（対象ディレクトリ一覧）を保持する。
    *   各 Emitter の `Emit()` メソッドが `EmitResult` を返すようにする。
    *   **immune モードの `CleanOrphanFiles` 呼び出しは各 Emitter の `Emit()` 内部から削除する。** 代わりに Deploy パイプライン側で統合的に処理する。

---

### Antigravity Emitter

#### [MODIFY] [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go)
*   **Description**: デフォルトパスを `.agents/` に変更。`Emit()` の戻り値を `(*EmitResult, error)` に変更。immune の orphan クリーンアップを削除。
*   **Technical Design**:
    ```go
    func (a *AntigravityEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string, string) {
        // Default paths changed from .agent/ to .agents/
        rulesPath := ".agents/rules/"
        skillsPath := ".agents/skills/"
        workflowsPath := ".agents/workflows/"
        // ... rest unchanged
    }

    func (a *AntigravityEmitter) resolveTargetPaths(resolved *manifest.ResolvedManifest) TargetPaths {
        tp := TargetPaths{
            Rules:     ".agents/rules/",
            Skills:    ".agents/skills/",
            Workflows: ".agents/workflows/",
        }
        // ... rest unchanged
    }

    func (a *AntigravityEmitter) Emit(...) (*EmitResult, error) {
        // ... existing emit logic ...
        // REMOVE: immune mode CleanOrphanFiles block (L244-L249)
        return &EmitResult{
            EmittedFiles: emittedFiles,
            TargetDirs:   []string{rulesDir, skillsDir, workflowsDir},
        }, nil
    }
    ```
*   **Logic**:
    *   L35-L37: `.agent/rules/`, `.agent/skills/`, `.agent/workflows/` を `.agents/rules/`, `.agents/skills/`, `.agents/workflows/` に変更。
    *   L256-L259: `resolveTargetPaths` のデフォルト値も同様に変更。
    *   L67: `Emit()` の戻り値を `(*EmitResult, error)` に変更。
    *   L244-L249: immune モードの `CleanOrphanFiles` ブロックを削除。
    *   L250付近: `return &EmitResult{EmittedFiles: emittedFiles, TargetDirs: []string{rulesDir, skillsDir, workflowsDir}}, nil` を返す。
    *   L292: `Check()` 内の `Emit()` 呼び出しも戻り値変更に対応。

---

### Codex Emitter

#### [MODIFY] [codex.go](file://features/tt/internal/prompt/emitter/codex.go)
*   **Description**: `Emit()` の戻り値を `(*EmitResult, error)` に変更。immune の orphan クリーンアップを削除。
*   **Technical Design**:
    ```go
    func (c *CodexEmitter) Emit(...) (*EmitResult, error) {
        // ... existing emit logic ...
        // REMOVE: immune mode CleanOrphanFiles block (L232-L238)
        return &EmitResult{
            EmittedFiles: emittedFiles,
            TargetDirs:   []string{rulesDir, skillsDir},
        }, nil
    }
    ```
*   **Logic**:
    *   L54: `Emit()` の戻り値を `(*EmitResult, error)` に変更。
    *   L232-L238: immune モードの `CleanOrphanFiles` ブロックを削除。
    *   L240付近: `EmitResult` を返す。
    *   L299: `Check()` 内の `Emit()` 呼び出しも戻り値変更に対応。

---

### Cursor Emitter

#### [MODIFY] [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go)
*   **Description**: `Emit()` の戻り値を `(*EmitResult, error)` に変更。immune の orphan クリーンアップを削除。
*   **Logic**:
    *   Cursor は `.cursor/` を使用しており `.agents/` とは無関係だが、インターフェース変更のために戻り値を合わせる。
    *   immune 処理を Emit 内部から削除し、`EmitResult` を返す。

---

### Claude Code Emitter

#### [MODIFY] [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go)
*   **Description**: `Emit()` の戻り値を `(*EmitResult, error)` に変更。immune の orphan クリーンアップを削除。
*   **Logic**: Cursor と同様。`.claude/` を使用しており `.agents/` とは無関係だが、インターフェース変更のために戻り値を合わせる。

---

### Deploy パイプライン

#### [MODIFY] [deploy.go](file://features/tt/internal/prompt/compiler/deploy.go)
*   **Description**: `DeployOptions` に `Mode` を追加（既存）。`Deploy()` 関数内で `Emit()` の戻り値から `EmitResult` を受け取り、immune モードの場合は orphan クリーンアップをここで実行する。
*   **Technical Design**:
    ```go
    // DeployResult holds the output of the deploy pipeline
    type DeployResult struct {
        Skipped       bool
        DigestCurrent string
        DigestPrev    string
        CompileResult *CompileResult
        Warnings      []string
        EmitResult    *emitter.EmitResult  // NEW: emitted files info
    }
    ```
*   **Logic**:
    *   `Compile()` の戻り値に含まれる `EmitResult` を `DeployResult.EmitResult` に格納する。
    *   `Compile()` が `Emit()` を呼んでいるため、Compile の戻り値にも `EmitResult` を追加する必要がある。
    *   **immune モードの orphan クリーンアップ**: `Deploy()` 関数自体では実行しない。呼び出し元の `runPromptDeploy` で統合的に処理する。

#### [MODIFY] [compiler.go](file://features/tt/internal/prompt/compiler/compiler.go)
*   **Description**: `Compile()` 内の `Emit()` 呼び出しの戻り値を処理し、`CompileResult` に `EmitResult` を含める。
*   **Technical Design**:
    ```go
    type CompileResult struct {
        IndexContent string
        ResolvedYAML string
        Resolved     *manifest.ResolvedManifest
        Errors       []manifest.ValidationError
        EmitResult   *emitter.EmitResult  // NEW
    }
    ```
*   **Logic**:
    *   L144: `emitObj.Emit()` の戻り値を受け取り、`result.EmitResult` に格納する。
    *   immune モードの `CleanOrphanFiles` は Emitter 内部から削除済みのため、ここでも呼ばない。
    *   `CompileResult.EmitResult` は `Deploy()` → `runPromptDeploy` まで伝搬させる。

---

### CLI コマンド

#### [MODIFY] [prompt.go](file://features/tt/cmd/prompt.go)
*   **Description**: `--mode` フラグを追加。immune モードでの共有ディレクトリターゲットの統合 orphan クリーンアップを実装。
*   **Technical Design**:
    ```go
    var (
        deployProject string
        deployTarget  string
        deployForce   bool
        deployDryRun  bool
        deployMode    string  // NEW
    )

    // init() に追加:
    promptDeployCmd.Flags().StringVar(&deployMode, "mode",
        "", "Emit mode: overwrite (default), skip, immune")
    ```
*   **Logic**:
    *   `runPromptDeploy` を以下のように変更:

    ```go
    func runPromptDeploy(cmd *cobra.Command, args []string) error {
        target := resolveTargetFlag(deployTarget)
        // ... resolve targets ...

        mode := emitter.EmitMode(deployMode)
        if mode != "" && !emitter.ValidEmitModes(mode) {
            return fmt.Errorf("invalid mode %q: must be overwrite, skip, or immune", deployMode)
        }

        // Collect all deploy results
        type deployEntry struct {
            target string
            result *compiler.DeployResult
        }
        var entries []deployEntry

        for _, t := range targets {
            result, err := compiler.Deploy(compiler.DeployOptions{
                ProjectPath: deployProject,
                Target:      t,
                Force:       deployForce,
                DryRun:      deployDryRun,
                Mode:        mode,
            })
            if err != nil {
                return fmt.Errorf("deploy failed for target %s: %w", t, err)
            }
            entries = append(entries, deployEntry{target: t, result: result})
            // ... print status ...
        }

        // Immune mode: unified orphan cleanup
        if mode == emitter.EmitModeImmune && !deployDryRun {
            // Merge emittedFiles from all targets that share directories
            mergedEmitted := make(map[string]bool)
            var allTargetDirs []string
            for _, e := range entries {
                if e.result.EmitResult != nil {
                    for k, v := range e.result.EmitResult.EmittedFiles {
                        mergedEmitted[k] = v
                    }
                    allTargetDirs = append(allTargetDirs, e.result.EmitResult.TargetDirs...)
                }
            }
            // Deduplicate target dirs
            uniqueDirs := deduplicateDirs(allTargetDirs)
            if _, err := emitter.CleanOrphanFiles(uniqueDirs, mergedEmitted, false); err != nil {
                return fmt.Errorf("immune orphan cleanup failed: %w", err)
            }
        }

        return nil
    }
    ```

    *   `deduplicateDirs` ヘルパー関数を追加:
    ```go
    func deduplicateDirs(dirs []string) []string {
        seen := make(map[string]bool)
        var result []string
        for _, d := range dirs {
            clean := filepath.Clean(d)
            if !seen[clean] {
                seen[clean] = true
                result = append(result, clean)
            }
        }
        return result
    }
    ```

---

### ターゲット設定

#### [MODIFY] [antigravity.yaml](file://prompts/manifest/targets/antigravity.yaml)
*   **Description**: Antigravity のデプロイ先パスを `.agents/` に変更。
*   **Technical Design**:
    ```yaml
    apiVersion: agent.meta/v1
    kind: target
    id: antigravity
    capabilities:
      rules: true
      skills: true
      workflows: true
      subagents: false
    paths:
      rules: .agents/rules/
      skills: .agents/skills/
      workflows: .agents/workflows/
    limits:
      rules:
        max_file_size: 12000
        on_exceed: warn
    ```

#### [MODIFY] [target.go](file://pkg/resolve/target.go)
*   **Description**: `targetMetaDirs` の antigravity エントリを `.agents/.meta/antigravity/` に変更。codex も `.agents/.meta/codex/` に変更し、メタデータディレクトリの衝突を回避する。
*   **Technical Design**:
    ```go
    var targetMetaDirs = map[string]string{
        "antigravity": ".agents/.meta/antigravity/",
        "cursor":      ".cursor/.meta/",
        "claude-code": ".claude/.meta/",
        "codex":       ".agents/.meta/codex/",
    }
    ```

---

### テストファイル

#### [MODIFY] [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go)
*   **Description**: `Emit()` の戻り値変更に合わせてテストを修正。全ての `Emit()` 呼び出しで `(*EmitResult, error)` を受け取るように変更。
*   **Logic**:
    *   既存の `emitter.Emit(...)` 呼び出し箇所（L129, L175, L217, L287, L340, L393）で戻り値を `result, err` に変更。
    *   `result.EmittedFiles` が正しく設定されていることを検証するテストケースを追加。

#### [MODIFY] [codex_test.go](file://features/tt/internal/prompt/emitter/codex_test.go)
*   **Description**: `Emit()` の戻り値変更に合わせてテストを修正。
*   **Logic**: 全ての `e.Emit(...)` 呼び出し箇所で戻り値を `result, err` に変更。

#### [MODIFY] [cursor_test.go](file://features/tt/internal/prompt/emitter/cursor_test.go)
*   **Description**: `Emit()` の戻り値変更に合わせてテストを修正。
*   **Logic**: 全ての `emitter.Emit(...)` 呼び出し箇所で戻り値を `result, err` に変更。

#### [MODIFY] [claude_code_test.go](file://features/tt/internal/prompt/emitter/claude_code_test.go)
*   **Description**: `Emit()` の戻り値変更に合わせてテストを修正。
*   **Logic**: 全ての `e.Emit(...)` 呼び出し箇所で戻り値を `result, err` に変更。

#### [MODIFY] [deploy_test.go](file://features/tt/internal/prompt/compiler/deploy_test.go)
*   **Description**: `DeployResult.EmitResult` フィールドの検証を追加。
*   **Logic**: 既存テストで `result.EmitResult` が nil でないことを検証。

#### [MODIFY] [target_test.go](file://pkg/resolve/target_test.go)
*   **Description**: `MetaDir` のテストケースを更新。antigravity と codex のメタディレクトリパスが変更されたことを反映。

#### [NEW] [deploy_immune_test.go](file://features/tt/internal/prompt/compiler/deploy_immune_test.go)
*   **Description**: immune モードでの統合 orphan クリーンアップの単体テスト。
*   **Technical Design**:
    ```go
    func TestDeploy_ImmuneMode_SharedDirectory(t *testing.T) {
        // Setup: create a temp project with both antigravity and codex targets
        // 1. Deploy antigravity with immune mode
        // 2. Deploy codex with immune mode
        // 3. Verify both targets' files exist (not deleted by each other)
    }

    func TestDeploy_ImmuneMode_OrphanRemoval(t *testing.T) {
        // Setup: create a temp project, deploy, then add orphan file
        // 1. Deploy with immune mode
        // 2. Verify orphan file is removed
        // 3. Verify emitted files still exist
    }
    ```

---

### ドキュメント

#### [MODIFY] [AGENTS.md](file://AGENTS.md)
*   **Description**: re-deploy により自動更新される。`--mode` フラグの追加を反映するには、マーカーセクションの更新は自動で行われる。

## Step-by-Step Implementation Guide

### Phase 1: Emitter インターフェース変更

1.  **EmitResult 型の追加**:
    *   [emitter.go](file://features/tt/internal/prompt/emitter/emitter.go) に `EmitResult` 構造体を追加。
    *   `Emitter` インターフェースの `Emit()` 戻り値を `(*EmitResult, error)` に変更。

2.  **Antigravity Emitter の修正**:
    *   [antigravity.go](file://features/tt/internal/prompt/emitter/antigravity.go) のデフォルトパスを `.agents/` に変更（`resolvePaths`, `resolveTargetPaths`）。
    *   `Emit()` の戻り値を変更し、immune の `CleanOrphanFiles` ブロックを削除。`EmitResult` を返す。
    *   `Check()` 内の `Emit()` 呼び出しの戻り値を `_, err` で受ける。

3.  **Codex Emitter の修正**:
    *   [codex.go](file://features/tt/internal/prompt/emitter/codex.go) の `Emit()` 戻り値を変更。immune ブロック削除。`EmitResult` を返す。
    *   `Check()` 内の `Emit()` 呼び出しの戻り値を `_, err` で受ける。

4.  **Cursor Emitter の修正**:
    *   [cursor.go](file://features/tt/internal/prompt/emitter/cursor.go) の `Emit()` 戻り値を変更。immune ブロック削除。`EmitResult` を返す。
    *   `Check()` 内の `Emit()` 呼び出しの戻り値を `_, err` で受ける。

5.  **Claude Code Emitter の修正**:
    *   [claude_code.go](file://features/tt/internal/prompt/emitter/claude_code.go) の `Emit()` 戻り値を変更。immune ブロック削除。`EmitResult` を返す。
    *   `Check()` 内の `Emit()` 呼び出しの戻り値を `_, err` で受ける。

6.  **全 Emitter テストの修正**:
    *   [emitter_test.go](file://features/tt/internal/prompt/emitter/emitter_test.go), [codex_test.go](file://features/tt/internal/prompt/emitter/codex_test.go), [cursor_test.go](file://features/tt/internal/prompt/emitter/cursor_test.go), [claude_code_test.go](file://features/tt/internal/prompt/emitter/claude_code_test.go) の全 `Emit()` 呼び出し箇所で戻り値を `result, err` に変更。

### Phase 2: Compile パイプライン修正

7.  **CompileResult 拡張**:
    *   [compiler.go](file://features/tt/internal/prompt/compiler/compiler.go) の `CompileResult` に `EmitResult *emitter.EmitResult` フィールドを追加。
    *   L144 の `emitObj.Emit()` 呼び出しの戻り値を受け取り、`result.EmitResult` に格納。

8.  **DeployResult 拡張**:
    *   [deploy.go](file://features/tt/internal/prompt/compiler/deploy.go) の `DeployResult` に `EmitResult *emitter.EmitResult` フィールドを追加。
    *   `result.EmitResult = compileResult.EmitResult` で伝搬。

### Phase 3: CLI 統合

9.  **--mode フラグの追加**:
    *   [prompt.go](file://features/tt/cmd/prompt.go) に `deployMode` 変数と `--mode` フラグを追加。
    *   `DeployOptions` への `Mode` の受け渡しを実装。

10. **immune orphan クリーンアップの統合**:
    *   `runPromptDeploy` で全ターゲットの deploy 完了後、immune モードの場合に統合 orphan クリーンアップを実行するロジックを追加。
    *   `deduplicateDirs` ヘルパー関数を追加。

### Phase 4: ターゲット設定変更

11. **antigravity.yaml の更新**:
    *   [antigravity.yaml](file://prompts/manifest/targets/antigravity.yaml) の paths を `.agents/` に変更。

12. **target.go のメタディレクトリ変更**:
    *   [target.go](file://pkg/resolve/target.go) の `targetMetaDirs` で antigravity を `.agents/.meta/antigravity/`、codex を `.agents/.meta/codex/` に変更。

### Phase 5: テスト追加・修正

13. **deploy_test.go の更新**:
    *   [deploy_test.go](file://features/tt/internal/prompt/compiler/deploy_test.go) で `EmitResult` フィールドの検証を追加。

14. **target_test.go の更新**:
    *   [target_test.go](file://pkg/resolve/target_test.go) の `MetaDir` テストケースを更新。

15. **deploy_immune_test.go の新規作成**:
    *   [deploy_immune_test.go](file://features/tt/internal/prompt/compiler/deploy_immune_test.go) を新規作成。immune モードでの統合 orphan クリーンアップテストを追加。

### Phase 6: 検証

16. **ビルド + 単体テスト実行**
17. **必要に応じてバックエンド統合テスト実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests** (prompt deploy 関連):
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```

### テスト項目設計のセルフレビュー

#### 観点チェックリスト

| # | 観点 | 確認内容 | 対応テスト |
|---|------|----------|-----------|
| 1 | 正常系の動作確認 | overwrite/skip/immune 各モードで正しくファイルが書き込まれるか | 既存 emit_mode_test.go + deploy_immune_test.go |
| 2 | 異常系・境界値 | 不正な mode 値が拒否されるか | prompt.go の mode バリデーション |
| 3 | 外部連携の実動作 | ファイルシステムへの書き込み/削除が正しく動作するか | deploy_test.go, deploy_immune_test.go |
| 4 | データの一貫性 | EmitResult が全ターゲットで正しく返されるか | emitter_test.go 系の戻り値検証 |
| 5 | 状態遷移の検証 | immune 後にファイルが正しい状態になっているか | deploy_immune_test.go |
| 6 | 設定・構成の反映 | antigravity.yaml の paths 変更が Emitter に反映されるか | emitter_test.go |
| 7 | 副作用の確認 | orphan クリーンアップで意図しないファイルが削除されないか | deploy_immune_test.go |

#### ボトムアップ確認順序

1. `EmitResult` 構造体 -> 各 Emitter の `Emit()` が正しく `EmitResult` を返すか (emitter_test.go 系)
2. `CompileResult.EmitResult` 伝搬 -> Compile 経由で EmitResult が取得できるか (compiler_test.go)
3. `DeployResult.EmitResult` 伝搬 -> Deploy 経由で EmitResult が取得できるか (deploy_test.go)
4. immune 統合クリーンアップ -> 共有ディレクトリで orphan が正しく処理されるか (deploy_immune_test.go)

#### 網羅性のセルフレビュー

1. **網羅性**: immune モードの統合クリーンアップが正しく動作し、かつ overwrite/skip モードに影響がないことを確認 -> 充分
2. **証拠の十分性**: 各テストはファイルの存在/不在を直接検証 -> 充分
3. **迂回排除**: EmitResult を経由せずに CleanOrphanFiles が呼ばれるパスがないことをコードレベルで確認 -> 充分

## Documentation

#### [MODIFY] [docs/manual/tt-user-manual.md](file://docs/manual/tt-user-manual.md)
*   **更新内容**: `tt prompt deploy` に `--mode` フラグが追加されたことを記載。overwrite/skip/immune の各モードの動作説明を追加。
