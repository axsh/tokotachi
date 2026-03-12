# 000-Fix-Scaffold-CatalogParsing

> **Source Specification**: [000-Fix-Scaffold-CatalogParsing.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/000-Fix-Scaffold-CatalogParsing.md)

## Goal Description

リモートの `tokotachi-scaffolds` リポジトリにおける `catalog.yaml` のフォーマット変更に対応し、`tt scaffold` コマンドが正しく動作するよう修正する。主な変更点は、カタログのインデックス形式パース、個別scaffold YAMLの読み込み、ZIPテンプレートの展開対応、`template_params` への対応。

## User Review Required

> [!IMPORTANT]
> - **Placement 定義の廃止**: 新フォーマットには `placement_ref` が存在しない。ZIPファイル内にplacement定義が含まれているか、あるいはクライアント側でデフォルトplacement（全ファイルをルート相対で配置）を生成するか、の判断が必要。本計画では **ZIP内にplacementが存在しない前提で、デフォルトplacement を自動生成する方針** とする。
> - **`depends_on` の扱い**: 今回のスコープでは依存関係チェックは実装しない（パース・保持のみ）。
> - **`old_value` の扱い**: `template_params` の `old_value` フィールドは、テンプレート内で旧値を新値に置換するために使用される可能性がある。今回はパース・保持のみとし、テンプレート置換ロジックの実装は別途検討。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| カタログパーサーの新フォーマット対応 | Proposed Changes > `catalog.go` |
| 個別 scaffold YAML の読み込み | Proposed Changes > `catalog.go`, `scaffold.go` |
| `ScaffoldEntry` 型の更新 | Proposed Changes > `catalog.go` |
| テンプレートの ZIP 対応 | Proposed Changes > `zip.go` (NEW), `scaffold.go` |
| デフォルト scaffold の解決 | Proposed Changes > `catalog.go` |
| `--list` の動作 | Proposed Changes > `scaffold.go` |
| 既存フラグの動作維持 | Verification Plan > Integration Tests |

## Proposed Changes

### scaffold パッケージ (`features/tt/internal/scaffold/`)

---

#### [MODIFY] [catalog_test.go](file://features/tt/internal/scaffold/catalog_test.go)
*   **Description**: 新フォーマット対応のテストを追加（TDD: テスト先行）
*   **Technical Design**:
    *   `TestParseCatalogIndex` — 新形式インデックスのパーステスト
    *   `TestParseCatalogIndex_InvalidYAML` — 異常系
    *   `TestParseScaffoldDetail` — 個別YAML のパーステスト（`depends_on`, `template_params` 含む）
    *   `TestResolveFromIndex_Default` — デフォルトエントリの解決
    *   `TestResolveFromIndex_ByName` — 名前指定
    *   `TestResolveFromIndex_ByCategory` — カテゴリのみ指定（複数返却）
    *   `TestResolveFromIndex_ByCategoryAndName` — カテゴリ＋名前指定
    *   `TestResolveFromIndex_NotFound` — 存在しないパターン指定
*   **Logic**:
    *   各テストで入力YAMLを文字列リテラルとして定義
    *   `TestParseCatalogIndex`: インデックスYAMLを `ParseCatalogIndex` に渡し、`CatalogIndex.Scaffolds` が正しいカテゴリ・名前・refのマップであることを検証
    *   `TestParseScaffoldDetail`: 個別YAMLを `ParseScaffoldDetail` に渡し、`ScaffoldEntry` のフィールド（`DependsOn`, `TemplateParams`）が正しいことを検証
    *   `TestResolveFromIndex_*`: `CatalogIndex.ResolveFromIndex(pattern)` の結果を検証

---

