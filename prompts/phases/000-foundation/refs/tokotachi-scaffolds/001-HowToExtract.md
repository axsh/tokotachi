# テンプレート抽出方法について

本ドキュメントは、リポジトリを参照するクライアント（`tt scaffold`）が、どのようにリポジトリをテンプレートとして利用すれば良いかを説明したものです。このドキュメントを読めば、ファイルのアクセス順、解析方法、テンプレートファイルの抽出と適用方法が理解できます。

---

## 概要

scaffold の取得には**2つの方式**があります。

### 方式A: ダイレクトアクセス（推奨）

ハッシュ計算により、`catalog.yaml` のダウンロードなしにシャーディングファイルへ直接到達します。scaffold の数が増えても API アクセス回数は一定（最小2回: meta.yaml + シャーディング YAML）です。

```
meta.yaml → ハッシュ計算 → シャーディング YAML → ZIP
（API 3回、category + name が既知なら meta.yaml も省略可能で 2回）
```

### 方式B: インデックス経由（フォールバック）

ハッシュ計算を実装できないクライアント向けに、`catalog.yaml`（インデックス）を利用してシャーディングパスを検索します。

```
meta.yaml → catalog.yaml → シャーディング YAML → ZIP
（API 4回）
```

> **Note**: `catalog.yaml` は補助用途のインデックスファイルです。将来 scaffold が膨大になった場合、ダウンロードと検索のコストが増加するため、方式A を推奨します。

---

## シャーディングパスの算出アルゴリズム

クライアントが `category` と `name` からシャーディングファイルのパスを直接算出できます。

### 入力

- `category`: scaffold のカテゴリ（例: `"feature"`）
- `name`: scaffold の名前（例: `"axsh-go-standard"`）

### アルゴリズム

```
1. key = category + "/" + name
   例: "feature/axsh-go-standard"

2. hash32 = FNV_1a_32(key)
   FNV-1a 32ビットハッシュを計算
   （offset_basis = 2166136261, prime = 16777619）

3. reduced = hash32 % 1679616
   1679616 = 36^4（ハッシュ空間の制限）

4. encoded = base36(reduced)
   0-9a-z の36進数でエンコード、4文字に0パディング

5. path = "catalog/scaffolds/{encoded[0]}/{encoded[1]}/{encoded[2]}/{encoded[3]}.yaml"
   例: "catalog/scaffolds/b/i/b/l.yaml"
```

### 疑似コード

```
function scaffold_shard_path(category, name):
    key = category + "/" + name

    # FNV-1a 32-bit
    hash = 2166136261  # offset basis
    for byte in key:
        hash = hash XOR byte
        hash = hash * 16777619
        hash = hash AND 0xFFFFFFFF  # 32-bit mask

    # Reduce to 36^4 space and encode
    reduced = hash % 1679616
    chars = "0123456789abcdefghijklmnopqrstuvwxyz"
    encoded = ""
    for i in 0..3:
        encoded = chars[reduced % 36] + encoded
        reduced = reduced / 36  # integer division

    return "catalog/scaffolds/" + encoded[0] + "/" + encoded[1] + "/" + encoded[2] + "/" + encoded[3] + ".yaml"
```

### 計算例

| category | name | key | ハッシュ | パス |
|---|---|---|---|---|
| root | default | `root/default` | `6jvn` | `catalog/scaffolds/6/j/v/n.yaml` |
| feature | axsh-go-standard | `feature/axsh-go-standard` | `bibl` | `catalog/scaffolds/b/i/b/l.yaml` |

---

## Step 1: メタデータの取得

リポジトリルートの `meta.yaml` を取得します。

```
GET /repos/{owner}/{repo}/contents/meta.yaml
```

```yaml
version: "1.0.0"
default_scaffold: "default"
updated_at: "2026-03-10T19:00:00+09:00"
```

- `updated_at` はキャッシュ判定に使用
- `default_scaffold` はユーザーが名前を指定しなかった場合のデフォルト

---

## Step 2: シャーディングファイルの取得

### 方式A: ダイレクトアクセス（推奨）

`category` と `name` からハッシュ計算でパスを算出し、直接取得します。

