# tokotachi-scaffolds リファレンスマニュアル

本ドキュメントは `tokotachi-scaffolds` リポジトリにテンプレートを作成する際のリファレンスです。

## リポジトリの目的

`tokotachi-scaffolds` は、`devctl scaffold` コマンドが使用するテンプレートリポジトリです。テンプレートは GitHub Contents API 経由でダウンロードされ、ユーザーのプロジェクトに適用されます。

---

## リポジトリ全体構造

```
tokotachi-scaffolds/
├── catalog.yaml          # カタログ定義（Segment 1）
├── templates/            # テンプレート実体（Segment 2）
│   ├── {template-name}/
│   │   ├── base/         # ベースファイル群
│   │   └── locale.{lang}/# 言語別オーバーレイ（任意）
│   └── ...
└── placements/           # 配置定義（Segment 3）
    ├── {placement-name}.yaml
    └── ...
```

---

## Segment 1: カタログ (`catalog.yaml`)

リポジトリルートに配置する **唯一の** カタログファイル。すべてのテンプレートのメタデータを一元管理します。

### スキーマ

```yaml
version: "1.0.0"

# パターン未指定時に使われるテンプレート名（scaffolds 内の name と一致必須）
default_scaffold: "default"

scaffolds:
  - name: "<テンプレート名>"           # devctl scaffold <name> で指定する名前
    category: "<カテゴリ名>"           # devctl scaffold <category> で指定するカテゴリ
    description: "<説明文>"
    template_ref: "templates/<dir>"   # Segment 2 へのパス
    placement_ref: "placements/<file>.yaml"  # Segment 3 へのパス
    requirements:                     # 前提条件
      directories: []                 # 存在が必要なディレクトリ一覧
      files: []                       # 存在が必要なファイル一覧
    options:                          # テンプレート変数（任意）
      - name: "<変数名>"
        description: "<説明>"
        required: true|false
        default: "<デフォルト値>"
```

### パターン解決ルール

| コマンド | 解決ロジック |
|---|---|
| `devctl scaffold` | `default_scaffold` の値で `name` を検索 |
| `devctl scaffold <name>` | `name` フィールドで完全一致検索 |
| `devctl scaffold <category>` | `category` で検索し候補一覧を返す |
| `devctl scaffold <category> <name>` | `category` + `name` の両方一致で検索 |

### 前提条件 (`requirements`)

- `directories`: 指定パスのディレクトリがプロジェクトに存在しなければエラー
- `files`: 指定パスのファイルがプロジェクトに存在しなければエラー
- 未充足時は `devctl scaffold default` 等を先に実行するようヒントを表示

### テンプレートオプション (`options`)

- `required: true` かつ未指定の場合、CLI 上でインタラクティブに入力を促す
- オプション値は Segment 2 のファイル内容・Segment 3 の `base_dir` で `{{.Name}}` 形式で利用可能
- `options` が不要なテンプレートでは省略可

---

## Segment 2: テンプレート実体 (`templates/`)

### ディレクトリ構造

```
templates/<template-name>/
├── base/               # メインのファイル群（必須）
│   ├── dir1/
│   │   └── file1.md
│   ├── dir2/
│   │   └── file2.yaml.tmpl
│   └── ...
└── locale.<lang>/      # 言語別オーバーレイ（任意）
    └── dir1/
        └── file1.md    # base/ の同名ファイルを上書き
```

### ルール

1. **`base/` は必須**。テンプレートのメインファイル群を格納する
2. **`base/` の言語は自由**。作成者の得意な言語で書いてよい（英語必須ではない）
3. **ディレクトリ構造はそのまま配置先に反映される**（`base/` のルートが `base_dir` に対応）
4. **`.gitkeep` ファイル**で空ディレクトリを表現する

### テンプレート変数（`.tmpl` ファイル）

- 拡張子 `.tmpl` のファイルは Go テンプレートとして処理される
- 処理後、`.tmpl` 拡張子は除去される（`go.mod.tmpl` → `go.mod`）
- 利用可能な変数は `catalog.yaml` の `options` で定義されたもの

```
# 例: go.mod.tmpl
module {{.GoModule}}

go 1.24.0
```

### ロケールオーバーレイ

- `locale.<lang>/` ディレクトリに、`base/` と同じ相対パスで差分ファイルを置く
- `devctl` はユーザーのロケールを検出し（`LANG`, `LC_ALL`, `--lang` フラグ）、該当する `locale.<lang>/` があればそのファイルで `base/` のファイルを上書き
- `locale.<lang>/` が存在しない場合は `base/` のみが使用される
- **ファイル単位のオーバーレイ**: `locale.<lang>/` に存在しないファイルは `base/` のものがそのまま使われる

