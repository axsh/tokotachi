# 001-Remove-CatalogYaml-Dependency

> **Source Specification**: [001-Remove-CatalogYaml-Dependency.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/001-Remove-CatalogYaml-Dependency.md)

## Goal Description

`tt scaffold` コマンドから `catalog.yaml` への依存を排除し、方式A（FNV-1a ハッシュによるダイレクトアクセス）でシャーディング YAML へ直接到達するように変更する。合わせて `catalog.yaml` キャッシュ機能と `.gitignore` 共通ユーティリティを実装する。

## User Review Required

> [!IMPORTANT]
> - **`gitignore.go` の配置先**: `features/tt/internal/scaffold/` 内に配置予定。scaffold パッケージ外で共有する見込みがなければ scaffold パッケージ内で十分。将来的に他パッケージから利用する場合は `features/tt/internal/gitutil/` に移動を検討。
> - **`meta.yaml` の取得**: キャッシュ判定のため `--list` 実行時のみ `meta.yaml` を取得する。通常の scaffold 実行時（引数あり/なし）は `meta.yaml` を取得しない。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point |
| :--- | :--- |
| シャーディングパス算出アルゴリズム | `shard.go` |
| デフォルト scaffold の定数定義 | `shard.go` (`DefaultCategory`, `DefaultName`) |
| 引数パターンごとの解決ロジック | `scaffold.go` (`fetchAndResolveEntry` 書き換え) |
| シャーディングファイルからのエントリ特定 | `scaffold.go` (`fetchAndResolveEntry`) |
| `catalog.yaml` フェッチの廃止 | `scaffold.go` (`Run`/`Apply`/`List` 変更) |
| `--list` の動作 | `scaffold.go` (`List` 関数変更) |
| `catalog.yaml` のキャッシュ機能 | `cache.go` |
| `.gitignore` 操作の共通ユーティリティ化 | `gitignore.go`, `applier.go` 変更 |

## Proposed Changes

### scaffold パッケージ (`features/tt/internal/scaffold/`)

---

#### [NEW] [shard_test.go](file://features/tt/internal/scaffold/shard_test.go)
*   **Description**: シャーディングパス算出のテスト（TDD: テスト先行）
*   **Technical Design**:
    ```go
    func TestShardPath(t *testing.T) {
        // テーブル駆動テスト
        tests := []struct {
            category, name, expected string
        }{
            {"root", "default", "catalog/scaffolds/6/j/v/n.yaml"},
            {"feature", "axsh-go-standard", "catalog/scaffolds/b/i/b/l.yaml"},
            {"project", "axsh-go-standard", "catalog/scaffolds/8/w/4/o.yaml"},
            {"feature", "axsh-go-kotoshiro-mcp", "catalog/scaffolds/i/4/2/h.yaml"},
        }
        // for _, tt := range tests { assert ShardPath(tt.category, tt.name) == tt.expected }
    }
    ```

---

#### [NEW] [shard.go](file://features/tt/internal/scaffold/shard.go)
*   **Description**: FNV-1a ハッシュ + base36 + パス算出、定数定義
*   **Technical Design**:
    ```go
    const (
        DefaultCategory = "root"
        DefaultName     = "default"
    )

    var KnownCategories = []string{"root", "project", "feature"}

    // ShardPath は category と name からシャーディングファイルのパスを算出する
    func ShardPath(category, name string) string
    ```
*   **Logic**:
    1. `key = category + "/" + name`
    2. FNV-1a 32bit ハッシュ計算:
       - `offset_basis = 2166136261`, `prime = 16777619`
       - `hash = offset_basis`
       - 各バイト: `hash = (hash XOR byte) * prime`, `hash &= 0xFFFFFFFF`
    3. `reduced = hash % 1679616` (1679616 = 36^4)
    4. base36 4文字エンコード（`0-9a-z`、0パディング、逆順ビルド）
    5. 返却: `"catalog/scaffolds/{c[0]}/{c[1]}/{c[2]}/{c[3]}.yaml"`

---

