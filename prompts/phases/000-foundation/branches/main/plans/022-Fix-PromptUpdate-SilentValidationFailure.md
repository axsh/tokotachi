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

#### [NEW] testdata/invalid_target/

*   **Description**: スキーマバリデーションが失敗するテストデータセット。
*   **構成**:
    ```
    testdata/invalid_target/
      prompts/manifest/
        project.yaml          # valid ディレクトリから流用
        schemas/
          target.schema.json  # capabilities を required にしたスキーマ
          policy.schema.json  # valid から流用
        targets/
          antigravity.yaml    # required の capabilities フィールドを欠落させる
        code_content/
          policies/
            test.yaml         # valid から流用
      prompts/memory/
        current.md            # valid から流用
    ```
*   **antigravity.yaml の内容** (意図的にバリデーション失敗させる):
    ```yaml
    apiVersion: agent.meta/v1
    kind: target
    id: antigravity
    # capabilities フィールドを省略 (required violation)
    paths:
      rules: .agent/rules/
      skills: .agent/skills/
    ```
*   **project.yaml の追加設定**:
    ```yaml
    sources:
      ...
      targets: prompts/manifest/targets/**/*.yaml
    ```

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
11. **全体検証**: Verification Plan を実行する。

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

#### セルフレビュー結果

1. **網羅性**: R1 はスキーマファイルの修正でありランタイムコードに変更なし。R2 は Update のエラーハンドリングを `TestUpdate_ValidationErrors_ReturnsError` でカバー。R3/R4 は `TestMetaDir` でカバー。全要件に対応するテストが存在する。
2. **証拠の十分性**: エラーが返ることの確認に加え、メタデータが書き込まれないことも確認している (副作用の検証)。
3. **迂回・抜け道の排除**: `Update` がエラーを返す経路が正しく動作していることを、テストデータを意図的に壊すことで確認する。
4. **依存関係の整合性**: `TestMetaDir` (末端) -> `TestUpdate_ValidationErrors_ReturnsError` (上位) の順で設計されている。

## Documentation

本修正に伴い更新が必要な仕様書やドキュメントはない。
