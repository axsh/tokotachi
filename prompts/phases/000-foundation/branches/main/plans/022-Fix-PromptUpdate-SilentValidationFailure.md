# 022-Fix-PromptUpdate-SilentValidationFailure

> **Source Specification**: [020-Fix-PromptUpdate-SilentValidationFailure.md](file:///prompts/phases/000-foundation/branches/main/ideas/020-Fix-PromptUpdate-SilentValidationFailure.md)

## Goal Description

`tt prompt update` がスキーマバリデーションエラーを握りつぶして "Update succeeded" と報告する問題、および `target.schema.json` とカタログ YAML のフィールド名不整合、`targetMetaDirs` のハードコードパスの問題を修正する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: target.schema.json を includes フィールドに対応させる | Proposed Changes > Catalog Schema |
| R2: tt prompt update でバリデーションエラーを報告する | Proposed Changes > Compiler (update.go) |
| R3: targetMetaDirs の antigravity パスを修正 | Proposed Changes > Resolve (target.go) |
| R4: targetMetaDirs の codex パスを修正 | Proposed Changes > Resolve (target.go) |

## Proposed Changes

### Resolve (pkg/resolve)

#### [MODIFY] [target_test.go](file:///pkg/resolve/target_test.go)

*   **Description**: `MetaDir` テストの期待値を新しいパスに更新する。
*   **Technical Design**:
    *   `TestMetaDir` のテーブルデータを修正:
    ```go
    tests := []struct {
        target string
        want   string
    }{
        {"antigravity", ".agent/.meta/antigravity/"},
        {"cursor", ".cursor/.meta/"},
        {"claude-code", ".claude/.meta/"},
        {"codex", ".codex/.meta/"},
    }
    ```
*   **Logic**: TDD に従い、先にテストの期待値を変更し、失敗を確認してから実装を修正する。

---

#### [MODIFY] [target.go](file:///pkg/resolve/target.go)

*   **Description**: `targetMetaDirs` の antigravity パスと codex パスを修正する。
*   **Technical Design**:
    ```go
    var targetMetaDirs = map[string]string{
        "antigravity": ".agent/.meta/antigravity/",
        "cursor":      ".cursor/.meta/",
        "claude-code": ".claude/.meta/",
        "codex":       ".codex/.meta/",
    }
    ```
*   **Logic**:
    *   antigravity: `.agents/.meta/antigravity/` -> `.agent/.meta/antigravity/` (sを除去。エミッターのデフォルト出力先 `.agent/` に合わせる)
    *   codex: `.agents/.meta/codex/` -> `.codex/.meta/` (他ターゲット cursor/claude-code と同じパターンに統一)

---

### Compiler (features/tt/internal/prompt/compiler)

#### [MODIFY] [update_test.go](file:///features/tt/internal/prompt/compiler/update_test.go)

*   **Description**: バリデーションエラー時に `Update` がエラーを返し、メタデータを書かないことを検証するテストケースを追加する。
*   **Technical Design**:
    ```go
    func TestUpdate_ValidationErrors_ReturnsError(t *testing.T) {
        // 不正な target YAML を含む testdata を用意し、
        // Update がエラーを返すことを確認
        tmpDir := t.TempDir()
        copyTestdata(t, filepath.Join("testdata", "invalid_target"), tmpDir)

        // git init (CheckForChanges で必要)
        cmdInit := exec.Command("git", "init")
        cmdInit.Dir = tmpDir
        require.NoError(t, cmdInit.Run())
        cmdConfigName := exec.Command("git", "config", "user.name", "test")
        cmdConfigName.Dir = tmpDir
        _ = cmdConfigName.Run()
        cmdConfigEmail := exec.Command("git", "config", "user.email", "test@test.com")
        cmdConfigEmail.Dir = tmpDir
        _ = cmdConfigEmail.Run()
        cmdAdd := exec.Command("git", "add", ".")
        cmdAdd.Dir = tmpDir
        require.NoError(t, cmdAdd.Run())
        cmdCommit := exec.Command("git", "commit", "-m", "initial")
        cmdCommit.Dir = tmpDir
        require.NoError(t, cmdCommit.Run())

        projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

        _, err := Update(UpdateOptions{
            ProjectPath: projectPath,
            Target:      "antigravity",
            Force:       true,
        })
        require.Error(t, err, "Update should return error when validation fails")
        assert.Contains(t, err.Error(), "validation error")

        // メタデータが書かれていないことを確認
        metaDir := filepath.Join(tmpDir, ".agent", ".meta", "antigravity")
        _, statErr := os.Stat(filepath.Join(metaDir, "last_update.yaml"))
        assert.True(t, os.IsNotExist(statErr),
            "metadata should not be written when validation errors exist")
    }
    ```
*   **Logic**:
    *   テストデータ `testdata/invalid_target/` を新規作成する (後述)
    *   このテストデータにはスキーマバリデーションに失敗するターゲット YAML を含める
    *   `Update` がエラーを返し、メタデータが書き込まれないことを検証

#### [NEW] testdata/catalog_template/

*   **Description**: カタログテンプレート `catalog/originals/axsh/go-standard-project/base/` の `prompts/` ディレクトリ一式をコピーしたテストデータセット。scaffold 後の状態を再現する。
*   **構成**: カタログの `prompts/` ディレクトリをそのままコピー。以下の主要ファイルを含む:
    ```
    testdata/catalog_template/
      prompts/
        manifest/
          project.yaml
          schemas/
            target.schema.json   # 修正後のスキーマ (includes対応)
          targets/
            antigravity.yaml     # includes フィールド使用
            codex.yaml
            claude-code.yaml
            cursor.yaml
          code_content/
            policies/*.md        # 6ファイル
            procedures/*.md      # 8ファイル
            capabilities/*.md    # 5ファイル
            refs/*.md            # 1ファイル
          safety/
            guards/*.yaml
            workers/*.yaml
            bundles/*.yaml
        memory/
          (空 or current.md)
    ```
*   **Logic**: テスト実行時に `copyTestdata` でこのディレクトリ一式を `t.TempDir()` にコピーし、git init -> git commit してから `Update()` を呼ぶ。

---

#### [MODIFY] [update_test.go](file:///features/tt/internal/prompt/compiler/update_test.go) (追加: カタログテンプレートテスト)

*   **Description**: カタログテンプレートを使い、全4ターゲットで `Update` が正常に完了し、コンテンツが正しいパスに配置されることを検証するテストケースを追加する。
*   **Technical Design**:
    ```go
    func TestUpdate_CatalogTemplate_AllTargets(t *testing.T) {
        tmpDir := t.TempDir()
        copyTestdata(t, filepath.Join("testdata", "catalog_template"), tmpDir)

        // git init
        initGitRepo(t, tmpDir)

        projectPath := filepath.Join(tmpDir, "prompts", "manifest", "project.yaml")

        tests := []struct {
            target   string
            rulesDir string
            skillsDir string
            metaDir  string
        }{
            {
                target:    "antigravity",
                rulesDir:  ".agent/rules",
                skillsDir: ".agent/skills",
                metaDir:   ".agent/.meta/antigravity",
            },
            {
                target:    "codex",
                rulesDir:  ".codex/rules",
                skillsDir: ".codex/skills",
                metaDir:   ".codex/.meta",
            },
            {
                target:    "claude-code",
                rulesDir:  ".claude/rules",
                skillsDir: ".claude/skills",
                metaDir:   ".claude/.meta",
            },
            {
                target:    "cursor",
                rulesDir:  ".cursor/rules",
                skillsDir: ".cursor/skills",
                metaDir:   ".cursor/.meta",
            },
        }

        for _, tt := range tests {
            t.Run(tt.target, func(t *testing.T) {
                result, err := Update(UpdateOptions{
                    ProjectPath: projectPath,
                    Target:      tt.target,
                    Force:       true,
                })
                require.NoError(t, err, "Update should succeed for target %s", tt.target)
                require.NotNil(t, result)
                require.Contains(t, result.TargetResults, tt.target)
                assert.False(t, result.TargetResults[tt.target].Skipped)

                // rules ディレクトリにファイルが配置されていることを確認
                rulesPath := filepath.Join(tmpDir, tt.rulesDir)
                entries, err := os.ReadDir(rulesPath)
                require.NoError(t, err,
                    "rules directory %s should exist for target %s", tt.rulesDir, tt.target)
                assert.Greater(t, len(entries), 0,
                    "rules directory should contain files for target %s", tt.target)

                // skills ディレクトリにファイルが配置されていることを確認
                skillsPath := filepath.Join(tmpDir, tt.skillsDir)
                entries, err = os.ReadDir(skillsPath)
                require.NoError(t, err,
                    "skills directory %s should exist for target %s", tt.skillsDir, tt.target)
                assert.Greater(t, len(entries), 0,
                    "skills directory should contain files for target %s", tt.target)

                // メタデータが正しいパスに書かれていることを確認
                metaPath := filepath.Join(tmpDir, tt.metaDir, "last_update.yaml")
                _, err = os.Stat(metaPath)
                require.NoError(t, err,
                    "metadata should exist at %s for target %s", tt.metaDir, tt.target)
            })
        }

        // .agents/ ディレクトリが存在しないことを確認
        agentsPath := filepath.Join(tmpDir, ".agents")
        _, err := os.Stat(agentsPath)
        assert.True(t, os.IsNotExist(err),
            ".agents/ directory should NOT exist (metadata should be in target-specific dirs)")
    }

    // initGitRepo はテスト用に git リポジトリを初期化するヘルパー
    func initGitRepo(t *testing.T, dir string) {
        t.Helper()
        cmds := [][]string{
            {"git", "init"},
            {"git", "config", "user.name", "test"},
            {"git", "config", "user.email", "test@test.com"},
            {"git", "add", "."},
            {"git", "commit", "-m", "initial"},
        }
        for _, args := range cmds {
            cmd := exec.Command(args[0], args[1:]...)
            cmd.Dir = dir
            if err := cmd.Run(); err != nil {
                t.Fatalf("git command %v failed: %v", args, err)
            }
        }
    }
    ```
*   **Logic**:
    *   カタログテンプレートを temp ディレクトリにコピーし、git 初期化
    *   4ターゲット全てで `Update()` を実行
    *   各ターゲットについて: (a) エラーなし (b) rules/ にファイルあり (c) skills/ にファイルあり (d) メタデータが正しいパスに存在
    *   `.agents/` ディレクトリが存在しないことを確認 (全ターゲットのメタデータが `.agents/` ではなく各ターゲット固有のパスに配置される)

---

#### [MODIFY] [update.go](file:///features/tt/internal/prompt/compiler/update.go)

*   **Description**: `Deploy` 結果のバリデーションエラーをチェックし、エラーがある場合はメタデータを書かずにエラーを返す。
*   **Technical Design**:
    *   `Update` 関数内の `Deploy` 呼び出し後、`WriteMetadata` 呼び出し前に以下のチェックを追加:
    ```go
    // Deploy 後にバリデーションエラーをチェック
    if deployResult.CompileResult != nil && len(deployResult.CompileResult.Errors) > 0 {
        for _, e := range deployResult.CompileResult.Errors {
            fmt.Fprintln(os.Stderr, e.Error())
        }
        return nil, fmt.Errorf("deploy failed for target %s: %d validation error(s)",
            t, len(deployResult.CompileResult.Errors))
    }
    ```
*   **Logic**:
    *   現在の L111 (`tr.DeployResult = deployResult`) の後、L113 (`if !opts.DryRun`) の前に挿入
    *   `CompileResult` が nil でないこと、かつ `Errors` スライスが非空であることを確認
    *   エラーの場合: 各エラーを stderr に出力し、`error` を返す
    *   これにより `WriteMetadata` に到達しない
    *   `import "fmt"` が既に存在するため追加不要

---

### Catalog Schema

#### [MODIFY] [target.schema.json](file:///catalog/originals/axsh/go-standard-project/base/prompts/manifest/schemas/target.schema.json)

*   **Description**: `includes` フィールドをスキーマに追加し、`capabilities` を required から除外する。
*   **Technical Design**:
    ```json
    {
      "$schema": "http://json-schema.org/draft-07/schema#",
      "$id": "target.schema.json",
      "title": "Target Definition",
      "type": "object",
      "required": ["apiVersion", "kind", "id", "paths"],
      "properties": {
        "apiVersion": { "type": "string", "const": "agent.meta/v1" },
        "kind": { "type": "string", "const": "target" },
        "id": {
          "type": "string",
          "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$"
        },
        "includes": {
          "type": "object",
          "properties": {
            "policy": { "type": "boolean" },
            "capability": { "type": "boolean" },
            "procedure": { "type": "boolean" },
            "subagent": { "type": "boolean" }
          },
          "additionalProperties": false
        },
        "capabilities": {
          "type": "object",
          "properties": {
            "rules": { "type": "boolean" },
            "skills": { "type": "boolean" },
            "workflows": { "type": "boolean" },
            "subagents": { "type": "boolean" }
          },
          "additionalProperties": false
        },
        "paths": {
          "type": "object",
          "properties": {
            "rules": { "type": "string" },
            "skills": { "type": "string" },
            "workflows": { "type": "string" }
          },
          "additionalProperties": false
        },
        "limits": {
          "type": "object",
          "properties": {
            "rules": { "$ref": "#/$defs/categoryLimit" },
            "skills": { "$ref": "#/$defs/categoryLimit" },
            "workflows": { "$ref": "#/$defs/categoryLimit" }
          },
          "additionalProperties": false
        },
        "index_file": {
          "type": "string",
          "description": "Path to the index file for this target."
        }
      },
      "additionalProperties": false,
      "$defs": {
        "categoryLimit": {
          "type": "object",
          "properties": {
            "max_file_size": {
              "type": "integer",
              "minimum": 1
            },
            "on_exceed": {
              "type": "string"
            }
          },
          "additionalProperties": false
        }
      }
    }
    ```
*   **Logic**:
    *   `required` から `capabilities` を除外 -> `["apiVersion", "kind", "id", "paths"]`
    *   `includes` プロパティを追加 (プロパティ: `policy`, `capability`, `procedure`, `subagent`)
    *   `capabilities` はレガシー互換として残す
    *   `additionalProperties: false` は維持 (不正フィールド検出に有用)

---

## Step-by-Step Implementation Guide

1. **テスト修正 (target_test.go)**: `TestMetaDir` の期待値を `.agent/.meta/antigravity/` と `.codex/.meta/` に変更する。
2. **テスト失敗確認**: `scripts/process/build.sh --skip-frontend --skip-etc` を実行し、`TestMetaDir` が失敗することを確認する。
3. **実装修正 (target.go)**: `targetMetaDirs` の値を修正する。
4. **テスト成功確認**: ビルドスクリプトを再実行し、`TestMetaDir` が成功することを確認する。
5. **テストデータ作成 (testdata/invalid_target/)**: バリデーション失敗するテストデータを作成する。
6. **テスト追加 (update_test.go)**: `TestUpdate_ValidationErrors_ReturnsError` を追加する。
7. **テスト失敗確認**: ビルドスクリプトを実行し、新テストが失敗することを確認する。
8. **実装修正 (update.go)**: `Update` 関数にバリデーションエラーチェックを追加する。
9. **テスト成功確認**: ビルドスクリプトを再実行し、全テストが成功することを確認する。
10. **スキーマ修正 (target.schema.json)**: カタログのスキーマを更新する。
11. **テストデータ作成 (testdata/catalog_template/)**: カタログテンプレートの `prompts/` 一式をコピーし、修正後のスキーマを含むテストデータを作成する。
12. **テスト追加 (update_test.go)**: `TestUpdate_CatalogTemplate_AllTargets` と `initGitRepo` ヘルパーを追加する。
13. **テスト成功確認**: ビルドスクリプトを再実行し、全テストが成功することを確認する。
14. **全体検証**: Verification Plan を実行する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

### E2E Tests

E2E テストは不要。本修正は CLI ツールの内部ロジックとスキーマファイルの修正であり、GUI コンポーネントへの影響はない。既存の単体テストと新規追加テストで検証範囲を十分にカバーできる。

### テスト項目設計

#### ボトムアップの確認順序

依存関係: `CLI (prompt.go)` -> `Update (update.go)` -> `Deploy (deploy.go)` -> `Compile (compiler.go)`
テスト対象は `Update` と `targetMetaDirs` の2箇所。

```
Step 1: targetMetaDirs (末端データ) のテスト -> MetaDir が正しいパスを返すことを確認
Step 2: Update (上位関数) のテスト -> バリデーションエラー時にエラーを返すことを確認
Step 3: Update + Catalog Template (統合) のテスト -> 全ターゲットでコンテンツが正しく配置されることを確認
```

#### テスト項目一覧

| # | テスト関数 | 観点 | 検証内容 |
|---|---|---|---|
| 1 | `TestMetaDir` (修正) | 正常系 | antigravity が `.agent/.meta/antigravity/` を返す |
| 2 | `TestMetaDir` (修正) | 正常系 | codex が `.codex/.meta/` を返す |
| 3 | `TestMetaDir` (既存) | 正常系 | cursor, claude-code は変更なし |
| 4 | `TestMetaDir_Unknown` (既存) | 異常系 | 未知ターゲットで空文字列を返す |
| 5 | `TestUpdate_ValidationErrors_ReturnsError` (新規) | 異常系 | バリデーションエラー時に error を返す |
| 6 | `TestUpdate_ValidationErrors_ReturnsError` (新規) | 副作用 | バリデーションエラー時にメタデータが書かれない |
| 7 | `TestUpdate_Drift` (既存) | 正常系/回帰 | 正常なテストデータでは引き続き動作する |
| 8 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | antigravity: `.agent/rules/` にファイル配置 |
| 9 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | antigravity: `.agent/skills/` にファイル配置 |
| 10 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | antigravity: `.agent/.meta/antigravity/last_update.yaml` が存在 |
| 11 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | codex: `.codex/rules/` にファイル配置 |
| 12 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | codex: `.codex/skills/` にファイル配置 |
| 13 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | codex: `.codex/.meta/last_update.yaml` が存在 |
| 14 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | claude-code: `.claude/rules/` にファイル配置 |
| 15 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | claude-code: `.claude/skills/` にファイル配置 |
| 16 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | claude-code: `.claude/.meta/last_update.yaml` が存在 |
| 17 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | cursor: `.cursor/rules/` にファイル配置 |
| 18 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | cursor: `.cursor/skills/` にファイル配置 |
| 19 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 正常系 | cursor: `.cursor/.meta/last_update.yaml` が存在 |
| 20 | `TestUpdate_CatalogTemplate_AllTargets` (新規) | 副作用 | `.agents/` ディレクトリが存在しない |

#### セルフレビュー結果

1. **網羅性**: R1 はスキーマファイルの修正であり、`TestUpdate_CatalogTemplate_AllTargets` で修正後スキーマを使ったバリデーション通過を間接的に検証。R2 は `TestUpdate_ValidationErrors_ReturnsError` でカバー。R3/R4 は `TestMetaDir` でカバーし、`TestUpdate_CatalogTemplate_AllTargets` でメタデータパスの実動作を検証。全要件に対応するテストが存在する。
2. **証拠の十分性**: 各ターゲットについて (a) rules にファイルがある (b) skills にファイルがある (c) メタデータが正しいパスにある の3点を確認。さらに `.agents/` ディレクトリが存在しないことで副作用の不在を確認。
3. **迂回・抜け道の排除**: `TestUpdate_CatalogTemplate_AllTargets` はカタログテンプレートの実データを使うため、スキーマ修正 (R1) が正しく行われていなければバリデーションエラーでテストが失敗する。フォールバックによる偽成功の余地はない。
4. **依存関係の整合性**: `TestMetaDir` (末端) -> `TestUpdate_ValidationErrors_ReturnsError` (異常系) -> `TestUpdate_CatalogTemplate_AllTargets` (統合正常系) の順で、ボトムアップに確認が積み上がっている。

## Documentation

本修正に伴い更新が必要な仕様書やドキュメントはない。