#### [NEW] [gitignore_test.go](file://features/tt/internal/scaffold/gitignore_test.go)
*   **Description**: gitignore ユーティリティのテスト（TDD: テスト先行）
*   **Technical Design**:
    ```go
    func TestGitignoreAddEntries(t *testing.T)        // 基本追加、重複スキップ
    func TestGitignoreAddEntries_WithComments(t *testing.T)  // コメント行を壊さない
    func TestGitignoreAddEntries_TrimTrailing(t *testing.T)  // 末尾スペース対応
    func TestGitignoreRemoveEntries(t *testing.T)     // エントリ削除
    func TestGitignoreAddEntries_CreateFile(t *testing.T)    // ファイル未存在時の新規作成
    func TestGitignoreHasEntry(t *testing.T)           // エントリ存在確認
    ```

---

#### [NEW] [gitignore.go](file://features/tt/internal/scaffold/gitignore.go)
*   **Description**: `.gitignore` 操作の共通ユーティリティ
*   **Technical Design**:
    ```go
    // Gitignore は .gitignore ファイルの読み書きとエントリ管理を行う
    type Gitignore struct {
        lines []string // 元のファイル内容（コメント・空行含む）
    }

    // LoadGitignore はファイルから読み込む（存在しなければ空）
    func LoadGitignore(path string) (*Gitignore, error)

    // Save はファイルに書き出す
    func (g *Gitignore) Save(path string) error

    // AddEntries は重複チェック付きでエントリを追加する
    // コメント行・空行はスキップし、有効なパターン行のみ比較
    func (g *Gitignore) AddEntries(entries []string) (added int)

    // RemoveEntries は指定パターンを完全一致で削除する
    func (g *Gitignore) RemoveEntries(entries []string) (removed int)

    // HasEntry は指定パターンのエントリが存在するか確認する
    func (g *Gitignore) HasEntry(entry string) bool
    ```
*   **Logic**:
    - `AddEntries`: 各行を `strings.TrimRight(line, " \t")` してから有効行（`#` で先頭でない、空でない）をセットに入れて重複チェック
    - `RemoveEntries`: lines スライスをフィルタして該当行を除去
    - `Save`: 末尾に改行を保証して書き出し

---

#### [NEW] [cache_test.go](file://features/tt/internal/scaffold/cache_test.go)
*   **Description**: キャッシュ読み書きテスト（TDD: テスト先行）
*   **Technical Design**:
    ```go
    func TestCacheStore_SaveAndLoad(t *testing.T)     // 保存→読み込みの往復
    func TestCacheStore_Load_NotExist(t *testing.T)    // キャッシュ未存在時
    func TestCacheStore_IsValid(t *testing.T)          // 有効性判定
    func TestCacheStore_EnsureGitignore(t *testing.T)  // .gitignore に .cache/ が追加される
    ```

---

#### [NEW] [cache.go](file://features/tt/internal/scaffold/cache.go)
*   **Description**: `catalog.yaml` のキャッシュ管理
*   **Technical Design**:
    ```go
    const (
        CacheDir      = ".kotoshiro/tokotachi/.cache"
        CacheFileName = "catalog.yaml"
    )

    // CachedCatalog はキャッシュファイルの構造
    type CachedCatalog struct {
        UpdatedAt string `yaml:"updated_at"`
        Catalog   []byte `yaml:"catalog"` // catalog.yaml の生データ
    }

    // CacheStore はキャッシュの読み書きを管理する
    type CacheStore struct {
        repoRoot string // .git があるディレクトリ
    }

    // NewCacheStore はキャッシュストアを初期化する
    func NewCacheStore(repoRoot string) *CacheStore

    // Load はキャッシュファイルを読み込む（未存在なら nil, nil）
    func (s *CacheStore) Load() (*CachedCatalog, error)

    // Save はキャッシュファイルを保存する
    // .gitignore への .cache/ エントリ追加も行う
    func (s *CacheStore) Save(catalog *CachedCatalog) error

    // CachePath はキャッシュファイルの絶対パスを返す
    func (s *CacheStore) CachePath() string

    // EnsureGitignore は .kotoshiro/tokotachi/.gitignore に .cache/ エントリを追加する
    func (s *CacheStore) EnsureGitignore() error
    ```
