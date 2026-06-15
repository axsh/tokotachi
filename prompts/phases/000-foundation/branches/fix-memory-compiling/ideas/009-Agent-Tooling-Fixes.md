# 009-Agent-Tooling-Fixes

## 背景 (Background)

遠方知識の体系化 (systematize-far-knowledge) ワークフローを実行した際に、以下の3つの問題が発覚した:

1. **knowledge list がネストカテゴリを検出できない**: `Store.List()` は `RootDir` 直下のディレクトリのみを走査するため、`agent/record/branch-package` のようなネストされたカテゴリパスを検出できない
2. **`_resolve_tool.sh` が Windows `.exe` に非対応**: フォールバック先が `$project_root/bin/tt` のみで、Windows 環境の `bin/tt.exe` を検出できない
3. **intake list が processed 済みイベントを表示する**: `MoveToProcessed` はファイルを移動するだけで SQLite の `index.db` 内の `status` カラムを更新しないため、`intake list --status pending` が古い情報を返す

## 要件 (Requirements)

### R1: knowledge list のネストカテゴリ対応 (必須)

- `Store.List()` は `RootDir` 以下を再帰的に走査し、`_category.yaml` を持つ全ディレクトリをカテゴリとして返す
- `CategoryInfo.Path` には `RootDir` からの相対パスを返す (例: `agent/record/branch-package`)
- 既存の単層カテゴリ (例: `error-handling`) は引き続き動作する
- `gatherCategoryInfo` は変更不要 (個別カテゴリの情報収集ロジックは正しい)

### R2: `_resolve_tool.sh` の Windows `.exe` 対応 (必須)

- `$project_root/bin/tt` が存在しない場合、`$project_root/bin/tt.exe` もフォールバックとして検索する
- 優先順位: `TT_TOOL` 環境変数 > PATH上の `tt` > `bin/tt` > `bin/tt.exe`
- `.exe` を追加するのは最後のローカルバイナリ検索のみ。PATH 検索に `.exe` を追加する必要はない (Windows の PATH 検索は `.exe` を自動補完する)

### R3: intake processed のインデックス同期 (必須)

- `intake processed` コマンド実行時に、ファイル移動と同時に `index.db` の `status` カラムを `"processed"` に更新する
- ファイル移動が成功しても DB 更新が失敗した場合はエラーを返す (ロールバックは不要、ファイルの移動は完了したものとして扱う)
- ファイルが既に pending に存在しない場合 (既に移動済み) は、DB のみ更新してエラーとしない

## 実現方針 (Implementation Approach)

### R1: `Store.List()` の再帰走査

現在の `List()` (L128-153) を `filepath.WalkDir` を使った再帰走査に変更する。

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
        info, err := s.gatherCategoryInfo(path, relPath)
        if err != nil {
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

### R2: `_resolve_tool.sh` の `.exe` フォールバック

L19-23 のローカルバイナリ検索部分を拡張:

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

### R3: `intake processed` のインデックス同期

`agent_intake.go` の `runAgentIntakeProcessed` に `index.db` の UPDATE 処理を追加。
`status` パッケージに `UpdateStatus(varDir, eventID, newStatus string) error` 関数を新設する。

```go
// status/update.go
func UpdateStatus(varDir, eventID, newStatus string) error {
    dbPath := filepath.Join(varDir, "intake", "index.db")
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return fmt.Errorf("failed to open index: %w", err)
    }
    defer db.Close()

    result, err := db.Exec(
        "UPDATE intake_events SET status = ? WHERE event_id = ?",
        newStatus, eventID,
    )
    if err != nil {
        return fmt.Errorf("failed to update status: %w", err)
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("event %s not found in index", eventID)
    }
    return nil
}
```

`agent_intake.go` の `runAgentIntakeProcessed`:

```go
func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
    varDir := filepath.Join("prompts", "memory", "var")
    eventID := args[0]

    if err := intake.MoveToProcessed(varDir, eventID); err != nil {
        return fmt.Errorf("failed to move event to processed: %w", err)
    }

    // Update index.db
    if err := status.UpdateStatus(varDir, eventID, "processed"); err != nil {
        fmt.Fprintf(os.Stderr, "[WARN] File moved but index update failed: %v\n", err)
    }

    fmt.Printf("Event %s moved to processed\n", eventID)
    return nil
}
```

## 検証シナリオ (Verification Scenarios)

### S1: knowledge list ネストカテゴリ

1. `prompts/memory/knowledge/` 以下に `agent/record/branch-package/_category.yaml` のようなネスト構造を用意する
2. `tt agent knowledge list` を実行する
3. `agent/record/branch-package` がカテゴリとして表示される
4. 既存の単層カテゴリ (テスト用に `testing/` を追加) も表示される

### S2: `_resolve_tool.sh` Windows フォールバック

1. `TT_TOOL` 環境変数を未設定にする
2. PATH 上に `tt` がない状態で、`bin/tt.exe` のみが存在する
3. `_resolve_tool.sh` を source した際に `$TOOL` が `$project_root/bin/tt.exe` に解決される

### S3: intake processed のインデックス同期

1. `tt agent record` でイベントを作成 (pending 状態で `index.db` に挿入される)
2. `tt agent intake list --status pending` で表示されることを確認
3. `tt agent intake processed <event-id>` を実行
4. ファイルが `processed/` に移動されている
5. `tt agent intake list --status pending` で表示されないことを確認
6. `tt agent intake list --status processed` で表示されることを確認

## テスト項目 (Testing for the Requirements)

### R1: knowledge list

- **単体テスト**: `store_test.go` に `TestStore_List_NestedCategories` を追加
  - ネストされたカテゴリ (例: `a/b/c`) が `List()` で返されることを検証
- **e2e テスト**: `far_knowledge_e2e_test.go` にネストカテゴリの list テストを追加

### R2: `_resolve_tool.sh`

- シェルスクリプトのため、手動検証のみ (シナリオ S2)

### R3: intake processed インデックス同期

- **単体テスト**: `status/update_test.go` に `TestUpdateStatus` を追加
  - DB に pending レコードを挿入し、`UpdateStatus` で processed に変更、SELECT で検証
- **統合テスト**: intake の e2e テストに processed 後の list 検証を追加

### ビルド・全体検証

1. ビルド + 単体テスト:
   ```
   scripts/process/build.sh
   ```

2. tt 統合テスト:
   ```
   scripts/process/integration_test.sh --categories "tt" --specify "Knowledge|Intake|FarKnowledge"
   ```
