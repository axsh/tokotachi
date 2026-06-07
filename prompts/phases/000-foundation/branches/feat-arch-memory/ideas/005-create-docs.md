# 仕様書：ドキュメントの作成と構造化

## 背景

プロジェクトの開発環境オーケストレーターである tt ツール（旧称 devctl）および、統合されたテンプレートカタログ（catalog）の仕組みについて、ドキュメントが分散していたり、古い記述が残ったままになっていました。
最新の実装に合わせてこれらを整理・統合し、開発者や利用者が容易に参照できるようにするため、トップに docs/ ディレクトリを設け、ユーザマニュアルと内部仕様資料を配置します。

## 要件

1. プロジェクトのルートに docs/ ディレクトリを作成し、その下に docs/manual/ と docs/specification/ の2つのディレクトリを作成します。
2. 以下のドキュメントを作成します。
   - **ユーザマニュアル** (`docs/manual/tt-user-manual.md`): tt ツールの詳細な使用方法。
   - **内部仕様資料** (`docs/specification/catalog-spec.md`): catalog ディレクトリの仕組みと構成。
3. すべての記述は最新の実装（features/tt, pkg/scaffold, features/templatizer）と一致させ、古いリファレンス（devctl 時代のものなど）から現在の仕様へ正しく読み替えて記述します。

## 実現方針

### 1. ディレクトリ構成の構築
プロジェクトルートに以下の階層を作成します。
```
docs/
├── manual/
│   └── tt-user-manual.md      # tt ツールのユーザマニュアル
└── specification/
    └── catalog-spec.md        # テンプレートカタログの内部仕様資料
```

### 2. ユーザマニュアルの執筆内容 (`docs/manual/tt-user-manual.md`)
- **概要**: tt ツール（旧 devctl）が何をするためのツールであるか。
- **基本操作フロー**: 開発環境の起動、編集、終了までの一連のフロー。
- **サブコマンド解説**:
  - `up`: 開発コンテナの起動とワークツリーの自動生成。
  - `down`: コンテナの停止・削除。
  - `open`: エディタの起動（DevContainer へのアタッチ、ローカルワークツリーの起動）。
  - `status`: フィーチャーの状態表示。
  - `shell`: コンテナ内でのシェル起動。
  - `exec`: コンテナ内でのコマンド実行。
  - `pr`: GitHub Pull Request の作成。
  - `close`: 環境のクローズ（down、ワークツリー削除、ブランチ削除）。
  - `list`: フィーチャーのブランチ一覧表示。
  - `scaffold`: テンプレートからのプロジェクト構造の自動生成（パラメータ収集、依存関係解決、競合ポリシー、ロールバックなど）。
- **主要なフラグ**:
  - グローバルフラグ（`--verbose`, `--dry-run`, `--report`）
  - 各サブコマンドの個別フラグ（`--editor`, `--attach`, `--yes`, `--rollback`, `--list` など）
- **環境変数**:
  - `TT_EDITOR` や `TT_CMD_CODE` など、エディタの解決や外部コマンドのオーバーライドに使用する環境変数。

### 3. 内部仕様資料の執筆内容 (`docs/specification/catalog-spec.md`)
- **カタログ構造**:
  - `catalog/originals/`: テンプレートのソースファイル。`scaffold.yaml`（メタデータと配置ルール placement の内包）と `base/` ディレクトリから構成される。
  - `catalog/scaffolds/`: templatizer によって自動生成される ZIP アーカイブとシャーディング YAML（FNV-1a 32-bit ハッシュによる階層化）の配置先。
  - ルートのメタデータとインデックス: `meta.yaml`（更新時間、デフォルトススキャフォールドなど）および `catalog.yaml`（自動生成されたインデックス）。
- **ビルド・リリースプロセス**:
  - `scripts/dist/content/release.sh` が行う処理（build.sh の実行、templatizer によるカタログ再生成、リモートへのコミット＆プッシュ）。
  - `templatizer` ツールが originals から scaffolds を生成するロジック（ZIP の作成、ハッシュ衝突時のリネーム、シャーディングパスの計算）。
- **クライアント（tt scaffold）の解決・適用プロセス**:
  - FNV-1a ハッシュを用いたシャーディングパスの直接算出による高速アクセス（方式A）およびインデックス経由の検索（方式B）。
  - 依存関係の解決、ZIP のダウンロードと展開、ロケールオーバーレイ（locale.<lang>/）の適用。
  - テンプレート変数の置換（.tmpl の処理と拡張子除去）。
  - 配置ルールの適用（conflict_policy や post_actions の実行）。
  - ロールバック（チェックポイント情報の保存と復元）。

## 検証シナリオ

1. 作成したドキュメントファイルが指定のパスに存在すること。
2. 各ドキュメントの内容が、実際の実装（ソースコード、引数定義、出力形式）と矛盾していないこと。
3. `scripts/process/build.sh` を実行し、ドキュメントの追加が既存のビルドやテストに悪影響を与えないこと。

## テスト項目

### ビルド・全体検証

1. ビルド＋単体テストの実行:
   ```bash
   ./scripts/process/build.sh --skip-frontend --skip-etc
   ```

2. ファイル存在確認テスト:
   ```bash
   test -f docs/manual/tt-user-manual.md && test -f docs/specification/catalog-spec.md
   ```