*   **Logic**:
    - `Save`:
      1. `.kotoshiro/tokotachi/.cache/` ディレクトリを作成（`os.MkdirAll`）
      2. YAML としてシリアライズして書き出し
      3. `.kotoshiro/tokotachi/.gitignore` に `.cache/` エントリを追加（`gitignore.go` ユーティリティ使用）
    - `Load`:
      1. キャッシュファイルの存在確認
      2. 存在すれば YAML デシリアライズして返す
    - `EnsureGitignore`: `LoadGitignore` → `AddEntries([".cache/"])` → `Save`

---

#### [MODIFY] [applier.go](file://features/tt/internal/scaffold/applier.go)
*   **Description**: `applyGitignoreEntries` を `gitignore.go` ユーティリティに置き換え
*   **Technical Design**:
    - 既存の `applyGitignoreEntries` 関数を削除
    - `ApplyPostActions` 内の呼び出しを変更:
      ```go
      // Before:
      // applyGitignoreEntries(actions.GitignoreEntries, repoRoot)

      // After:
      gitignorePath := filepath.Join(repoRoot, ".gitignore")
      gi, err := LoadGitignore(gitignorePath)
      // error handling
      gi.AddEntries(actions.GitignoreEntries)
      gi.Save(gitignorePath)
      ```
    - `splitLines` ヘルパーは `gitignore.go` に移動するか、両方で使えるようにする

---

#### [MODIFY] [scaffold.go](file://features/tt/internal/scaffold/scaffold.go)
*   **Description**: `fetchAndResolveEntry` を方式A （シャーディングパス算出）に書き換え、`List` 関数をキャッシュ付き処理に変更
*   **Technical Design**:
    *   `fetchAndResolveEntry` の変更:
        ```go
        func fetchAndResolveEntry(opts RunOptions, spinner *Spinner) (
            *github.Client, *ScaffoldEntry, []ScaffoldEntry, error) {

            downloader, _ := github.NewClient(opts.RepoURL)

            // 引数解析
            category, name := resolveArgs(opts.Pattern)

            // category のみ（--list 以外） → エラー
            if name == "" {
                return nil, nil, nil, fmt.Errorf("scaffold name is required: tt scaffold <category> <name>")
            }

            // シャーディングパス算出（方式A）
            shardPath := ShardPath(category, name)

            // シャーディング YAML 取得
            shardData, err := downloader.FetchFile(shardPath)
            // error handling

            // パース＋category+nameでフィルタ
            entries, _ := ParseScaffoldDetail(shardData)
            entry := findEntry(entries, category, name)
            // ...
        }
        ```
    *   `resolveArgs` ヘルパー:
        ```go
        // resolveArgs は引数パターンから category と name を解決する
        func resolveArgs(pattern []string) (category, name string) {
            switch len(pattern) {
            case 0:
                return DefaultCategory, DefaultName
            case 1:
                return pattern[0], "" // category のみ
            default:
                return pattern[0], pattern[1]
            }
        }
        ```
    *   `findEntry` ヘルパー:
        ```go
        // findEntry はエントリリストから category+name で一致するものを返す
        func findEntry(entries []ScaffoldEntry, category, name string) (*ScaffoldEntry, error)
        ```
    *   `List` 関数の変更:
        ```go
        func List(repoURL string, repoRoot string, filterCategory string) ([]ScaffoldEntry, error) {
            // 1. meta.yaml を取得して updated_at を確認
            // 2. キャッシュを確認（CacheStore.Load）
            // 3. キャッシュ有効 → キャッシュ使用
            // 4. キャッシュ無効 → catalog.yaml ダウンロード → ParseCatalogIndex → キャッシュ保存
            // 5. filterCategory が指定されていれば該当カテゴリのみフィルタ
            // 6. 各 ref の ScaffoldDetail を fetch して全エントリを返す
        }
        ```

---

