# 002-scaffolds-integration

> **Source Specification**: [002-scaffolds-integration.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/branches/feat-arch-memory/ideas/002-scaffolds-integration.md)

## Goal Description

サブリポジトリ `tokotachi-scaffolds` のコード・スクリプトを本リポジトリに統合し、重複したスクリプトやコードを共有化・整理します。
具体的には、以下の作業を実施します：
- `scripts/process/build.sh` に `features/templatizer` のビルド・テストと `catalog/originals/` 以下のGoプロジェクトの動的ビルド・テストを統合。
- 新規ディレクトリ `scripts/dist/content/` を作成し、カタログ再生成と git push を自動化するリリーススクリプト `release.sh` を実装。
- `scripts/dist/README.md` を更新し、新たなカタログリリース手順を記述。
- `001-HowToExtract.md` を `prompts/phases/000-foundation/refs/tokotachi-scaffolds/` 配下にコピー。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| ビルドプロセスの統合 | Proposed Changes > `scripts/process/build.sh` |
| 統合テストプロセスの統合 | Proposed Changes > `scripts/process/integration_test.sh` |
| リリースプロセスの統合と移行 | Proposed Changes > `scripts/dist/content/release.sh` |
| ドキュメントの更新と整理 | Proposed Changes > `scripts/dist/README.md`, `001-HowToExtract.md` コピー |

## Proposed Changes

### Build & Release Infrastructure

#### [MODIFY] [build.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/process/build.sh)
*   **Description**: `features/templatizer` および `catalog/originals` の自動ビルド・テスト機能を追加します。
*   **Technical Design**:
    *   `build_templatizer` 関数および `build_originals` 関数を追加し、`main` 関数から呼び出します。
    *   `build_originals` では、`find` コマンドで `catalog/originals` 配下の `go.mod` を検索し、動的にテストおよびビルドを実行して `bin/` に出力します。Windows環境では出力バイナリに `.exe` を付与します。

#### [NEW] [release.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/content/release.sh)
*   **Description**: カタログ再生成と git push を自動化するコンテンツリリーススクリプトを新規追加します。
*   **Technical Design**:
    *   `scripts/dist/shared/_lib.sh` をインポートして環境定義やヘルパーを利用します。
    *   `scripts/process/build.sh` を実行して検証し、`bin/templatizer`（または `bin/templatizer.exe`）を用いて `./bin/templatizer ./catalog/` を実行します。
    *   変更がある場合、`git add catalog/scaffolds/ catalog.yaml meta.yaml` を実行し、`git commit -m 'update catalog'` でコミットし、`git push origin main` でプッシュします。

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/README.md)
*   **Description**: `content/` ディレクトリとコンテンツリリース手順について説明を追加します。

#### [NEW] [001-HowToExtract.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/prompts/phases/000-foundation/refs/tokotachi-scaffolds/001-HowToExtract.md)
*   **Description**: サブリポジトリからクライアント向けの仕様書ドキュメントを移行します。

---

## Step-by-Step Implementation Guide

1.  **`001-HowToExtract.md` のコピー**:
    *   `prompts/phases/000-foundation/refs/repos/tokotachi-scaffolds/prompts/specifications/001-HowToExtract.md` を `prompts/phases/000-foundation/refs/tokotachi-scaffolds/001-HowToExtract.md` へコピーします。

2.  **`scripts/process/build.sh` の改修**:
    *   `build_templatizer()` 関数を追加し、`features/templatizer` のビルド（`bin/templatizer` への出力）とテストを行います。
    *   `build_originals()` 関数を追加し、`catalog/originals` 配下の `go.mod` を動的に検索してテストおよびビルドを実行し、`bin/` に出力します。
    *   `main()` 関数でこれらを順次呼び出すよう改修します。

3.  **`scripts/dist/content/release.sh` の作成**:
    *   `scripts/dist/content/` フォルダを作成します。
    *   `release.sh` を作成し、ビルド、`templatizer` によるカタログ更新、および Git コミット・プッシュの処理を記述します。
    *   スクリプトに実行権限を付与します。

4.  **`scripts/dist/README.md` の更新**:
    *   `content/` フォルダ配下のリリースフローおよびコマンド例を追記します。

5.  **動作確認と検証の実行**:
    *   検証計画に沿ってテストを実行します。

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    全体ビルドを実行し、追加した `templatizer` および `catalog/originals` のビルド・テストが正常に動作することを確認します。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    既存の統合テスト（`tt`、`release-note`）に影響がないことを確認します。
    ```bash
    ./scripts/process/integration_test.sh
    ```

### Manual Verification
1.  **リリースプロセスのシミュレーション**:
    `catalog/originals/axsh/go-standard-feature/base/README.md` などのドキュメントやコメントに軽微な変更を加え、以下を実行します。
    ```bash
    # リリーススクリプトの実行確認（ドライランや動作確認）
    ./scripts/dist/content/release.sh
    ```
    - カタログの ZIP および YAML インデックスが再生成され、Git にコミットされるところまで動作することを確認します（実際にリモートへのプッシュが成功することを確認）。

---

## テスト設計のセルフレビュー

1.  **ボトムアップ検証順序の遵守**:
    - `build.sh` に追加する `build_templatizer` や `build_originals` は、依存関係のない個別の Go モジュールです。まずこれらの単体テストが個別に動作することを確認した上で、全体ビルドおよびリリーススクリプトの検証を積み上げるように計画しています。
2.  **網羅性の検証**:
    - 要件であるビルドプロセスの統合、リリースプロセスの新規作成、ドキュメントの更新・移行のすべてがステップバイステップガイドおよび検証計画に含まれています。
3.  **総合判定プロセス**:
    - 最後に `./scripts/process/build.sh` と `./scripts/process/integration_test.sh` を通して実行し、プロジェクト全体に影響がないことを確認します。

---

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/README.md)
*   **更新内容**: `content/` ディレクトリとコンテンツリリース手順について説明を追加。
