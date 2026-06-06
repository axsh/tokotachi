---
id: current-overview
kind: memory
title: 現在のプロジェクト概要
status: current
topics:
  - 概要
  - モジュール
  - 構造
triggers:
  - プロジェクトへのオンボーディング
  - 全体的なアーキテクチャの理解
  - パッケージ境界の変更
depends_on: []
evidence:
  - コード
  - ドキュメント
review:
  human_required_for:
    - 重大な構造変更
owners:
  - アーキテクチャ
last_reviewed: 2026-06-06
---

# 現在のプロジェクト概要

このドキュメントでは、現在のプロジェクト構造、モジュールの責務、および依存関係について説明します。

## 技術スタック

- **言語**: Go (v1.24+)
- **アーキテクチャ**: モジュール化 / レイヤードアーキテクチャ (Modular / Layered Architecture)
- **ビルドシステム**: カスタムbashスクリプト (`scripts/process/`)

## リポジトリの構造

```
.
├── features/                   # 機能モジュール (垂直スライス)
│   ├── tt/                     # メインCLIツール機能 (main.go, go.mod等)
│   ├── templatizer/            # テンプレート生成機能
│   ├── release-note/           # リリースノート作成機能
│   └── integration-test/       # 統合テスト実行用の機能
│
├── shared/                     # 共有ライブラリおよびユーティリティ
│   └── libs/                   # 再利用可能なライブラリパッケージ
│
├── pkg/                        # ルートモジュール用の公開/再利用可能パッケージ群 (action, scaffold等)
│
├── internal/                   # 外部非公開のユーティリティパッケージ群 (cmdexec, log等)
│
├── tools/                      # ツール用のメタデータおよびインストーラー
│
├── tests/                      # 各機能の統合テストコード
│
├── catalog/                    # スキャフォールドのカタログテンプレート
│
├── scripts/                    # ビルドおよびユーティリティスクリプト
│   ├── process/                # ビルドパイプラインスクリプト (build.sh, integration_test.sh)
│   └── utils/                  # ユーティリティスクリプト (show_current_status.sh等)
│
├── prompts/                    # コーディングエージェント設定 (ソースオブトゥルース)
│   ├── memory/                 # プロジェクトメモリ (本ディレクトリ)
│   ├── manifest/               # 共通IRマニフェスト定義
│   ├── rules/                  # コーディング、テスト、計画のルール
│   └── phases/                 # フェーズごとの開発仕様書
│
├── .agent/                     # Antigravity固有の設定
│   ├── workflows/              # ワークフロー定義
│   └── rules/                  # エージェント指示書
│
├── AGENTS.md                   # ワークスペースレベルのルール (サンドボックス境界)
├── tokotachi.go                # ルートレベルのコアライブラリAPI
├── go.mod                      # ルートモジュールの定義
└── .gitmodules                 # Gitサブモジュールの参照
```

## モジュールの責務

### features/
機能モジュール（垂直スライス）を格納します。
- `features/tt/`: メインCLIアプリケーションモジュール。エントリーポイント (`main.go`) とCLI固有のロジックを含みます。
- `features/templatizer/`, `features/release-note/`, `features/integration-test/`: その他の機能モジュール。

### shared/libs/
機能モジュール間で共有される、再利用可能なライブラリパッケージ。「インターフェースを受け取り、構造体を返す」原則に従います。

### pkg/
ルートモジュール `github.com/axsh/tokotachi` 内の公開パッケージ群。各機能パッケージ (action, detect, scaffold 等) が含まれ、再利用可能です。

### internal/
ルートモジュール内のプライベートパッケージ群 (cmdexec, log 等)。外部モジュールからは直接参照されません。

### tools/
インストーラーやツールのメタデータ。

### tests/
各機能の統合テスト。

### catalog/
スキャフォールド作成時のカタログテンプレート。

### scripts/
開発ワークフローを自動化するビルドおよびユーティリティスクリプト。
- `process/`: パイプラインスクリプト（build.sh、integration_test.sh）
- `utils/`: 補助スクリプト

> [!IMPORTANT]
> `go build`、`go test`、`npm run build` を直接実行することは禁止されています。
> 常に `scripts/process/` にあるスクリプトを使用してください。

### prompts/
コーディングエージェントの設定とプロジェクトドキュメント。
- `memory/`: プロジェクトメモリ（設計ナレッジベース）
- `manifest/`: エージェント用マニフェスト定義
- `rules/`: コーディング規格、テストルール、計画ルール
- `phases/`: フェーズごとの開発仕様書および計画書

### .agent/
Antigravity固有の設定。エージェントが実行するワークフローや指示書が含まれます。

## 依存関係の方向

```
features/tt (および他のfeatures) --> shared/libs, pkg, internal
       |
       v
  prompts/rules (エージェントによって参照される)
  .agent/ (Antigravityによって消費される)
```

依存関係の方向は厳密に内向きです。機能モジュールは共有ライブラリや `pkg`、`internal` に依存できますが、共有ライブラリや `pkg` が機能モジュールに依存することはできません。
