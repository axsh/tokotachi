# 009-Agent-Tooling-Fixes

> **Source Specification**: [009-Agent-Tooling-Fixes.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/009-Agent-Tooling-Fixes.md)

## Goal Description

systematize-far-knowledge ワークフロー実行時に発覚した3つのツール問題を修正する:
(R1) knowledge list のネストカテゴリ走査、(R2) _resolve_tool.sh の Windows .exe 対応、(R3) intake processed のインデックス同期。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: Store.List() がネストカテゴリを検出する | Proposed Changes > store.go |
| R1: CategoryInfo.Path に RootDir からの相対パスを返す | Proposed Changes > store.go |
| R1: 既存の単層カテゴリが引き続き動作する | Proposed Changes > store_test.go |
| R2: bin/tt.exe をフォールバックで検索する | Proposed Changes > _resolve_tool.sh |
| R2: 優先順位: TT_TOOL > PATH tt > bin/tt > bin/tt.exe | Proposed Changes > _resolve_tool.sh |
| R3: intake processed 時に index.db の status を更新する | Proposed Changes > agent_intake.go |
| R3: ファイル移動成功・DB 更新失敗時はエラーを返す | Proposed Changes > agent_intake.go |

## Proposed Changes

### R1: agent/knowledge パッケージ

#### [MODIFY] [store_test.go](file:///features/tt/internal/agent/knowledge/store_test.go)

*   **Description**: ネストカテゴリの List テストを追加 (TDD: Red first)
*   **Technical Design**:
    ```go
    func TestStore_List_NestedCategories(t *testing.T) {
        root := t.TempDir()
        s := NewStore(root)
        contentDir := t.TempDir()

        cf1 := writeContentFile(t, contentDir, "c1.md", "Content 1")
        cf2 := writeContentFile(t, contentDir, "c2.md", "Content 2")

        // Create nested categories using Add
        require.NoError(t, s.Add("agent/record/branch-package", "Branch Package Info", "Branch package desc", cf1, []string{"E-001"}))
        require.NoError(t, s.Add("prompt/memory-architecture", "Memory Architecture", "Arch desc", cf2, []string{"E-002"}))

        result, err := s.List()
        require.NoError(t, err)

        // Should find both nested categories
        paths := make([]string, len(result))
        for i, r := range result {
            paths[i] = r.Path
        }
        assert.Contains(t, paths, "agent/record/branch-package")
        assert.Contains(t, paths, "prompt/memory-architecture")
    }
    ```
*   **Logic**:
    *   2つのネストカテゴリ (`agent/record/branch-package`, `prompt/memory-architecture`) を作成
    *   `List()` が両方を返し、`Path` がルートからの相対パスであることを検証
    *   既存の `TestStore_List_WithCategories` は単層カテゴリのリグレッションテストとして機能

#### [MODIFY] [store.go](file:///features/tt/internal/agent/knowledge/store.go)

*   **Description**: `List()` を `filepath.WalkDir` による再帰走査に変更
*   **Technical Design**:
    *   import に `"io/fs"` を追加
    *   `List()` 関数 (L128-153) を以下に置き換え:
    ```go
    func (s *Store) List() ([]CategoryInfo, error) {
        var result []CategoryInfo

        if _, err := os.Stat(s.RootDir); os.IsNotExist(err) {
            return result, nil
        }

        err := filepath.WalkDir(s.RootDir, func(path string, d fs.DirEntry, err error) error {
            if err != nil || !d.IsDir() {
                return nil
            }
            // Check if this directory has _category.yaml
            if _, statErr := os.Stat(filepath.Join(path, categoryMetaFile)); statErr != nil {
                return nil
            }
            relPath, _ := filepath.Rel(s.RootDir, path)
            info, gatherErr := s.gatherCategoryInfo(path, relPath)
            if gatherErr != nil {
                return nil // skip broken categories
            }
            result = append(result, *info)
            return nil
        })
        if err != nil {
            return nil, fmt.Errorf("failed to walk knowledge root: %w", err)
        }

        return result, nil
    }
    ```
*   **Logic**:
    *   `filepath.WalkDir` は `RootDir` 以下を再帰的に走査する
    *   各ディレクトリで `_category.yaml` の存在をチェックし、存在するものだけをカテゴリとして扱う
    *   `filepath.Rel(s.RootDir, path)` でルートからの相対パスを算出し `CategoryInfo.Path` に設定
    *   `_category.yaml` を持たない中間ディレクトリ (例: `agent/`, `agent/record/`) はスキップされる

---

### R2: scripts パッケージ

#### [MODIFY] [_resolve_tool.sh](file:///scripts/code/_resolve_tool.sh)

*   **Description**: ローカルバイナリ検索に `bin/tt.exe` のフォールバックを追加
*   **Technical Design**:
    *   L19-23 の後に `.exe` チェックを追加:
    ```bash
    local local_bin="$project_root/bin/tt"
    if [ -x "$local_bin" ]; then
        echo "$local_bin"
        return 0
    fi
    local local_bin_exe="$project_root/bin/tt.exe"
    if [ -x "$local_bin_exe" ]; then
        echo "$local_bin_exe"
        return 0
    fi
    ```
