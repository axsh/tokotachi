# 仕様書: Tokotachi Scaffoldsのリポジトリ統合

本仕様書は、サブリポジトリである `tokotachi-scaffolds` を本リポジトリに統合し、重複したスクリプトやコードを共有化・整理するための仕様を定義します。

---

## 背景 (Background)

現在、本リポジトリとサブリポジトリである `tokotachi-scaffolds` は別々に管理されており、双方に重複したビルド、テスト、およびリリース用のスクリプトが存在します。
開発効率および保守性を向上させるため、重複するコードを本リポジトリに一元化し、リリースプロセスを本リポジトリの `scripts/dist` 以下に統合します。
また、サブリポジトリ側の不要なスクリプトを移行・排除し、本リポジトリだけでビルド、テスト、およびカタログのパッケージングとデプロイ（リリース）が完結する状態を目指します。

---

## 要件 (Requirements)

1. **ビルドプロセスの統合**:
   - 本リポジトリの `scripts/process/build.sh` で、以下の処理が実行可能であること:
     - `features/templatizer` のビルドとテストを行い、バイナリを `bin/templatizer`（Windows環境では `bin/templatizer.exe`）として出力する。
     - `catalog/originals/` 配下にあるすべてのGoプロジェクト（`go.mod` が存在するディレクトリ）を動的に検出し、それぞれのビルドおよび単体テストを実行する。
     - ビルド成果物は `bin/{プロジェクト名}`（Windows環境では `bin/{プロジェクト名}.exe`）として出力する。

2. **統合テストプロセスの統合**:
   - `scripts/process/integration_test.sh` を本リポジトリにおける唯一の統合テスト実行スクリプトとして利用する。
   - `catalog/originals/` 以下の統合テストについては、個別のユニット/統合ビルドで実行可能な状態であることを保証する。

3. **リリースプロセスの統合と移行**:
   - `scripts/dist/content/` ディレクトリを新規作成し、そこにカタログパッケージング用のリリーススクリプト `release.sh` を配置する。
   - このリリーススクリプトは、以下の手順を自動化する:
     - `scripts/process/build.sh` を実行して、ビルドと単体テストが正常に通過することを確認する。
     - `./bin/templatizer ./catalog/` を実行し、`catalog/originals` から `catalog/scaffolds` への再パッケージング（ZIPアーカイブ作成およびYAMLファイルのシャーディング）を行う。
     - 生成された `catalog/scaffolds/` 以下のファイル、`catalog.yaml`、および `meta.yaml` を Git に追加し、コミットメッセージ "update catalog" で `main` ブランチにコミットおよびプッシュする。
   - スクリプトにはヘルプ表示（`--help`）やエラーハンドリング、カラー出力を実装する。

4. **ドキュメントの更新と整理**:
   - `scripts/dist/README.md` を更新し、`content` ディレクトリおよび新しいカタログリリース手順について記載を追加する。
   - サブリポジトリ内の重要な仕様ドキュメント `001-HowToExtract.md` を本リポジトリの `prompts/phases/000-foundation/refs/tokotachi-scaffolds/` にコピーして保存する。

---

## 実現方針 (Implementation Approach)

### 1. `scripts/process/build.sh` の改修
本リポジトリの `build.sh` に、サブリポジトリの `build.sh` から以下の機能をポーティングしてマージします。
- `build_go` 関数内で `features/templatizer` などの新しい feature も含めて自動ビルドとテストができるように見直すか、もしくは個別の `build_templatizer` 関数を追加します。
- `build_originals` 関数を追加し、`catalog/originals` 以下の `go.mod` を検索して自動でビルドおよびテストを走らせ、`bin/` にバイナリを出力する処理を追加します。

### 2. `scripts/dist/content/release.sh` の新規作成
サブリポジトリの `scripts/process/release.sh` をベースとして、`scripts/dist/content/release.sh` を作成します。
- 実行位置をリポジトリルートに変更します。
- `./bin/templatizer` のパスおよび `./catalog/` のパスが本リポジトリの構成と一致するように調整します。
- バージョン管理が必要な場合は `scripts/dist/tool/release.sh` の設計を参考にしつつ、カタログアセットの更新はGitプッシュで完了する単純な仕組み（元の scaffolds `release.sh` 相当）を踏襲します。

### 3. `scripts/dist/README.md` の改修
- `content/` ディレクトリ配下の構成（`release.sh`）の説明を追加します。
- コマンド例として `./scripts/dist/content/release.sh` を用いたカタログアセットの更新・リリース手順を追記します。

### 4. `001-HowToExtract.md` のコピー
- `prompts/phases/000-foundation/refs/repos/tokotachi-scaffolds/prompts/specifications/001-HowToExtract.md`
- コピー先: `prompts/phases/000-foundation/refs/tokotachi-scaffolds/001-HowToExtract.md`

---

## 検証シナリオ (Verification Scenarios)

以下に、実装完了後の動作を確認するための具体的な手順を示します。

### シナリオ1: フルビルドの実行
1. 本リポジトリのルートで `scripts/process/build.sh` を実行する。
2. 以下の結果を確認する:
   - `features/templatizer` のビルドとテストが通り、`bin/templatizer`（Windowsでは `bin/templatizer.exe`）が生成されていること。
   - `catalog/originals/` 以下のGoプロジェクト（`go-kotoshiro-mcp-feature`、`go-standard-feature` 等）のビルドとテストが通り、`bin/` に対応するバイナリが生成されていること。

### シナリオ2: カタログ更新とリリースプロセスの実行
1. `catalog/originals` 以下のファイルを軽微に変更する（例: コメントの追加など）。
2. `scripts/dist/content/release.sh` を実行する。
3. 以下の結果を確認する:
   - ビルドが正常に走り、続いて `templatizer` が実行され、`catalog/scaffolds` 以下の対応するファイル群が更新されること。
   - Gitに更新ファイルがステージされ、"update catalog" というコミットメッセージでコミットが作成されること。

---

## テスト項目 (Testing for the Requirements)

要件が正しく満たされているかを確認するための自動・手動テストの対応を示します。

### 自動検証

1. **全体ビルドとテストの確認**:
   ```bash
   ./scripts/process/build.sh
   ```
   - templatizer および catalog/originals のテストがすべて通過することを確認する。

2. **統合テストの確認**:
   ```bash
   ./scripts/process/integration_test.sh
   ```
   - 本リポジトリの既存の統合テストがリグレッションなく通過することを確認する。