```
# 解決順序の例（locale=ja の場合）:
# locale.ja/features/README.md が存在する → locale.ja 版を使用
# locale.ja/scripts/.gitkeep が存在しない → base/ 版を使用
```

---

## Segment 3: 配置定義 (`placements/`)

### スキーマ

```yaml
version: "1.0.0"

# 配置先ベースディレクトリ（リポジトリルートからの相対パス）
# テンプレート変数を含めることが可能: "features/{{.Name}}"
base_dir: "."

# コンフリクト解決ポリシー
conflict_policy: "skip"    # skip | overwrite | append | error

# テンプレート処理設定
template_config:
  template_extension: ".tmpl"   # テンプレートファイルの拡張子
  strip_extension: true         # 処理後に拡張子を除去するか

# ファイル名マッピング（任意）
file_mappings:
  - source: "dot-gitignore"     # テンプレート内のファイル名
    target: ".gitignore"        # 実際のファイル名

# 後処理アクション
post_actions:
  gitignore_entries:            # .gitignore に追記するエントリ
    - "work/*"
  file_permissions:             # ファイルパーミッション設定
    - pattern: "scripts/**/*.sh"
      executable: true
```

### コンフリクト解決ポリシー

| ポリシー | 動作 | 主な用途 |
|---|---|---|
| `skip` | 既存ファイルはスキップ、新規のみ作成 | デフォルト構成（冪等実行可能） |
| `overwrite` | 既存ファイルを上書き | テンプレート更新の強制適用 |
| `append` | 既存ファイルの末尾に追記 | 設定ファイルへの追記 |
| `error` | 既存ファイルがあればエラーで中止 | 新規作成専用テンプレート |

### ファイル名マッピング (`file_mappings`)

- `.gitignore` のようなドットファイルをテンプレート内で `dot-gitignore` として管理し、配置時に `.gitignore` にリネーム
- 通常は不要（ファイル名がそのまま使える場合は省略可）

### 後処理アクション (`post_actions`)

- `gitignore_entries`: `.gitignore` に追記するパターン。既存エントリと重複する場合はスキップ
- `file_permissions`: ファイルパーミッションの設定ルール（配列）
  - `pattern`: グロブパターン（`**` による再帰マッチング対応）
  - `executable`: `true` に設定すると `0755` パーミッションを付与（`mode` の糖衣構文）
  - `mode`: 8進数文字列（例: `"0600"`, `"0644"`, `"0755"`）で明示的にパーミッションを設定
  - `executable` と `mode` の両方が指定された場合、`mode` が優先される
  - どちらか一方は必ず指定する必要がある

```yaml
post_actions:
  gitignore_entries:
    - "work/*"
  file_permissions:
    - pattern: "scripts/**/*.sh"   # スクリプトファイルに実行権限を付与
      executable: true
    - pattern: "secrets/**/*"      # シークレットファイルのアクセスを制限
      mode: "0600"
```

> **注意**: Windows 環境では `os.Chmod` による Unix パーミッションビットのサポートが限定的です。Git の `core.fileMode` 設定に依存します。

---

## テンプレート作成の手順

1. **`templates/<name>/base/` にファイル群を配置**
   - プロジェクトに展開したいファイル・ディレクトリをそのまま置く
   - 空ディレクトリは `.gitkeep` で表現

2. **`placements/<name>.yaml` に配置定義を作成**
   - `base_dir`、`conflict_policy`、`post_actions` を適切に設定

3. **`catalog.yaml` にエントリを追加**
   - `name`, `category`, `template_ref`, `placement_ref` を記述
   - 必要に応じて `requirements` と `options` を定義

4. **（任意）ロケールオーバーレイを追加**
   - `templates/<name>/locale.<lang>/` を作り、言語固有ファイルのみ配置

---

## devctl scaffold の実行フロー

```
devctl scaffold [category] [name]
     │
     ├─ catalog.yaml をダウンロード
     ├─ パターン解決（name/category で検索）
     ├─ 前提条件チェック（requirements）
     ├─ placement.yaml をダウンロード
     ├─ templates/<ref>/base/ をダウンロード
     ├─ locale オーバーレイを適用（該当言語があれば）
     ├─ テンプレート変数を適用（.tmpl ファイル）
     ├─ 実行計画を表示（dry-run + コンフリクト判定 + パーミッション変更予定）
     ├─ ユーザー確認 [y/N]
     ├─ チェックポイント作成（git stash）
     ├─ ファイル配置（conflict_policy に従う）
     ├─ 後処理（gitignore 追記 + ファイルパーミッション適用）
     └─ 完了
```