#### [MODIFY] [catalog.go](file://features/tt/internal/scaffold/catalog.go)
*   **Description**: 新フォーマット対応の型・パーサー・解決ロジックを追加
*   **Technical Design**:
    *   新しい型:
        ```go
        // CatalogIndex は新しいインデックス形式の catalog.yaml を表す
        type CatalogIndex struct {
            // Scaffolds は category -> name -> ref の2段マップ
            Scaffolds map[string]map[string]string `yaml:"scaffolds"`
        }

        // Dependency はscaffoldの依存関係を表す
        type Dependency struct {
            Category string `yaml:"category"`
            Name     string `yaml:"name"`
        }

        // ScaffoldDetail は個別scaffold YAMLの構造
        type ScaffoldDetail struct {
            Scaffolds []ScaffoldEntry `yaml:"scaffolds"`
        }
        ```
    *   `ScaffoldEntry` の拡張:
        ```go
        type ScaffoldEntry struct {
            Name           string       `yaml:"name"`
            Category       string       `yaml:"category"`
            Description    string       `yaml:"description"`
            TemplateRef    string       `yaml:"template_ref"`
            OriginalRef    string       `yaml:"original_ref"`
            DependsOn      []Dependency `yaml:"depends_on"`
            TemplateParams []Option     `yaml:"template_params"`
            // 後方互換フィールド（旧フォーマット用）
            PlacementRef   string       `yaml:"placement_ref"`
            Requirements   Requirements `yaml:"requirements"`
            Options        []Option     `yaml:"options"`
        }
        ```
    *   新関数:
        ```go
        // ParseCatalogIndex は新形式のインデックスYAMLをパースする
        func ParseCatalogIndex(data []byte) (*CatalogIndex, error)

        // ParseScaffoldDetail は個別scaffold YAMLをパースする
        func ParseScaffoldDetail(data []byte) ([]ScaffoldEntry, error)

        // ResolveFromIndex はインデックスからパターンに一致するrefを返す
        // pattern:
        //   nil/空 → "root"/"default" をデフォルトとして返す
        //   1要素 → 名前一致を試み、なければカテゴリ一致
        //   2要素 → [0]=category, [1]=name で完全一致
        // 戻り値: []IndexRef (category, name, ref のスライス)
        func (idx *CatalogIndex) ResolveFromIndex(pattern []string) ([]IndexRef, error)
        ```
    *   `IndexRef` 型:
        ```go
        type IndexRef struct {
            Category string
            Name     string
            Ref      string // 個別 YAML ファイルのパス
        }
        ```
*   **Logic**:
    *   `ParseCatalogIndex`: `yaml.Unmarshal` で `CatalogIndex` にデコード。`Scaffolds` が空の場合エラー。
    *   `ParseScaffoldDetail`: `yaml.Unmarshal` で `ScaffoldDetail` にデコードし、`.Scaffolds` を返す。
    *   `ResolveFromIndex`:
        1. `pattern` が空: `Scaffolds["root"]["default"]` を返す。存在しなければエラー。
        2. `pattern` が1要素: 全カテゴリの全名前を走査し名前一致を検索。見つからなければカテゴリ一致（そのカテゴリの全エントリを返す）。
        3. `pattern` が2要素: `Scaffolds[pattern[0]][pattern[1]]` を検索。
    *   既存の `ParseCatalog`, `Catalog`, `ResolvePattern` は変更せず残す（旧フォーマットの後方互換）。

---

#### [NEW] [zip_test.go](file://features/tt/internal/scaffold/zip_test.go)
*   **Description**: ZIPバイト列から `[]DownloadedFile` への変換テスト（TDD: テスト先行）
*   **Technical Design**:
    *   `TestExtractZip_BasicFiles` — 複数ファイルを含むZIPを展開
    *   `TestExtractZip_WithSubdirectories` — サブディレクトリ含むZIP
    *   `TestExtractZip_EmptyZip` — 空ZIP でエラーなく空スライスが返ること
    *   `TestExtractZip_InvalidData` — 不正データでエラー
*   **Logic**:
    *   テスト内で `archive/zip` を使用してZIPバイト列を動的に生成
    *   `ExtractZip(data []byte)` を呼び出し、`[]DownloadedFile` の内容を検証

---

