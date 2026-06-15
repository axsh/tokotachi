# 008-Knowledge-Frontmatter-Mismatch

> **Source Specification**: [008-Knowledge-Frontmatter-Mismatch.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/008-Knowledge-Frontmatter-Mismatch.md)

## Goal Description

`tt agent knowledge add` が生成する knowledge ドキュメントの YAML frontmatter に `id` および `status` フィールドが欠落しており、`tt prompt compile` のバリデーションが失敗する問題を修正する。`agent/knowledge` パッケージの `KnowledgeFileMeta` 構造体にフィールドを追加し、生成ロジックを更新することで、knowledge ファイルがそのまま prompt compile を通過できるようにする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `knowledge add` の出力が `prompt compile` のバリデーションを通過すること | Proposed Changes > types.go, store.go |
| `MemoryDoc` の frontmatter 仕様を正とし `agent/knowledge` 側を合わせること | Proposed Changes > types.go |
| `id` フィールドを追加し `knowledge_id` と同じ値を設定すること | Proposed Changes > types.go, store.go |
| `status` フィールドを追加しデフォルト値 `current` を設定すること | Proposed Changes > types.go, store.go |
| 既存のテストが全て通過すること | Verification Plan |

## Proposed Changes

### agent/knowledge パッケージ

#### [MODIFY] [frontmatter_test.go](file:///features/tt/internal/agent/knowledge/frontmatter_test.go)

*   **Description**: `KnowledgeFileMeta` の新フィールド (`ID`, `Status`) の読み書きラウンドトリップを検証するテストを追加。
*   **Technical Design**:
    *   `TestReadWriteFrontmatter` に `ID` と `Status` フィールドのアサーションを追加
    *   新テスト `TestReadWriteFrontmatter_IDAndStatus` を追加して、出力された YAML に `id:` と `status:` が含まれることを検証
*   **Logic**:
    *   テストで `KnowledgeFileMeta` を作成する際に `ID: "api-error-responses"`, `Status: "current"` を設定
    *   `WriteFrontmatter` で書き出したファイルを `ReadFrontmatter` で読み返し、`meta.ID == "api-error-responses"` および `meta.Status == "current"` を assert
    *   ファイル内容を直接読み取り、`id: api-error-responses` と `status: current` の行が含まれることを assert

#### [MODIFY] [store_test.go](file:///features/tt/internal/agent/knowledge/store_test.go)

*   **Description**: `TestStore_Add` と `TestStore_Append` で新フィールドの存在を検証。
*   **Technical Design**:
    *   `TestStore_Add` に `meta.ID` と `meta.Status` のアサーションを追加
*   **Logic**:
    *   `assert.Equal(t, "api-error-responses", meta.ID)` を追加
    *   `assert.Equal(t, "current", meta.Status)` を追加

#### [MODIFY] [types.go](file:///features/tt/internal/agent/knowledge/types.go)

*   **Description**: `KnowledgeFileMeta` に `ID` と `Status` フィールドを追加。
*   **Technical Design**:
    ```go
    type KnowledgeFileMeta struct {
        ID             string    `yaml:"id"`
        KnowledgeID    string    `yaml:"knowledge_id"`
        Title          string    `yaml:"title"`
        Status         string    `yaml:"status"`
        CategoryPath   string    `yaml:"category_path"`
        CreatedAt      time.Time `yaml:"created_at"`
        LastUpdated    time.Time `yaml:"last_updated"`
        SourceEventIDs []string  `yaml:"source_event_ids"`
    }
    ```
*   **Logic**:
    *   `ID` は YAML タグ `"id"` で、`prompt/memory/frontmatter.go` の `ParseFrontmatter` が期待する `metaData["id"]` に対応
    *   `Status` は YAML タグ `"status"` で、`ParseFrontmatter` が期待する `metaData["status"]` に対応
    *   `KnowledgeID` は後方互換性のために残す（既存コードが参照しているため）

#### [MODIFY] [store.go](file:///features/tt/internal/agent/knowledge/store.go)

*   **Description**: `Add` と `Append` で `KnowledgeFileMeta` を作成する際に `ID` と `Status` を設定。
*   **Technical Design**:
    *   `Add` 関数 (L68-75) の `KnowledgeFileMeta` 生成部を変更
    *   `Append` 関数 (L101-108) の `KnowledgeFileMeta` 生成部を変更
