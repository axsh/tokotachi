# 005-create-docs

> **Source Specification**: [005-create-docs.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/branches/feat-arch-memory/ideas/005-create-docs.md)

## Goal Description
プロジェクトルートに docs/ ディレクトリを作成し、その下に docs/manual/ と docs/specification/ の2つのディレクトリを作成します。それぞれに tt ツール（旧称 devctl）の最新のユーザマニュアルと、テンプレートカタログの内部仕様資料を配置します。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| プロジェクトのルートに docs/ ディレクトリを作成し、その下に docs/manual/ と docs/specification/ の2つのディレクトリを作成する。 | Step-by-Step Implementation Guide > Step 1 |
| docs/manual/tt-user-manual.md に tt ツールの詳細な使用方法を作成する。 | Proposed Changes > docs/manual/tt-user-manual.md |
| docs/specification/catalog-spec.md に catalog ディレクトリの仕組みと構成を作成する。 | Proposed Changes > docs/specification/catalog-spec.md |
| すべての記述は最新の実装と一致させる。 | Proposed Changes |

## Proposed Changes

### docs

#### [NEW] [tt-user-manual.md](file:///docs/manual/tt-user-manual.md)
*   **Description**: tt ツールのユーザマニュアル。旧 devctl ツールから tt ツールへのリネームを反映し、最新のコマンドやフラグ、環境変数について解説します。
*   **Technical Design**:
    以下の構成でマニュアルを執筆します。
    - **概要**: tt ツール（旧 devctl）の目的と役割。
    - **基本操作フロー**: 開発コンテナの起動、編集、終了までの一連のフロー。
    - **サブコマンド詳細仕様**:
      - `up`: 開発用コンテナの起動と worktree 自動生成。
        - フラグ: `--editor cursor|code|ag|claude`、`--ssh`、`--rebuild`、`--no-build`
      - `down`: コンテナの停止・削除。
      - `open`: エディタの起動。
        - フラグ: `--editor`、`--attach`
      - `status`: フィーチャーの状態表示。
      - `shell`: コンテナ内シェル起動。
      - `exec`: コンテナ内でのコマンド実行。
      - `pr`: GitHub Pull Request の作成。
      - `close`: 環境のクローズ（down、worktree削除、ブランチ削除）。
        - フラグ: `--force`
      - `list`: ブランチ一覧表示。
        - フラグ: `--full`、`--env`
      - `scaffold`: テンプレートからのプロジェクト構造自動生成。
        - フラグ: `--yes`、`--rollback`、`--list`、`--repo`、`--branch`、`--lang`、`--root`、`-v`、`--default`、`--skip-deps`、`--force`
    - **グローバルフラグ**: `--verbose`、`--dry-run`、`--report <file>`
    - **環境変数**:
      - `TT_EDITOR`: デフォルトエディタの指定
      - `TT_CMD_CODE` / `TT_CMD_CURSOR` / `TT_CMD_AG` / `TT_CMD_CLAUDE` / `TT_CMD_GIT` / `TT_CMD_GH`: 各外部コマンドパスのオーバーライド

#### [NEW] [catalog-spec.md](file:///docs/specification/catalog-spec.md)
*   **Description**: テンプレートカタログの内部仕様資料。古い scaffolds リポジトリ仕様から現在のモノレポ統合された catalog/ の実態への読み替えを行い、正確な仕組みをドキュメント化します。
*   **Technical Design**:
    以下の構成で仕様資料を執筆します。
    - **カタログディレクトリ構成**:
      - `catalog/originals/`: テンプレートのソースファイル。`scaffold.yaml`（メタデータと配置ルール placement を内包）と `base/` ディレクトリ。
      - `catalog/scaffolds/`: templatizer によって自動生成される ZIP アーカイブとシャーディング YAML（FNV-1a 32-bit ハッシュによる階層化）の配置先。
      - `meta.yaml`: バージョン、デフォルトテンプレート名、更新時間 `updated_at`。
      - `catalog.yaml`: 自動生成されるインデックス。
    - **ビルド・リリースプロセス**:
      - `scripts/dist/content/release.sh` が、`build.sh` での検証後に `bin/templatizer` を実行してカタログを再生成する。
      - `templatizer` ツールが originals から zip を生成し、ハッシュ衝突時にはファイル名に連番を付与する。
    - **シャーディングハッシュ算出アルゴリズム**:
      - FNV-1a 32-bitハッシュから4桁のbase36値を求め、`catalog/scaffolds/{hash[0]}/{hash[1]}/{hash[2]}/{hash[3]}.yaml` を算出するロジック。
      - offset_basis = 2166136261、prime = 16777619、reduced = hash32 % 1679616。
    - **クライアント（tt scaffold）の解決・適用プロセス**:
      - 方式A（ダイレクトアクセス）および方式B（インデックスアクセス）。
      - 依存関係の解決（DependsOn の解決順序）。
      - ZIP 展開とロケールオーバーレイ（locale.<lang>/）の適用。
      - テンプレート変数の置換（.tmpl の処理と拡張子除去）。
      - 配置ルールの適用（conflict_policy、post_actions の実行）。
      - ロールバック（git stash や checkpoint の保存と復元）。

## Step-by-Step Implementation Guide

1.  **Create Directories**:
    *   `docs/manual/` ディレクトリと `docs/specification/` ディレクトリを作成する。
2.  **Write tt User Manual**:
    *   `docs/manual/tt-user-manual.md` を作成し、最新の実装に基づいてユーザマニュアルを日本語で執筆する。
3.  **Write Catalog Specification**:
    *   `docs/specification/catalog-spec.md` を作成し、最新の実装とアルゴリズムに基づいて内部仕様資料を日本語で執筆する。
4.  **Verification**:
    *   検証計画（Verification Plan）に記載の検証コマンドを実行し、ビルドやテストへの影響がないことを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ビルドおよび既存テストに影響がないことを確認するために、ビルドスクリプトを実行します。
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **File Exist Check**:
    作成したドキュメントファイルが指定のパスに存在することを確認します。
    ```bash
    test -f docs/manual/tt-user-manual.md && test -f docs/specification/catalog-spec.md
    ```

### Manual Verification
なし（自動ファイル存在確認とビルド検証のみ）。

## Documentation
（本計画自体が新規ドキュメント作成であるため、既存の prompts/specifications 以下の仕様書の変更はありません。）
