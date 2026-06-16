# 006-Remove-Memory-Index

> **Source Specification**: prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/006-Remove-Memory-Index.md

## Goal Description

旧メモリシステムの遺物である `prompts/memory/index.md` とその生成基盤を完全に削除する。関連する Go コード、設定、テストデータ、ワークフロー参照を全て除去し、compile/deploy パイプラインから index 生成ロジックを取り除く。

## User Review Required

- 「メモリの確認」ステップ (execute-implementation-plan, create-specification) は代替なしで完全削除する。ユーザー承認済み。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. ファイル削除 index.md | Step 1: prompts/memory/index.md 削除 |
| 2. Go コード削除 indexer.go | Step 2: memory/indexer.go, indexer_test.go 削除 |
| 2. compiler.go index 生成除去 | Step 3: compiler.go 修正 |
| 2. CompileResult.IndexContent 削除 | Step 3: compiler.go 修正 |
| 2. deploy.go コメント修正 | Step 4: deploy.go 修正 |
| 2. compiler_test.go 修正 | Step 5: compiler_test.go 修正 |
| 3. project.yaml memory_index 削除 | Step 6: project.yaml 修正 |
| 4. OutputConfig.MemoryIndex 削除 | Step 6: types.go 修正 |
| 5. template.go memory kind 削除 | Step 7: template.go 修正 |
| 6. template_test.go 修正 | Step 7: template_test.go 修正 |
| 7. ガード削除 | Step 8: deny-direct-edit-of-index.yaml 削除 |
| 8. ワークフロー参照除去 (6ファイル) | Step 9: procedures, policy, capability 修正 |
| 9. prompt update 確認 | Step 10: デプロイ確認 |

## Proposed Changes

### Go: memory パッケージ

#### [DELETE] [indexer.go](file:///features/tt/internal/prompt/memory/indexer.go)
*   **Description**: GenerateIndex 関数とヘルパー関数 (computeDigest, filterByStatus, sortByPriority, toRelativePath) を全て削除

#### [DELETE] [indexer_test.go](file:///features/tt/internal/prompt/memory/indexer_test.go)
*   **Description**: indexer.go に対応するテスト全体を削除

---

### Go: compiler パッケージ

#### [MODIFY] [compiler.go](file:///features/tt/internal/prompt/compiler/compiler.go)
*   **Description**: index.md 生成ロジックを除去
*   **Technical Design**:
    *   `CompileResult` 構造体から `IndexContent string` フィールドを削除
    *   `import` から `"github.com/axsh/tokotachi/features/tt/internal/prompt/memory"` を削除 (frontmatter.go の参照が残る場合は残す。ParseAllMemoryDocs の呼び出しを残す場合、import は残す)
    *   Step 10 (GenerateIndex) を削除: L92-97 の `indexContent, err := memory.GenerateIndex(memDocs)` と `result.IndexContent = indexContent` を削除
    *   Step 12 の index.md 書き出し部分を削除: L108-112 の `indexPath` と `writeFile(indexPath, ...)` を削除
    *   Step 番号をリナンバリング
*   **Logic**:
    *   `memory.GenerateIndex()` 呼び出しを完全に削除
    *   `cfg.Outputs.MemoryIndex` への書き込みを削除
    *   memDocs の走査 (Step 4) と ID ユニーク検証 (Step 6-7) は残す (index とは独立した機能)

#### [MODIFY] [compiler_test.go](file:///features/tt/internal/prompt/compiler/compiler_test.go)
*   **Description**: IndexContent 関連アサーションを除去
*   **Technical Design**:
    *   `TestCompile_Valid`: L25-27 (`result.IndexContent` チェック)、L32-38 (IndexContent 内容検証) を削除
    *   `TestCompile_DryRun`: L54-58 (`indexPath` 存在チェック)、L61-63 (`result.IndexContent` チェック) を削除
    *   `TestCompile_WriteFiles`: L83-91 (index.md 書き込み検証) を削除
    *   `TestCompile_WithValidationErrors`: L117-119 (`result.IndexContent` が空であるチェック) を削除

