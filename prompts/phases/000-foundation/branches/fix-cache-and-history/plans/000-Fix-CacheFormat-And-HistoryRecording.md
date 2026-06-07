# 000-Fix-CacheFormat-And-HistoryRecording

> **Source Specification**: [000-Fix-CacheFormat-And-HistoryRecording.md](file://prompts/phases/000-foundation/ideas/fix-cache-and-history/000-Fix-CacheFormat-And-HistoryRecording.md)

## Goal Description

キャッシュフォーマットをディレクトリベースに改善し、ダウンロード履歴の統合テストを補強し、`--cwd` フラグを `--root {path}` に変更する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: キャッシュを `meta.yaml` + `data` のディレクトリ構造に変更 | Proposed Changes > scaffold パッケージ > `cache.go`, `cache_test.go`, `scaffold.go` |
| R2: 依存チェーン展開時の履歴記録テスト補強 | Proposed Changes > 統合テスト > `tt_scaffold_test.go` |
| R3: `--cwd` → `--root {path}` に変更 | Proposed Changes > CLI > `scaffold.go` (cmd), 統合テスト |

## Proposed Changes

### scaffold パッケージ

#### [MODIFY] [cache_test.go](file://pkg/scaffold/cache_test.go)

*   **Description**: 新しいディレクトリベースのキャッシュ構造に対応するテストに変更
*   **Technical Design**:
    *   `TestCacheStore_SaveAndLoad` — `meta.yaml` と `data` ファイルが正しく保存・読み込みされることを検証
    *   `TestCacheStore_Load_NotExist` — キャッシュが存在しない場合の動作を検証
    *   `TestCacheStore_IsValid` — `meta.yaml` の `updated_at` で有効性判定を検証
    *   `TestCacheStore_EnsureGitignore` — `.gitignore` 更新を検証
    *   `TestCacheStore_DataFileIntegrity` — `data` ファイルが完全に無加工で保存されることを検証（バイナリデータの round-trip）
*   **Logic**:
    *   `Save` 後に `{cacheDir}/repository_data/catalog.yaml/meta.yaml` と `{cacheDir}/repository_data/catalog.yaml/data` の存在を確認
    *   `meta.yaml` を YAML パースして `updated_at` と `cached_at` フィールドを検証
    *   `data` ファイルの内容が入力データと完全一致することを検証
    *   テストケース:
    ```go
    tests := []struct {
        name        string
        updatedAt   string
        catalogData []byte // raw binary content
    }{
        {"yaml_content", "2026-03-10T19:00:00+09:00", []byte("scaffolds:\n  root:\n    default: path\n")},
        {"binary_content", "2026-03-15T00:00:00Z", []byte{0x00, 0xFF, 0x89, 0x50}},
    }
    ```

#### [MODIFY] [cache.go](file://pkg/scaffold/cache.go)

*   **Description**: キャッシュストアのパス構造をディレクトリベースに変更し、`Save`/`Load` を `meta.yaml` + `data` ファイル方式に修正
*   **Technical Design**:
    ```go
    const (
        CacheDir      = ".kotoshiro/tokotachi/.cache"
        CacheCategory = "repository_data"  // cache category name
        CacheItemName = "catalog.yaml"     // cached item name (used as folder name)
        MetaFileName  = "meta.yaml"
        DataFileName  = "data"
    )

    // CacheMeta represents the meta.yaml file for a cached item.
    type CacheMeta struct {
        UpdatedAt string `yaml:"updated_at"` // remote timestamp for validity check
        CachedAt  string `yaml:"cached_at"`  // local timestamp of when the cache was saved
    }

    // CacheStore manages reading and writing of cached catalog data.
    type CacheStore struct {
        repoRoot string
    }
    ```
    *   `CachedCatalog` 構造体を削除し、`CacheMeta` 構造体に置き換え
    *   `CachePath()` → キャッシュアイテムのディレクトリパスを返す（`{repoRoot}/.cache/repository_data/catalog.yaml/`）
    *   `metaPath()` → `{CachePath()}/meta.yaml`
    *   `dataPath()` → `{CachePath()}/data`
*   **Logic**:
    *   **`Save(updatedAt string, data []byte) error`**: 
        1. `os.MkdirAll(CachePath(), 0o755)` でディレクトリ作成
        2. `CacheMeta{UpdatedAt: updatedAt, CachedAt: time.Now().UTC().Format(time.RFC3339)}` を `meta.yaml` に YAML で保存
        3. `os.WriteFile(dataPath(), data, 0o644)` で `data` ファイルにバイナリデータを無加工で保存
        4. `EnsureGitignore()` を呼び出し
    *   **`Load() (*CacheMeta, []byte, error)`**:
        1. `meta.yaml` をパースして `CacheMeta` を取得
        2. `data` ファイルから `[]byte` を読み込み
        3. いずれかのファイルが存在しない場合は `nil, nil, nil` を返す
    *   **`IsValid(remoteUpdatedAt string) bool`**:
        1. `meta.yaml` のみ読み込み、`UpdatedAt == remoteUpdatedAt` で判定

#### [MODIFY] [scaffold.go](file://pkg/scaffold/scaffold.go)

*   **Description**: `List()` 関数のキャッシュ利用部分を新しい `CacheStore` API に合わせて修正
*   **Technical Design**:
    *   `List()` 内の L426-451 のキャッシュ読み込み・保存ロジックを変更
*   **Logic**:
    *   **キャッシュ読み込み** (旧: `cached.CatalogData` → 新: `Load()` の戻り値 `[]byte`):
        ```go
        // Try cache
        if repoRoot != "" {
            cache := NewCacheStore(repoRoot)
            if cache.IsValid(meta.UpdatedAt) {
                _, data, err := cache.Load()
                if err == nil && data != nil {
                    catalogData = data
                }
            }
        }
        ```
    *   **キャッシュ保存** (旧: `Save(&CachedCatalog{...})` → 新: `Save(updatedAt, data)`):
        ```go
        if repoRoot != "" {
            cache := NewCacheStore(repoRoot)
            _ = cache.Save(meta.UpdatedAt, catalogData)
        }
        ```

---

### 統合テスト

#### [MODIFY] [tt_scaffold_test.go](file://tests/integration-test/tt_scaffold_test.go)

*   **Description**: 依存チェーン展開時の `downloaded.yaml` 記録のテストを `TestScaffoldWithDependencies` に追加。`TestScaffoldCwdFlag` を `TestScaffoldRootFlag` に変更。
*   **Technical Design**:
    *   `TestScaffoldWithDependencies` に `DownloadHistoryRecording` サブテストを追加
    *   `TestScaffoldCwdFlag` → `TestScaffoldRootFlag` にリネーム、`--cwd` → `--root {tmpDir}` に変更
*   **Logic**:
    *   **`DownloadHistoryRecording` サブテスト**:
        1. `TestScaffoldWithDependencies` の scaffold 適用完了後に `downloaded.yaml` を読み込む
        2. `root/default` が記録されていることを `assert.Contains` で検証
        3. `project/axsh-go-standard` が記録されていることを検証
        4. `feature/axsh-go-standard` が記録されていないことを、キーが2つ（`root` と `project`）しかないことで検証
    *   **`TestScaffoldRootFlag`**:
        1. `tmpDir` を作成（Git リポジトリでない）
        2. `runTTInDir(t, ".", "scaffold", "--root", tmpDir, "--yes")` で実行
        3. `tmpDir` 配下にファイルが展開されることを検証

---

### CLI

#### [MODIFY] [scaffold.go (cmd)](file://features/tt/cmd/scaffold.go)

*   **Description**: `--cwd` bool フラグを `--root` string フラグに変更
*   **Technical Design**:
    ```go
    var (
        // scaffoldFlagCwd bool → 削除
        scaffoldFlagRoot string  // new: --root flag
    )

    func init() {
        // 旧: scaffoldCmd.Flags().BoolVar(&scaffoldFlagCwd, "cwd", false, ...)
        // 新:
        scaffoldCmd.Flags().StringVar(&scaffoldFlagRoot, "root", "",
            "Specify root directory path instead of auto-detecting Git root")
    }

    func runScaffold(cmd *cobra.Command, args []string) error {
        // 旧: repoRoot := resolveRepoRoot(scaffoldFlagCwd)
        // 新:
        repoRoot := resolveRepoRoot(scaffoldFlagRoot)
        // ...
    }

    // resolveRepoRoot determines the target root directory.
    // If rootPath is non-empty, uses that path directly.
    // Otherwise, tries "git rev-parse --show-toplevel" first,
    // falling back to os.Getwd() on failure.
    func resolveRepoRoot(rootPath string) string {
        if rootPath != "" {
            return rootPath
        }
        // ... existing git logic ...
    }
    ```
*   **Logic**:
    *   `scaffoldFlagCwd bool` → `scaffoldFlagRoot string` に変更
    *   `--cwd` フラグ定義行を `--root` に変更（`BoolVar` → `StringVar`）
    *   `resolveRepoRoot(useCwd bool)` → `resolveRepoRoot(rootPath string)` にシグネチャ変更
    *   `rootPath` が空でなければそのまま返す。空なら従来通り Git ルートを検出

## Step-by-Step Implementation Guide

### Phase 1: キャッシュフォーマット改善 (R1)

1.  **テスト先行: `cache_test.go` 修正**:
    *   `TestCacheStore_SaveAndLoad` を新しい API（`Save(updatedAt, data)`, `Load() (*CacheMeta, []byte, error)`）に合わせて書き直す
    *   `TestCacheStore_DataFileIntegrity` テストを追加（バイナリデータの round-trip 検証）
    *   `TestCacheStore_Load_NotExist`, `TestCacheStore_IsValid`, `TestCacheStore_EnsureGitignore` を新 API に合わせて修正
    *   ビルド失敗を確認（`CacheMeta` 未定義、`Save`/`Load` のシグネチャ不一致）

2.  **実装: `cache.go` 修正**:
    *   `CachedCatalog` → `CacheMeta` に置き換え
    *   定数 `CacheCategory`, `CacheItemName`, `MetaFileName`, `DataFileName` を追加
    *   `CachePath()`, `metaPath()`, `dataPath()` メソッドを修正・追加
    *   `Save(updatedAt string, data []byte) error` を実装
    *   `Load() (*CacheMeta, []byte, error)` を実装
    *   `IsValid(remoteUpdatedAt string) bool` を修正
    *   テスト成功を確認

3.  **実装: `scaffold.go` の `List()` 関数修正**:
    *   キャッシュ読み込み部分を新しい `Load()` API に合わせる
    *   キャッシュ保存部分を新しい `Save()` API に合わせる
    *   テスト成功を確認

4.  **ビルド確認**:
    *   `./scripts/process/build.sh` を実行してビルドと単体テストの成功を確認

### Phase 2: ダウンロード履歴テスト補強 (R2)

5.  **統合テスト: `tt_scaffold_test.go` に `DownloadHistoryRecording` サブテスト追加**:
    *   `TestScaffoldWithDependencies` 内に `DownloadHistoryRecording` サブテストを追加
    *   依存チェーン展開後の `downloaded.yaml` を読み込み、静的 scaffold が記録され動的 scaffold が記録されないことを検証

### Phase 3: `--root` オプション (R3)

6.  **テスト先行: 統合テスト `TestScaffoldRootFlag` 修正**:
    *   `TestScaffoldCwdFlag` → `TestScaffoldRootFlag` にリネーム
    *   `--cwd` → `--root {tmpDir}` に変更

7.  **実装: `features/tt/cmd/scaffold.go` 修正**:
    *   `scaffoldFlagCwd` → `scaffoldFlagRoot` に変更
    *   `--cwd` → `--root` フラグ定義を変更
    *   `resolveRepoRoot` のシグネチャと実装を修正

8.  **ビルド確認**:
    *   `./scripts/process/build.sh` を実行

### Phase 4: 最終検証

9.  **統合テスト実行**:
    *   `./scripts/process/integration_test.sh --categories scaffold` を実行して全統合テストが通ることを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories scaffold --specify "TestScaffoldWithDependencies"
    ./scripts/process/integration_test.sh --categories scaffold --specify "TestScaffoldRootFlag"
    ```
    *   **Log Verification**:
        *   `TestScaffoldWithDependencies/DownloadHistoryRecording`: `downloaded.yaml` に `root/default` と `project/axsh-go-standard` が含まれ、`feature` カテゴリが含まれないことを確認
        *   `TestScaffoldRootFlag`: `--root` で指定したパスにファイルが展開されることを確認

## Documentation

本計画の変更範囲では、`prompts/specifications` フォルダ以下に影響を受ける既存ドキュメントはありません。
