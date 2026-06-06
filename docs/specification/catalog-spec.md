# テンプレートカタログ 内部仕様資料

本資料は、プロジェクト初期構造や機能構造を生成するテンプレートカタログ（`catalog` ディレクトリ）の設計、構成、およびビルドと適用プロセスについて記述した内部仕様資料です。

---

## 1. カタログのディレクトリ構造

カタログは、開発者が編集するソースファイル（`originals`）と、クライアントに配信するために自動生成される配布物（`scaffolds`）に明確に分離されています。

```
catalog/
├── originals/            # テンプレートのソースファイル（開発者が編集）
│   └── <org>/
│       └── <scaffold-name>/
│           ├── scaffold.yaml  # テンプレート定義（placement 設定を内包）
│           └── base/          # 展開されるベースファイル群
│           └── locale.<lang>/ # 言語別の差分ファイル（任意）
│
└── scaffolds/            # 配布用ファイル（templatizer により自動生成）
    └── <h[0]>/<h[1]>/<h[2]>/
        └── <h[3]>.yaml   # シャーディングされたメタデータファイル
        └── <name>.zip    # テンプレートファイルを固めたアーカイブ
```

また、リポジトリルートに以下のインデックスおよびメタデータが自動生成されます。

- **`catalog.yaml`**: 全テンプレートのインデックスファイル。カテゴリとテンプレート名から、対応するシャーディング YAML のパスをマッピングします。
- **`meta.yaml`**: カタログ全体のメタデータファイル。更新日時（`updated_at`）やデフォルトテンプレート名（`default_scaffold`）を保持し、クライアント側のキャッシュ検証に利用されます。

---

## 2. テンプレートの定義ファイル (`scaffold.yaml`)

`catalog/originals/<org>/<scaffold-name>/scaffold.yaml` は、テンプレートのメタデータと適用時の配置ルールを定義します。

```yaml
name: "axsh-go-standard"
category: "project"
description: "AXSH Go Standard Project"
depends_on:
  - category: "root"
    name: "default"
original_ref: "catalog/originals/axsh/go-standard-project"
placement:
  base_dir: "."
  conflict_policy: "skip"    # skip | overwrite | append | error
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions:
    file_permissions:
      - pattern: "scripts/**/*.sh"
        executable: true
```

### 配置定義 (`placement`)
- `base_dir`: 展開先のベースディレクトリ。テンプレート変数（例: `features/{{.Name}}`）を使用できます。
- `conflict_policy`: 同名の既存ファイルが存在する場合の処理ポリシー。
  - `skip`: 既存ファイルを維持し、生成をスキップします。
  - `overwrite`: 既存ファイルを上書きします。
  - `append`: 既存ファイルの末尾に追記します。
  - `error`: エラーとして処理を中断します。
- `post_actions`:
  - `gitignore_entries`: `.gitignore` に追記するパターンを指定します。
  - `file_permissions`: パターンに一致するファイルへのパーミッション変更（`chmod`）を指定します（`executable: true` で `0755` 適用）。

---

## 3. ビルドおよびリリースプロセス

カタログテンプレートのビルドとリリースは、`scripts/dist/content/release.sh` によって完全に自動化されています。

```
[ originals ] --(templatizer)--> [ scaffolds / catalog.yaml / meta.yaml ] --(git commit & push)--> [ 配信リポジトリ ]
```

### 実行フロー
1. **検証の実行**: `scripts/process/build.sh` を呼び出し、ツールやオリジナルコードのビルドとテストを検証します。
2. **カタログの再生成**: `bin/templatizer` を実行して `catalog/originals/` をスキャンします。
   - `originals` 内のファイル群を ZIP アーカイブとして `catalog/scaffolds/` 配下にパッケージングします。
   - 後述のアルゴリズムに従い、シャーディングされたメタデータ YAML（ZIP の参照パスを内包）を生成します。
   - `catalog.yaml` および `meta.yaml` を再生成します。
3. **リモートへの公開**: 生成されたすべてのカタログファイルをステージングし、`update catalog` メッセージでコミットの上、リモートの作業ブランチにプッシュします。

---