#### [MODIFY] [deploy.go](file:///features/tt/internal/prompt/compiler/deploy.go)
*   **Description**: L96-97 のコメントから index.md 言及を除去
*   **Technical Design**:
    *   L96: `// Recompute digest after compile because compile may generate files` を更新
    *   L97: `// (e.g. index.md) into source directories, changing the effective digest.` の index.md 言及を除去

---

### Go: manifest パッケージ

#### [MODIFY] [types.go](file:///features/tt/internal/prompt/manifest/types.go)
*   **Description**: OutputConfig から MemoryIndex フィールドを削除
*   **Technical Design**:
    ```go
    type OutputConfig struct {
        ResolvedManifest string `yaml:"resolved_manifest"`
        // MemoryIndex を削除
    }
    ```

---

### Go: emitter パッケージ

#### [MODIFY] [template.go](file:///features/tt/internal/prompt/emitter/template.go)
*   **Description**: `memory` kind の解決ロジックを削除
*   **Technical Design**:
    *   `resolveRef()` 内の L58-59 `case "memory": return ctx.MemBase + "/" + id + ".md"` を削除
    *   `TemplateContext.MemBase` フィールドは他の用途で使われている可能性があるため、呼び出し元を確認して不要なら削除

#### [MODIFY] [template_test.go](file:///features/tt/internal/prompt/emitter/template_test.go)
*   **Description**: `{{memory:index}}` テストケースを削除
*   **Technical Design**:
    *   L43-47 の "memory index reference" テストケースを削除

---

### テストデータ

#### [MODIFY] testdata/valid/prompts/manifest/project.yaml
*   `memory_index` 行を削除

#### [MODIFY] testdata/invalid/prompts/manifest/project.yaml
*   `memory_index` 行を削除

#### [MODIFY] [config_test.go](file:///features/tt/internal/prompt/compiler/config_test.go)
*   L18 のテストデータ YAML 文字列から `memory_index` 部分を削除

---

### 設定ファイル

#### [MODIFY] [project.yaml](file:///prompts/manifest/project.yaml)
*   **Description**: `memory_index` 出力設定を削除
*   **Technical Design**:
    ```yaml
    outputs:
      resolved_manifest: tmp/dist/manifest.resolved.yaml
      # memory_index 行を削除
    ```

---

### 削除対象ファイル

#### [DELETE] [index.md](file:///prompts/memory/index.md)
*   旧メモリシステムのインデックスファイル

#### [DELETE] [deny-direct-edit-of-index.yaml](file:///prompts/manifest/safety/guards/deny-direct-edit-of-index.yaml)
*   index.md の直接編集を禁止するガード

---

### ワークフロー / ポリシー / Capability

#### [MODIFY] [execute-implementation-plan.md](file:///prompts/manifest/code_content/procedures/execute-implementation-plan.md)
*   **Description**: Section 1.3「メモリの確認」ステップ (L25-27) を完全削除
*   **Logic**: `{{memory:index}}` と `{{memory:invariants}}` への参照を含むステップ全体を削除。番号をリナンバリング不要 (3. が消えて 2. の次が Section 2 になる)

#### [MODIFY] [create-specification.md](file:///prompts/manifest/code_content/procedures/create-specification.md)
*   **Description**: Section 1.2「メモリの確認」ステップ (L23-25) を完全削除
*   **Logic**: `{{memory:index}}` と `{{memory:decisions}}`, `{{memory:open-questions}}` への参照を含むステップ全体を削除

#### [MODIFY] [far-knowledge-memory.md](file:///prompts/manifest/code_content/policies/far-knowledge-memory.md)
*   **Description**: L12-14 の `prompts/memory/index.md` と `inbox.md` への言及を削除
*   **Logic**: 
    *   L12: `Before changing architecture-sensitive code, read prompts/memory/index.md.` を削除
    *   L13: `After such changes, update the relevant memory document.` を削除
    *   L14: `If unsure where to write, append to prompts/memory/inbox.md.` を削除
    *   これらは存在しないファイルへの参照であるため完全除去