#### [NEW] [zip.go](file://features/tt/internal/scaffold/zip.go)
*   **Description**: ZIPダウンロード・展開機能
*   **Technical Design**:
    ```go
    // ExtractZip はZIPバイト列を展開し、DownloadedFileスライスとして返す
    // ディレクトリエントリは無視し、ファイルのみを返す
    func ExtractZip(data []byte) ([]DownloadedFile, error)
    ```
*   **Logic**:
    1. `bytes.NewReader(data)` → `zip.NewReader` で ZIP リーダーを作成
    2. 各エントリについて:
        - ディレクトリ（末尾 `/`）はスキップ
        - ファイルの場合は `RelativePath` と `Content` を読み込み、`DownloadedFile` として収集
    3. ファイルが0件でもエラーにしない（空のテンプレートの可能性）

---

#### [MODIFY] [scaffold.go](file://features/tt/internal/scaffold/scaffold.go)
*   **Description**: `Run` と `Apply` と `List` を新フォーマットに対応させる
*   **Technical Design**:
    *   `Run` 関数の変更:
        1. `ParseCatalog` → `ParseCatalogIndex` に変更
        2. `catalog.ResolvePattern` → `catalogIndex.ResolveFromIndex` に変更
        3. 返されたrefから `downloader.FetchFile(ref)` で個別YAMLを取得
        4. `ParseScaffoldDetail` でエントリ詳細を取得
        5. テンプレート取得: `entry.TemplateRef` が `.zip` で終わる場合は `downloader.FetchFile` で ZIPダウンロード → `ExtractZip` で展開。そうでなければ従来の `FetchDirectory` を使用。
        6. `entry.TemplateParams` が存在する場合は `CollectOptionValues` に渡す（`entry.Options` の代わりに）。両方存在する場合は `TemplateParams` を優先。
        7. Placement: `entry.PlacementRef` が空の場合はデフォルト `Placement` を生成（`BaseDir: ""`, `ConflictPolicy: "skip"`, 空の `FileMappings`）
    *   `Apply` 関数の変更:
        - `Run` と同様に `ParseCatalogIndex` + `ParseScaffoldDetail` + ZIP対応のフローに変更
        - `entry.PlacementRef` が空の場合のデフォルトplacement 対応
    *   `List` 関数の変更:
        - `ParseCatalogIndex` でインデックスを取得
        - 全refをループして個別YAMLをfetch + parse
        - 全エントリを `[]ScaffoldEntry` として返す
*   **Logic**:
    *   テンプレートファイル取得の分岐:
        ```go
        var templateFiles []DownloadedFile
        if strings.HasSuffix(entry.TemplateRef, ".zip") {
            zipData, err := downloader.FetchFile(entry.TemplateRef)
            // ... error handling
            templateFiles, err = ExtractZip(zipData)
            // ... error handling
        } else {
            templateFiles, err = downloader.FetchDirectory(entry.TemplateRef + "/base")
            // ... error handling
        }
        ```
    *   デフォルトPlacement生成:
        ```go
        func defaultPlacement() *Placement {
            return &Placement{
                ConflictPolicy: "skip",
            }
        }
        ```
    *   Options の統合:
        ```go
        // TemplateParams を Options として扱う
        options := entry.Options
        if len(entry.TemplateParams) > 0 {
            options = entry.TemplateParams
        }
        ```

---

### 統合テスト (`tests/integration-test/`)

#### [MODIFY] [tt_scaffold_test.go](file://tests/integration-test/tt_scaffold_test.go)
*   **Description**: 新フォーマットに対応したテスト期待値の更新
*   **Technical Design**:
    *   `TestScaffoldDefault`: ZIPテンプレートから展開される実際のファイル構造に合わせて `expectedFiles` を更新する必要がある可能性
    *   `TestScaffoldList`: 新しいエントリ名（`axsh-go-standard` 等）を含む出力を検証
    *   `TestScaffoldDefaultLocaleJa`: ZIPベースのロケール対応を確認（ZIPにロケールが含まれない場合はテスト内容を調整）
    *   `TestScaffoldCwdFlag`: 基本的にロジック変更なし（パスの期待値変更は可能性あり）