#### [MODIFY] [catalog.go](file://features/tt/internal/scaffold/catalog.go)
*   **Description**: 変更なし。`ParseCatalogIndex`, `ParseScaffoldDetail`, `ResolveFromIndex` は `List`/フォールバック用として残す。

---

### テスト (`tests/integration-test/`)

#### [MODIFY] [tt_scaffold_test.go](file://tests/integration-test/tt_scaffold_test.go)
*   **Description**: `category` のみ指定（`--list` なし）でのエラーテスト追加
*   **Technical Design**:
    ```go
    func TestScaffoldCategoryOnlyError(t *testing.T)  // category のみ → エラー
    ```

---

### コマンド (`features/tt/cmd/`)

#### [MODIFY] [scaffold.go](file://features/tt/cmd/scaffold.go)
*   **Description**: `List` 関数のシグネチャ変更に合わせて呼び出し側を更新（`repoRoot` 引数追加）
*   **Logic**: `--list` + 引数1つの場合は `filterCategory` としてセット

## Step-by-Step Implementation Guide

### Phase 1: シャーディングパス算出（TDD）

- [x] 1. `shard_test.go` を作成（4つの計算例テスト）
- [x] 2. ビルド → テスト失敗を確認
- [x] 3. `shard.go` を実装（`ShardPath`, `DefaultCategory`, `DefaultName`, `KnownCategories`）
- [x] 4. ビルド → テスト成功を確認

### Phase 2: `.gitignore` ユーティリティ（TDD）

- [x] 5. `gitignore_test.go` を作成（9テスト）
- [x] 6. ビルド → テスト失敗を確認
- [x] 7. `gitignore.go` を実装
- [x] 8. ビルド → テスト成功を確認
- [x] 9. `applier.go` の `applyGitignoreEntries` を `gitignore.go` ユーティリティに置き換え
- [x] 10. ビルド → 既存テスト (`TestApplyGitignore_*`) がパスすることを確認

### Phase 3: キャッシュ機能（TDD）

- [x] 11. `cache_test.go` を作成（4テスト）
- [x] 12. ビルド → テスト失敗を確認
- [x] 13. `cache.go` を実装
- [x] 14. ビルド → テスト成功を確認

### Phase 4: scaffold.go の方式A 対応

- [x] 15. `scaffold.go` の `fetchAndResolveEntry` を方式A に書き換え
- [x] 16. `resolveArgs` と `findEntry` ヘルパーを追加
- [x] 17. `List` 関数をキャッシュ付き処理に変更
- [x] 18. `features/tt/cmd/scaffold.go` の呼び出し側を更新
- [x] 19. ビルド → 単体テスト成功を確認

### Phase 5: 統合テスト

- [x] 21. 統合テスト実行 → 全 `TestScaffold*` テストがパスすることを確認
- [x] 22. 全テストリグレッション確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**:
        - `TestShardPath` (4ケース) パス
        - `TestGitignoreAddEntries*` (6テスト) パス
        - `TestCacheStore*` (4テスト) パス
        - 既存テスト `TestApplyGitignore_*` (4テスト) がリグレッションなしでパス
        - 既存テスト `TestParseCatalogIndex*`, `TestParseScaffoldDetail*` パス

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffold"
    ```
    *   **確認事項**:
        - `TestScaffoldDefault` — デフォルト root/default テンプレートが方式A で適用
        - `TestScaffoldList` — `catalog.yaml` キャッシュ経由でテンプレート一覧が表示
        - `TestScaffoldDefaultLocaleJa` — 日本語ロケール
        - `TestScaffoldCwdFlag` — CWD モード
        - `TestScaffoldCategoryOnlyError` — category のみ指定でエラー
    *   **Log Verification**: `catalog.yaml` がダウンロードされず、シャーディング YAML が直接フェッチされていること（`--list` 以外）

## Documentation

#### [MODIFY] [001-Remove-CatalogYaml-Dependency.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/001-Remove-CatalogYaml-Dependency.md)
*   **更新内容**: 実装完了後に検証結果セクションを追加