#### [MODIFY] [record-far-knowledge.md](file:///prompts/manifest/code_content/capabilities/record-far-knowledge.md)
*   **Description**: references 欄から `prompts/memory/index.md` を削除 (L12)
*   **Technical Design**:
    ```yaml
    references:
      # "prompts/memory/index.md" を削除
      - "prompts/memory/schemas/agent-record-payload.schema.json"
    ```

## Step-by-Step Implementation Guide

1. **Step 1: ファイル削除**
   * `prompts/memory/index.md` を削除
   * `prompts/manifest/safety/guards/deny-direct-edit-of-index.yaml` を削除
   * `features/tt/internal/prompt/memory/indexer.go` を削除
   * `features/tt/internal/prompt/memory/indexer_test.go` を削除

2. **Step 2: manifest types.go 修正**
   * `OutputConfig` から `MemoryIndex` フィールドを削除

3. **Step 3: compiler.go 修正**
   * `CompileResult.IndexContent` フィールドを削除
   * Step 10 (GenerateIndex) のコード 5行を削除
   * Step 12 の index.md 書き出し部分 5行を削除
   * 不要になった memory import を削除 (ParseAllMemoryDocs がまだ使われているか確認)
   * Step 番号コメントをリナンバリング

4. **Step 4: deploy.go コメント修正**
   * L96-97 のコメントから index.md 言及を除去

5. **Step 5: compiler_test.go 修正**
   * 4つのテスト関数から IndexContent 関連アサーションを除去

6. **Step 6: config_test.go + testdata 修正**
   * `config_test.go` のテスト YAML 文字列から `memory_index` を削除
   * `testdata/valid/prompts/manifest/project.yaml` から `memory_index` を削除
   * `testdata/invalid/prompts/manifest/project.yaml` から `memory_index` を削除

7. **Step 7: template.go + template_test.go 修正**
   * `resolveRef()` から `memory` case を削除
   * テストから `memory index reference` テストケースを削除

8. **Step 8: project.yaml 修正**
   * `prompts/manifest/project.yaml` から `memory_index` 行を削除

9. **Step 9: ワークフロー / ポリシー / capability 修正**
   * `execute-implementation-plan.md` Section 1.3 削除
   * `create-specification.md` Section 1.2 削除
   * `far-knowledge-memory.md` L12-14 削除
   * `record-far-knowledge.md` references から index.md 削除

10. **Step 10: ビルド + テスト + デプロイ確認**
    * `./scripts/process/build.sh --backend-only` でビルド + 単体テスト
    * `./scripts/code/prompt/update.sh` でデプロイ確認
    * `git grep` で残存参照がないことを確認

## Verification Plan

### Automated Verification

1. **Build + Unit Tests**:
   ```bash
   ./scripts/process/build.sh --backend-only
   ```

2. **Full Build (最終確認)**:
   ```bash
   ./scripts/process/build.sh
   ```

3. **Prompt Deploy 確認**:
   ```bash
   ./scripts/code/prompt/update.sh
   ```

4. **残存参照チェック** (refs/ を除く):
   ```bash
   git grep "memory/index.md" -- ':!prompts/phases/000-foundation/refs/'
   git grep "memory:index" -- ':!prompts/phases/000-foundation/refs/'
   git grep "IndexContent" -- ':!prompts/phases/000-foundation/refs/'
   git grep "MemoryIndex" -- ':!prompts/phases/000-foundation/refs/'
   git grep "GenerateIndex" -- ':!prompts/phases/000-foundation/refs/'
   ```

### 統合テスト

本変更はファイル削除とコード除去のみであり、新規機能追加は含まない。compile/deploy パイプラインの既存テストがビルドパイプラインで実行されるため、追加の統合テストは不要。

## Documentation

#### [MODIFY] [006-Remove-Memory-Index.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/006-Remove-Memory-Index.md)
*   **更新内容**: 完了ステータスを追記 (実装完了後)