```
# 例: category="feature", name="axsh-go-standard"
# → ハッシュ計算 → "catalog/scaffolds/b/i/b/l.yaml"

GET /repos/{owner}/{repo}/contents/catalog/scaffolds/b/i/b/l.yaml
```

### 方式B: インデックス経由（フォールバック）

ハッシュ計算を実装できない場合、`catalog.yaml` をダウンロードしてパスを検索します。

```
GET /repos/{owner}/{repo}/contents/catalog.yaml
```

```yaml
scaffolds:
  root:
    default: "catalog/scaffolds/6/j/v/n.yaml"
  project:
    axsh-go-standard: "catalog/scaffolds/8/w/4/o.yaml"
  feature:
    axsh-go-standard: "catalog/scaffolds/b/i/b/l.yaml"
    axsh-go-kotoshiro-mcp: "catalog/scaffolds/i/4/2/h.yaml"
```

`scaffolds[category][name]` でパスを取得し、そのパスのファイルをダウンロードします。

### スキャフォールドの特定

| 入力 | 方式A（ダイレクト） | 方式B（インデックス） |
|---|---|---|
| `<category> <name>` | ハッシュ計算で直接取得 | `scaffolds[category][name]` |
| `<name>` のみ | category ごとにハッシュ計算を試行 | 全 category から検索 |
| 引数なし | `meta.yaml` の `default_scaffold` を使用 | 同左 |
| `<category>` のみ | インデックスが必要（方式B にフォールバック） | `category` 配下の全候補を返す |

### シャーディングファイルの内容

```yaml
scaffolds:
  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    depends_on:
      - category: "project"
        name: "axsh-go-standard"
    template_ref: "catalog/scaffolds/b/i/b/go-standard-feature.zip"
    original_ref: "catalog/originals/axsh/go-standard-feature"
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        default: "github.com/axsh/tokotachi/features/myprog"
      - name: "program_name"
        description: "Program name"
        required: true
        default: "myprog"
```

- シャーディングファイルは配列形式（ハッシュ衝突時に複数エントリ格納）
- `category` + `name` でフィルタして目的のエントリを取得

---

## Step 3: テンプレートファイルの取得

### 3.1 ZIP アーカイブのダウンロード

シャーディングファイル内の `template_ref` フィールドのパスから ZIP をダウンロードします。

```yaml
# シャーディングファイルのエントリ例
template_ref: "catalog/scaffolds/b/i/b/go-standard-feature.zip"
```

```
GET /repos/{owner}/{repo}/contents/catalog/scaffolds/b/i/b/go-standard-feature.zip
```

### 3.2 ZIP の展開

ダウンロードした ZIP をテンポラリディレクトリに展開します。ZIP 内には `base/`（ベースファイル群）、`locale.<lang>/`（ロケールオーバーレイ、任意）、`scaffold.yaml` が含まれます。

```
# ZIP 展開後の構造例（root/project-default の場合）
base/
  AGENTS.md
  features/
    README.md
  prompts/
    phases/
      000-foundation/
        ideas/.gitkeep
        plans/.gitkeep
      README.md
    rules/.gitkeep
  scripts/.gitkeep
  shared/
    README.md
    libs/README.md
  work/
    README.md
locale.ja/
  features/
    README.md
  prompts/
    phases/
      README.md
  shared/
    README.md
    libs/README.md
  work/
    README.md
scaffold.yaml
```

> **Note**: ZIP 内の `scaffold.yaml` は scaffold 定義のコピーです。クライアントはシャーディング YAML から情報を取得済みのため、この `scaffold.yaml` は展開対象から除外してください。

### 3.3 ロケールオーバーレイの適用

ZIP 展開後、以下の手順でロケールに応じたファイルセットを作成します。

1. **ロケールの検出**: ユーザーのロケールを検出する（優先順: `--lang` フラグ > `LC_ALL` > `LANG`）
2. **locale ディレクトリの確認**: ZIP 内に `locale.<lang>/` が存在するか確認する
3. **ファイルセットの作成**:
   - **`locale.<lang>/` が存在する場合**:
     - `base/` のファイルを作業ディレクトリにコピー
     - `locale.<lang>/` のファイルで上書き（同名ファイルのみ置換）
   - **`locale.<lang>/` が存在しない場合**:
     - `base/` のファイルのみを作業ディレクトリにコピー