*   **Logic**:
    *   `bin/tt` が見つからない場合のみ `bin/tt.exe` を検索する
    *   優先順位は仕様通り: `TT_TOOL` 環境変数 > PATH 上の `tt` > `bin/tt` > `bin/tt.exe`

---

### R3: intake processed のインデックス同期

#### [MODIFY] [list_test.go](file:///features/tt/internal/agent/status/list_test.go)

*   **Description**: ステータス更新後の List フィルタリングテストを追加 (TDD: Red first)
*   **Technical Design**:
    ```go
    func TestList_FilterByStatus_AfterUpdate(t *testing.T) {
        tmpDir := t.TempDir()
        idx := setupTestIndex(t, tmpDir)

        storeTestEvent(t, idx, "E-010", "antigravity", "main", "Task to process")
        storeTestEvent(t, idx, "E-011", "antigravity", "main", "Task stays pending")

        // Update E-010 to processed
        require.NoError(t, idx.UpdateStatus("E-010", "processed"))
        idx.Close()

        // List pending: should only see E-011
        pendingItems, err := List(tmpDir, ListOptions{Status: "pending"})
        require.NoError(t, err)
        assert.Len(t, pendingItems, 1)
        assert.Equal(t, "E-011", pendingItems[0].EventID)

        // List processed: should only see E-010
        processedItems, err := List(tmpDir, ListOptions{Status: "processed"})
        require.NoError(t, err)
        assert.Len(t, processedItems, 1)
        assert.Equal(t, "E-010", processedItems[0].EventID)
    }
    ```
*   **Logic**:
    *   2つの pending イベントを作成
    *   `Index.UpdateStatus` で1つを processed に変更
    *   `List` で status フィルタが正しく機能することを検証

#### [MODIFY] [agent_intake.go](file:///features/tt/cmd/agent_intake.go)

*   **Description**: `runAgentIntakeProcessed` にインデックス更新処理を追加
*   **Technical Design**:
    *   import に `"github.com/axsh/tokotachi/features/tt/internal/agent/storage"` を追加
    *   `runAgentIntakeProcessed` 関数を変更:
    ```go
    func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
        varDir := filepath.Join("prompts", "memory", "var")
        eventID := args[0]

        if err := intake.MoveToProcessed(varDir, eventID); err != nil {
            return fmt.Errorf("failed to move event to processed: %w", err)
        }

        // Update index.db status
        dbPath := filepath.Join(varDir, "intake", "index.db")
        idx, err := storage.NewIndex(dbPath)
        if err != nil {
            fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update skipped: %v\n", err)
        } else {
            defer idx.Close()
            if err := idx.UpdateStatus(eventID, "processed"); err != nil {
                fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update failed: %v\n", err)
            }
        }

        fmt.Printf("Event %s moved to processed\n", eventID)
        return nil
    }
    ```
*   **Logic**:
    *   ファイル移動が成功した後に `storage.NewIndex` で DB を開き `UpdateStatus` を呼ぶ
    *   `Index.UpdateStatus` は既に `storage/index.go` L217-230 に実装済み
    *   DB のオープンやアップデートが失敗した場合は WARN メッセージを出力するが、ファイル移動は既に完了しているためエラーとしては返さない (仕様: 「ロールバックは不要」)

## Step-by-Step Implementation Guide

1.  **テストの追加 (TDD: Red)**:
    *   `store_test.go` に `TestStore_List_NestedCategories` を追加
    *   `list_test.go` に `TestList_FilterByStatus_AfterUpdate` を追加
    *   テスト実行して失敗を確認:
        ```bash
        ./scripts/process/build.sh --backend-only
        ```

2.  **R1: store.go の修正 (TDD: Green)**:
    *   `List()` を `filepath.WalkDir` 再帰走査に変更
    *   import に `"io/fs"` を追加

3.  **R3: agent_intake.go の修正 (TDD: Green)**:
    *   `runAgentIntakeProcessed` に `storage.NewIndex` + `UpdateStatus` の呼び出しを追加
    *   import に `"github.com/axsh/tokotachi/features/tt/internal/agent/storage"` を追加

4.  **テスト実行 (TDD: Green 確認)**:
    *   修正後にテスト通過を確認:
        ```bash
        ./scripts/process/build.sh --backend-only
        ```

5.  **R2: _resolve_tool.sh の修正**:
    *   `bin/tt.exe` フォールバックを追加

6.  **Verification Plan の実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:

    本実装はファイルI/Oおよび SQLite 操作を含むため、統合テストが必要。
    ただし `tests/` 配下に knowledge や intake 専用のカテゴリは存在しない。
    e2e テストは `features/tt/internal/agent/e2e/` にあり `go test` (build.sh 内) で実行される。

    統合テストカテゴリ `tt` に関連テストが含まれる可能性があるため確認:
    ```bash
    ./scripts/process/integration_test.sh --categories "tt" --specify "Knowledge|Intake|FarKnowledge"
    ```

3.  **_resolve_tool.sh の手動検証**:
    シェルスクリプトの変更は自動テスト対象外。以下の手順で確認:
    ```bash
    unset TT_TOOL
    source scripts/code/_resolve_tool.sh
    echo "$TOOL"
    ```
    `bin/tt.exe` が出力されることを確認。
