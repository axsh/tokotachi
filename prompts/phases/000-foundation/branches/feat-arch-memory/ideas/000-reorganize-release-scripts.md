# リリーススクリプトおよび開発環境構築スクリプトの整理仕様書

## 背景

現在、`scripts/dist` ディレクトリ配下には、CLIツールのビルド、リリース成果物の作成、GitHubへのアップロードなどを行う各種リリーススクリプトが配置されています。しかし、このディレクトリには以下の課題があります。

1. **メインスクリプトとサブスクリプトの混在**
   - ユーザー（開発者やオペレーター）が直接実行するスクリプト（例: `github-upload.sh`）と、それらから内部的に呼び出されるサブスクリプト（例: `build.sh`, `release.sh`, `publish.sh`, `_lib.sh`）が同じディレクトリに並んでいるため、どれを実行すべきか混乱を招きやすい状態です。
2. **ツールのリリースとコンテンツのリリースの分離不足**
   - 現在このディレクトリでリリース対象となっているのは `tt` ツールのみですが、今後は別リポジトリ `axsh/tokotachi-scaffolds` を本リポジトリに合流させる計画があります。
   - 合流後は、ツール自体のリリース（`tt` のパッケージングとGitHubアップロード）に加え、ダウンロードコンテンツのパッケージングとpush（コンテンツのリリース）も独立して発生します。
   - 将来的にツールとコンテンツそれぞれのリリース処理が拡張された際、さらにスクリプトが混在して複雑化する懸念があります。
3. **開発環境用スクリプトの混在**
   - `bootstrap-tools.sh` や `install-tools.sh`、`dev.sh` などのローカル開発環境の構築・実行を支援するスクリプトが `dist` 配下に混在しており、リリースのコンテキストと開発環境準備のコンテキストが整理されていません。

これらを解決するため、ディレクトリ構造を整理し、ツールリリース・コンテンツリリース・開発環境構築の役割を明確に分離した構成に再整理します。

## 要件

1. **実行エントリポイントの明確化**
   - ユーザーが直接実行する「公開スクリプト」のみを上位階層に見えるようにし、内部サブスクリプトは `internal` ディレクトリ配下に隠蔽します。
2. **リリース対象（ツール・コンテンツ）の構造分離**
   - ツールリリース用のスクリプト群と、将来のコンテンツリリース用のスクリプト群を、それぞれ `tool/` および `content/` ディレクトリに分離します。
3. **開発環境構築スクリプトの再配置**
   - 開発者の初期セットアップや開発環境起動のためのスクリプトは、リリース関連の `dist/` ディレクトリから切り離し、`scripts/` 直下の適切なディレクトリ（例: `scripts/dev/` などの開発用コンテキスト）へ移動します。
4. **互換性と既存動作の維持**
   - 整理後も、従来のビルド・パッケージング・リリース（GitHubアップロード）の一連の動作が問題なく機能すること。
   - スクリプト間の依存関係（ライブラリの読み込みパスなど）が正しく解決されること。

## 実装方針

### 新しいディレクトリ構造案

`scripts/` 配下の構造を以下のように再整理します。

```
scripts/
├── dist/                          # リリース（配信・パッケージング）関連
│   ├── README.md                  # 各種リリースの概要説明
│   ├── tool/                      # ツール（tt）リリース用
│   │   ├── release.sh             # [公開] ツールリリースのメインエントリポイント (旧 github-upload.sh)
│   │   └── internal/              # ツールリリース用内部サブスクリプト
│   │       ├── build.sh           # (旧 build.sh)
│   │       ├── package.sh         # (旧 release.sh) ※名称変更して役割を明確化
│   │       ├── publish.sh         # (旧 publish.sh)
│   │       ├── install.ps1        # ttのWindows向けインストーラ
│   │       └── uninstall.ps1      # ttのWindows向けアンインストーラ
│   │
│   ├── content/                   # コンテンツリリース用 (将来用ディレクトリ)
│   │   └── README.md              # コンテンツリリース手順のプレースホルダー
│   │
│   └── shared/                    # ツール・コンテンツ共通
│       └── _lib.sh                # 共通ユーティリティ (旧 _lib.sh)
│
├── dev/                           # 開発者用ユーティリティ・セットアップ
│   ├── bootstrap.sh               # 開発環境の初期セットアップ (旧 scripts/dist/bootstrap-tools.sh)
│   ├── install-tools.sh           # 開発ツールのローカルインストール (旧 scripts/dist/install-tools.sh)
│   └── dev.sh                     # 開発環境の起動 (旧 scripts/dist/dev.sh)
│
├── process/                       # 既存のビルド・検証パイプライン
│   ├── build.sh
│   └── integration_test.sh
│
└── utils/                         # 既存のユーティリティ
    └── show_current_status.sh
```