4. 結果として得られたファイル群を以降のテンプレート処理（Step 4）に使用する

---

## Step 4: テンプレート変数の適用

### 4.1 パラメータの収集

`template_params` に定義されたパラメータをユーザーから収集します。

```yaml
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myfunction"
  - name: "program_name"
    description: "Program name"
    required: true
    default: "myfunction"
```

| フィールド | ルール |
|---|---|
| `required: true` | 未指定の場合、CLI でインタラクティブに入力を促す |
| `default` | ユーザーが値を指定しなかった場合に使用 |

### 4.2 `.tmpl` ファイルの処理

拡張子 `.tmpl` を持つファイルをテンプレートとして処理し、パラメータ値を埋め込みます。

```
# 処理前: go.mod.tmpl
module {{module_path}}

go 1.24.0

# 処理後: go.mod  （.tmpl 拡張子は除去）
module github.com/myorg/myapp

go 1.24.0
```

- テンプレート変数は `{{パラメータ名}}` 形式で参照
- 処理後、`.tmpl` 拡張子は除去される

---

## Step 5: 配置ルールの適用

シャーディングファイル内の `placement` セクション（シャーディング YAML に内包）に基づいてファイルを配置します。

### 5.1 配置先の決定

`placement.base_dir` フィールドでファイルの配置先ルートを決定します。テンプレート変数を含めることが可能です。

```yaml
placement:
  base_dir: "features/{{feature_name}}"
```

### 5.2 ファイル名マッピング

`placement.file_mappings` が定義されている場合、テンプレート内のファイル名を実際のファイル名に変換します。

```yaml
placement:
  file_mappings:
    - source: "dot-gitignore"
      target: ".gitignore"
```

### 5.3 コンフリクト解決

`placement.conflict_policy` に基づいて、既存ファイルとの衝突を処理します。

| ポリシー | 動作 |
|---|---|
| `skip` | 既存ファイルはスキップ、新規のみ作成 |
| `overwrite` | 既存ファイルを上書き |
| `append` | 既存ファイルの末尾に追記 |
| `error` | 既存ファイルがあればエラーで中止 |

### 5.4 後処理

`placement.post_actions` に定義されたアクションを実行します。

```yaml
placement:
  post_actions:
    gitignore_entries:              # .gitignore に追記
      - "work/*"
    file_permissions:               # パーミッション設定
      - pattern: "scripts/**/*.sh"
        executable: true
```

---

## 全体フロー図

```
tt scaffold [category] [name]
     │
     ├─ 1. meta.yaml をダウンロード
     │      └─ バージョン、デフォルト名、更新日時を取得
     │
     ├─ 2. シャーディングパスの取得（2方式から選択）
     │      ├─ [方式A] category + name からハッシュ計算で直接算出（推奨）
     │      └─ [方式B] catalog.yaml をダウンロードして検索（フォールバック）
     │
     ├─ 3. シャーディング YAML をダウンロード
     │      └─ scaffolds 配列から category + name でエントリを特定
     │
     ├─ 4. template_ref の ZIP をダウンロード
     │      └─ テンポラリディレクトリに展開
     │
     ├─ 5. ロケールオーバーレイを適用（該当言語があれば）
     │
     ├─ 6. テンプレートパラメータを収集
     │      └─ required=true のパラメータを入力促進
     │
     ├─ 7. .tmpl ファイルにパラメータを適用
     │      └─ {{param}} → 実際の値に置換、.tmpl 拡張子除去
     │
     ├─ 8. placement（内包）から配置ルールを読み込み
     │      └─ base_dir, conflict_policy を取得
     │
     ├─ 9. 実行計画を表示（dry-run）
     │      └─ 追加・スキップ・上書きファイル一覧 + パーミッション変更予定
     │
     ├─ 10. ユーザー確認 [y/N]
     │
     ├─ 11. チェックポイント作成（git stash）
     │
     ├─ 12. ファイル配置
     │       ├─ ファイル名マッピングを適用
     │       └─ conflict_policy に従って配置
     │
     ├─ 13. 後処理
     │       ├─ gitignore_entries を .gitignore に追記
     │       └─ file_permissions を適用
     │
     └─ 完了
```