*   **Logic**:
    *   `Add` 関数の meta 生成部 (L68-75):
        ```go
        meta := &KnowledgeFileMeta{
            ID:             knowledgeID,
            KnowledgeID:    knowledgeID,
            Title:          title,
            Status:         "current",
            CategoryPath:   categoryPath,
            CreatedAt:      now,
            LastUpdated:    now,
            SourceEventIDs: sourceEvents,
        }
        ```
    *   `Append` 関数の meta 生成部 (L101-108):
        ```go
        meta := &KnowledgeFileMeta{
            ID:             knowledgeID,
            KnowledgeID:    knowledgeID,
            Title:          title,
            Status:         "current",
            CategoryPath:   categoryPath,
            CreatedAt:      now,
            LastUpdated:    now,
            SourceEventIDs: sourceEvents,
        }
        ```

### agent/e2e パッケージ

#### [MODIFY] [far_knowledge_e2e_test.go](file:///features/tt/internal/agent/e2e/far_knowledge_e2e_test.go)

*   **Description**: `TestFarKnowledge_Phase0_KnowledgeAdd` に `ID` と `Status` のアサーションを追加。
*   **Technical Design**:
    *   L41 の `assert.Equal(t, "api-error-responses", meta.KnowledgeID)` の後に追加
*   **Logic**:
    *   `assert.Equal(t, "api-error-responses", meta.ID)` を追加
    *   `assert.Equal(t, "current", meta.Status)` を追加

### 既存 knowledge ファイルの修正

#### [MODIFY] [branchpackageinfo-and-slugify.md](file:///prompts/memory/knowledge/agent/record/branch-package/branchpackageinfo-and-slugify.md)

*   **Description**: 既存の knowledge ファイルの frontmatter に `id` と `status` フィールドを追加。
*   **Technical Design**:
    *   frontmatter に `id: branchpackageinfo-and-slugify` を追加
    *   frontmatter に `status: current` を追加
*   **Logic**:
    *   変更後の frontmatter:
        ```yaml
        ---
        id: branchpackageinfo-and-slugify
        knowledge_id: branchpackageinfo-and-slugify
        title: BranchPackageInfo and Slugify
        status: current
        category_path: agent/record/branch-package
        created_at: 2026-06-15T13:53:46.3437296Z
        last_updated: 2026-06-15T13:53:46.3437296Z
        source_event_ids:
            - E-01KTHNQGQXX4S6M0EETHKRPT0S
        ---
        ```

## Step-by-Step Implementation Guide

1.  **テストの追加 (TDD: Red)**:
    *   `frontmatter_test.go` の `TestReadWriteFrontmatter` に `meta.ID` と `meta.Status` のフィールドを設定し、ラウンドトリップ後にそれらが保持されることを assert するコードを追加
    *   `store_test.go` の `TestStore_Add` に `meta.ID` と `meta.Status` の assert を追加
    *   `far_knowledge_e2e_test.go` の `TestFarKnowledge_Phase0_KnowledgeAdd` に `meta.ID` と `meta.Status` の assert を追加
    *   テストを実行して失敗することを確認:
        ```bash
        ./scripts/process/build.sh --skip-frontend --skip-etc
        ```

2.  **`types.go` の修正 (TDD: Green - 構造体)**:
    *   `KnowledgeFileMeta` 構造体に `ID string yaml:"id"` と `Status string yaml:"status"` フィールドを追加

3.  **`store.go` の修正 (TDD: Green - ロジック)**:
    *   `Add` 関数の `KnowledgeFileMeta` 生成部に `ID: knowledgeID` と `Status: "current"` を追加
    *   `Append` 関数の `KnowledgeFileMeta` 生成部に `ID: knowledgeID` と `Status: "current"` を追加

4.  **テストの実行 (TDD: Green 確認)**:
    *   テストを実行して全て通過することを確認:
        ```bash
        ./scripts/process/build.sh --skip-frontend --skip-etc
        ```

5.  **既存 knowledge ファイルの修正**:
    *   `prompts/memory/knowledge/agent/record/branch-package/branchpackageinfo-and-slugify.md` の frontmatter に `id` と `status` を追加

6.  **prompt compile の検証**:
    *   `./bin/tt.exe prompt compile --dry-run` を実行してバリデーションエラーが発生しないことを確認

7.  **Verification Plan の実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Prompt Compile Validation**:
    ```bash
    ./bin/tt.exe prompt compile --dry-run
    ```
    *   **Log Verification**: `ERROR` を含む行が出力されないことを確認。全ターゲットに対して正常にコンパイルが完了すること。

3.  **統合テスト**:

    本実装はファイルI/Oを行う knowledge store に変更を加えるため、e2e テストが必要。
    ただし、e2e テストは `features/tt/internal/agent/e2e/` パッケージ内にあり、`build.sh` の単体テスト実行で網羅される（`go test ./...` の一部として実行）。
    別途 `integration_test.sh` によるテストは不要。理由: `tests/` 配下に knowledge 関連の統合テストカテゴリが存在しない。e2e テストは `features/tt/internal/agent/e2e/` にあり `go test` で実行される。