### 変更点と修正内容

1. **ファイル移動とリネーム**
   - `scripts/dist/github-upload.sh` -> `scripts/dist/tool/release.sh`
   - `scripts/dist/build.sh` -> `scripts/dist/tool/internal/build.sh`
   - `scripts/dist/release.sh` -> `scripts/dist/tool/internal/package.sh` (混同を避けるため `package.sh` にリネーム)
   - `scripts/dist/publish.sh` -> `scripts/dist/tool/internal/publish.sh`
   - `scripts/dist/install.ps1` -> `scripts/dist/tool/internal/install.ps1`
   - `scripts/dist/uninstall.ps1` -> `scripts/dist/tool/internal/uninstall.ps1`
   - `scripts/dist/_lib.sh` -> `scripts/dist/shared/_lib.sh`
   - `scripts/dist/bootstrap-tools.sh` -> `scripts/dev/bootstrap.sh`
   - `scripts/dist/install-tools.sh` -> `scripts/dev/install-tools.sh`
   - `scripts/dist/dev.sh` -> `scripts/dev/dev.sh`
2. **スクリプト内のパスおよびソース参照の修正**
   - 各スクリプトで `source` している `_lib.sh` のパスを `../../shared/_lib.sh` 等に書き換えます。
   - `_lib.sh` 内で定義されている `SCRIPT_DIR` および `REPO_ROOT` の解決ロジックが、サブフォルダ移動後も正しく機能するように調整します。
   - `release.sh` (旧 `github-upload.sh`) が内部で呼び出す `build.sh`, `package.sh` (旧 `release.sh`), `publish.sh` のパスを `internal/` 配下に向くように修正します。
   - 開発用スクリプト（`bootstrap.sh`, `install-tools.sh` など）が呼び出している `build.sh` などのリリース内サブスクリプトへの参照を、移動後の新しいパス（`scripts/dist/tool/internal/build.sh` など）に修正します。

## 検証シナリオ

移動および修正を行った後、以下のシナリオで正しく動作することを確認します。

### シナリオ1: 開発環境セットアップの検証
1. `scripts/dev/bootstrap.sh` を実行する。
2. 内部で `scripts/dev/install-tools.sh` が呼ばれ、さらに `scripts/dist/tool/internal/build.sh` が実行されて `tt` が正しくビルドされることを確認する。
3. ビルドされたバイナリが `bin/` 配下にコピーされ、実行可能になっていることを確認する。
4. `scripts/dev/dev.sh tt` を実行して、開発環境が正常に起動することを確認する。

### シナリオ2: ツールリリースビルドの単体検証
1. `scripts/dist/tool/internal/build.sh tt` を実行し、エラーなく `dist/tt/` 以下にバイナリが生成されることを確認する。
2. `scripts/dist/tool/internal/package.sh tt v9.9.9` を実行し、`dist/tt/v9.9.9/` 以下にアーカイブ（tar.gz/zip）およびチェックサムファイルが生成されることを確認する。

### シナリオ3: ツールリリース一括実行の検証
1. `scripts/dist/tool/release.sh tt` をオプションなしで実行する（GitHub APIの書き込み手前で止まる、あるいは dry-run 処理を確認する）。
   ※GitHubへの実際のパブリッシュは行わないように注意する。

## テスト項目

整理されたスクリプトが正常に動作することを確認するため、以下の自動化・半自動化されたコマンドを実行して検証します。

### ビルド・全体検証

1. **プロジェクト全体のビルド＋単体テストの実行**
   ```bash
   ./scripts/process/build.sh
   ```
   （開発用スクリプトやリリース用スクリプトの修正が、既存のビルドパイプラインに悪影響を与えていないことを確認します）

2. **整理後の開発環境・ビルドスクリプトのテスト実行**
   ```bash
   # 1. 開発用ツールのローカルビルドとインストール
   ./scripts/dev/install-tools.sh tt
   
   # 2. 生成されたバイナリの動作確認
   ./bin/tt --version
   
   # 3. ツールリリース用パッケージ作成処理の単体動作確認 (GitHubアップロード手前まで)
   ./scripts/dist/tool/internal/build.sh tt
   ./scripts/dist/tool/internal/package.sh tt v0.0.1-test
   ```