## 4. シャーディングハッシュ算出アルゴリズム

多数のテンプレートが存在する場合でも高速にメタデータへアクセスできるよう、クライアントはテンプレートのカテゴリと名前から、メタデータファイル（シャーディング YAML）の配置パスを直接算出します。

### 算出ステップ
1. **キーの作成**: `key = category + "/" + name`（例: `feature/axsh-go-standard`）
2. **FNV-1a 32-bit ハッシュの計算**:
   - `offset_basis = 2166136261`
   - `prime = 16777619`
   - 各バイトについて `hash = (hash XOR byte) * prime` を計算（32-bit 空間にマスク）。
3. **ハッシュ空間の制限**: `reduced = hash32 % 1679616`（$1679616 = 36^4$）
4. **Base36 エンコード**: 0-9およびa-zを用いた36進数にエンコードし、4文字にゼロパディング（例: `bibl`）。
5. **パスの構築**: `catalog/scaffolds/{encoded[0]}/{encoded[1]}/{encoded[2]}/{encoded[3]}.yaml`（例: `catalog/scaffolds/b/i/b/l.yaml`）

> **ハッシュ衝突時のハンドリング**: 稀に異なるキーが同じハッシュ値（パス）に縮退した場合に備えて、シャーディング YAML の内部は配列構造（`scaffolds: [...]`）になっています。また、ZIP アーカイブ名が衝突した場合は、`templatizer` が自動的に末尾に `-2.zip` などの連番を付与して回避します。

---

## 5. クライアント（tt scaffold）の適用プロセス

`tt scaffold` コマンドが実行された時の解決と適用は、以下の順序で行われます。

```
[引数の解析] ──> [シャーディングパス解決 (方式A/B)] ──> [依存関係の再帰解決] ──> [ZIPのダウンロード・展開] ──> [ロケール適用] ──> [変数置換] ──> [配置・後処理]
```

### 各ステップの詳細

1. **メタデータ取得とキャッシュ**:
   - `meta.yaml` を取得して更新日時を比較します。キャッシュが無効な場合のみ `catalog.yaml` を取得・更新します。
2. **シャーディングパス解決**:
   - **方式A（ダイレクトアクセス）**: アルゴリズムによって算出したシャーディング YAML を GitHub API 経由で直接フェッチします（API 接続を最小化するための推奨方式）。
   - **方式B（インデックスアクセス）**: ハッシュ計算を行えない古いクライアント向けに、`catalog.yaml` インデックスからパスを検索します。
3. **依存関係の解決**:
   - 目的のテンプレートのエントリにある `depends_on` を走査し、依存する親テンプレートを再帰的に解決して、適用順序を決定します（ボトムアップ順）。
4. **ZIP アーカイブの展開とロケール適用**:
   - 指定された ZIP データをダウンロードし、一時フォルダに展開します。
   - クライアントのロケール（環境変数 `LANG` や `--lang` フラグ）を検出し、ZIP 内に `locale.<lang>/` ディレクトリが存在する場合、`base/` ディレクトリのファイルをロケール固有ファイルで上書き（マージ）します。
5. **テンプレート変数の置換**:
   - 定義されたパラメータ（`template_params`）の値を収集（CLI プロンプトまたは `-v` オプションから）します。
   - 拡張子 `.tmpl` を持つファイルを Go テンプレートエンジンで処理し、変数を埋め込んだ後、`.tmpl` 拡張子を除去して配置します。
6. **ファイルの配置と後処理**:
   - 競合解決ポリシーに従い、生成ファイルをターゲットディレクトリ（`base_dir`）に展開します。
   - パーミッション設定や `.gitignore` への追記（`post_actions`）を適用します。
7. **ロールバックの仕組み**:
   - ファイルの書き込み前に、作業ディレクトリの未コミットの変更を自動的に `git stash` して退避し、適用したファイル一覧を記録した `tt-scaffold-checkpoint` チェックポイントファイルを作成します。
   - 適用に失敗した場合、またはユーザーが明示的に `tt scaffold --rollback` を実行した場合は、チェックポイントファイルを元に作成されたファイルを削除し、`git stash pop` を行って適用前の状態に安全に復元します。
