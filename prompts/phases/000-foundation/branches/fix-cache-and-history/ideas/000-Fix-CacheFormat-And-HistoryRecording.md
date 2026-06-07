# キャッシュフォーマットの改善・履歴テスト補強・`--root` オプション追加

## 背景 (Background)

### 課題 1: キャッシュフォーマットの非効率性

現在、`.kotoshiro/tokotachi/.cache/catalog.yaml` にダウンロードしたカタログデータを保存する際、`[]byte` 型のデータを YAML の `catalog_data` フィールドとして直接エンコードしている（[cache.go](file://pkg/scaffold/cache.go) の `CachedCatalog` 構造体）。

YAML で `[]byte` をエンコードすると、各バイトが10進数のリストとして表現されるため、1バイトのデータに対して5〜7バイトを消費する。将来的にカタログデータが大きくなると、キャッシュファイルのサイズが実体の5〜7倍に膨れ上がる問題がある。

**現在の構造:**
```yaml
# .kotoshiro/tokotachi/.cache/catalog.yaml
updated_at: "2026-03-10T19:00:00+09:00"
catalog_data:
  - 115
  - 99
  - 97
  - ...
```

### 課題 2: ダウンロード履歴のテスト不足

`downloaded.yaml` の記録について、依存チェーン展開時のテストカバレッジが存在しない。

**コード調査結果:**
[applyDependencyChain()](file://pkg/scaffold/scaffold.go#L296-L359) は各 scaffold ごとに `fetchTemplateAndPlacement()` → `IsDynamic(placement)` で個別に判定しており、コード上は正しく動作する。ただし、これを検証するテストが存在しないため、テストを補強する。

### 課題 3: `--cwd` オプションの使い勝手改善

現在の `--cwd` フラグ（[scaffold.go](file://features/tt/cmd/scaffold.go) L43）は bool 型で、現在のワーキングディレクトリをルートとして使用する。しかし、事前に `cd` でディレクトリを移動する必要があり使い勝手が悪い。

**現在の動作 ([resolveRepoRoot](file://features/tt/cmd/scaffold.go#L153-L166)):**
- `--cwd` なし: `git rev-parse --show-toplevel` で Git ルートを自動検出、失敗時は `os.Getwd()`
- `--cwd` あり: `os.Getwd()` を使用

**要望:** `--root {path}` に変更し、任意のパスをルートディレクトリとして指定できるようにする。

---

## 要件 (Requirements)

### 必須要件

#### R1: キャッシュフォーマットの改善

`.cache/` 以下にディレクトリベースのキャッシュ構造を導入し、キャッシュデータを無加工のバイナリファイルとして保存する。

**新しいディレクトリ構造:**
```
.kotoshiro/tokotachi/.cache/
└── repository_data/        # カテゴリ名 (キャッシュの種類)
    └── catalog.yaml/       # キャッシュ対象ファイル名のフォルダ
        ├── meta.yaml       # メタ情報 (updated_at, cached_at)
        └── data            # ダウンロードしたファイルの実体 (無加工)
```

- **カテゴリ名**: `repository_data`（他のキャッシュと衝突しない命名）
- **`meta.yaml`**: `updated_at`（リモートのタイムスタンプ）と `cached_at`（キャッシュ日時）を格納
- **`data`**: ダウンロードしたファイルをそのまま保存（一切のエンコーディングなし）

#### R2: ダウンロード履歴のテスト補強

依存チェーン展開時の `downloaded.yaml` 記録が正しく動作することを検証するテストを追加する。

- 静的 scaffold は `downloaded.yaml` に記録されること
- 動的 scaffold（`BaseDir` に `{{...}}` を含む）は記録されないこと
- 依存チェーン内で静的/動的が混在する場合、個別に正しく判定されること

#### R3: `--cwd` を `--root {path}` に変更

- `--cwd`（bool型フラグ）を削除
- `--root {path}`（string型フラグ）を追加
- **`--root` 指定あり**: 指定されたパスをルートディレクトリとして使用
- **`--root` 指定なし**: 従来通り `git rev-parse --show-toplevel` で Git ルートを自動検出、失敗時は `os.Getwd()`

---

## 実現方針 (Implementation Approach)

### 修正対象ファイル

#### 1. キャッシュフォーマットの改善

- **`pkg/scaffold/cache.go`**: `CacheStore` のパス構造をディレクトリベースに変更
  - `CachedCatalog` → `CacheMeta`（`meta.yaml` 用）に変更
  - `Save()` → `meta.yaml` + `data` ファイルを別々に保存
  - `Load()` → `meta.yaml` + `data` ファイルを読み込み
  - `IsValid()` → `meta.yaml` の `updated_at` で判定
- **`pkg/scaffold/cache_test.go`**: テストを新構造に対応
- **`pkg/scaffold/scaffold.go`**: `List()` 関数のキャッシュ利用部分を修正

#### 2. ダウンロード履歴のテスト補強

- **`tests/integration-test/tt_scaffold_test.go`**: `TestScaffoldWithDependencies` に `downloaded.yaml` 検証サブテストを追加

#### 3. `--root` オプション

- **`features/tt/cmd/scaffold.go`**:
  - `scaffoldFlagCwd bool` → `scaffoldFlagRoot string` に変更
  - `--cwd` フラグ定義を `--root` に変更
  - `resolveRepoRoot()` のシグネチャを `resolveRepoRoot(rootPath string) string` に変更
- **`tests/integration-test/tt_scaffold_test.go`**: `TestScaffoldCwdFlag` を `TestScaffoldRootFlag` に変更

---

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: キャッシュフォーマット

1. `tt scaffold --list` を実行
2. `.kotoshiro/tokotachi/.cache/repository_data/catalog.yaml/` ディレクトリ構造を確認
3. `meta.yaml` のフォーマットと `data` の内容が正しいことを確認
4. 再度 `tt scaffold --list` を実行し、キャッシュヒットすることを確認

### シナリオ 2: ダウンロード履歴

1. `tt scaffold feature axsh-go-standard --yes --default --v feature_name=testfeature` を実行
2. `downloaded.yaml` に静的 scaffold が記録されていることを確認
3. 動的 scaffold は記録されていないことを確認

### シナリオ 3: `--root` オプション

1. `tt scaffold --root /tmp/myproject --yes` を実行
2. `/tmp/myproject` 配下にファイルが展開されることを確認
3. `--root` なしで従来通り Git ルート自動検出が動作することを確認

---

## テスト項目 (Testing for the Requirements)

### 単体テスト

| 要件 | テスト対象 | 検証方法 |
|------|-----------|---------|
| R1 | `CacheStore.Save` / `Load` | `meta.yaml` と `data` ファイルの保存・読み込み |
| R1 | `CacheStore.IsValid` | `meta.yaml` の `updated_at` による有効性判定 |
| R1 | 旧キャッシュとの互換性 | 旧フォーマット存在時に新規キャッシュを作成 |

```bash
./scripts/process/build.sh
```

### 統合テスト

| 要件 | テスト対象 | 検証方法 |
|------|-----------|---------|
| R1 | キャッシュ構造 | `tt scaffold --list` 後のディレクトリ構造検証 |
| R2 | 依存チェーンの履歴記録 | `TestScaffoldWithDependencies` 拡張 |
| R3 | `--root` フラグ | `TestScaffoldRootFlag`（旧 `TestScaffoldCwdFlag` を改修） |

```bash
./scripts/process/integration_test.sh
```