*   **Logic**:
    *   まずビルド＋既存テストを実行して、新しいテンプレートから生成されるファイル構造を確認してからテスト期待値を調整する（段階的アプローチ）

## Step-by-Step Implementation Guide

### Phase 1: 単体テスト・コア型の追加

1.  **catalog_test.go にテストケースを追加**:
    *   `TestParseCatalogIndex`, `TestParseScaffoldDetail`, `TestResolveFromIndex_*` のテストを追加
    *   ビルド実行 → テスト失敗を確認 (Failed First)

2.  **catalog.go に新型・新関数を追加**:
    *   `CatalogIndex`, `ScaffoldDetail`, `Dependency`, `IndexRef` 型を追加
    *   `ParseCatalogIndex`, `ParseScaffoldDetail`, `ResolveFromIndex` 関数を実装
    *   ビルド実行 → 単体テスト成功を確認

### Phase 2: ZIP 展開機能

3.  **zip_test.go を作成**:
    *   `TestExtractZip_*` テストを追加
    *   ビルド実行 → テスト失敗を確認 (Failed First)

4.  **zip.go を作成**:
    *   `ExtractZip` 関数を実装
    *   ビルド実行 → 単体テスト成功を確認

### Phase 3: scaffold.go の更新

5.  **scaffold.go の `Run` 関数を更新**:
    *   `ParseCatalogIndex` → `ResolveFromIndex` → `FetchFile(ref)` → `ParseScaffoldDetail` のフロー実装
    *   ZIP テンプレートダウンロード・展開の分岐実装
    *   `TemplateParams` / `Options` の統合
    *   デフォルト Placement 生成

6.  **scaffold.go の `Apply` 関数を更新**:
    *   `Run` と同様の新フォーマットフローを実装

7.  **scaffold.go の `List` 関数を更新**:
    *   全インデックスrefを巡回して全エントリを収集するロジック

8.  **ビルドと単体テスト実行**:
    ```bash
    ./scripts/process/build.sh
    ```

### Phase 4: 統合テスト・調整

9.  **統合テスト実行**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffold"
    ```

10. **テスト期待値の調整**:
    *   テスト結果に応じて `tt_scaffold_test.go` の期待値を更新
    *   ロケール対応がZIPに含まれない場合の対応

11. **全テストを再実行してリグレッションを確認**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**: 全単体テストが成功すること。特に以下のテスト:
        - `TestParseCatalogIndex`
        - `TestParseCatalogIndex_InvalidYAML`
        - `TestParseScaffoldDetail`
        - `TestResolveFromIndex_Default`
        - `TestResolveFromIndex_ByName`
        - `TestResolveFromIndex_ByCategory`
        - `TestResolveFromIndex_ByCategoryAndName`
        - `TestResolveFromIndex_NotFound`
        - `TestExtractZip_BasicFiles`
        - `TestExtractZip_WithSubdirectories`
        - `TestExtractZip_EmptyZip`
        - `TestExtractZip_InvalidData`

2.  **Integration Tests (Scaffold)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestScaffold"
    ```
    *   **確認事項**: 以下の4テストが全て成功すること:
        - `TestScaffoldDefault` — デフォルトテンプレートの適用
        - `TestScaffoldList` — テンプレート一覧表示
        - `TestScaffoldDefaultLocaleJa` — 日本語ロケール
        - `TestScaffoldCwdFlag` — CWDフラグの動作

3.  **Full Test Suite**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   **確認事項**: 他の既存テストにリグレッションが発生しないこと

## Documentation

#### [MODIFY] [000-Fix-Scaffold-CatalogParsing.md](file://prompts/phases/000-foundation/ideas/fix-scaffolds/000-Fix-Scaffold-CatalogParsing.md)
*   **更新内容**: 実装完了後に検証結果セクションを追加
