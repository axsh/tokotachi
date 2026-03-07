# 008 — DevTools Platform 構成リファクタリング

## 背景 (Background)

仕様書 007 に基づき DevTools Platform のリポジトリ構成を構築したが、以下の改善が必要であることが判明した:

1. **`scripts/tools/` の命名問題**: このディレクトリの内容（`build`, `release`, `publish` 等）は「アプリケーションの配布パイプライン」に関するスクリプトであり、`tools` という名称は汎用的すぎて役割が不明確。`dist/`（成果物出力先）と対になる `scripts/dist/` がより適切。
2. **`compatibility-matrix.yaml` の不要性**: OS × エディタの互換性マトリクスは現段階では使用されておらず、具体的な利用箇所がない。不要なメタデータとして削除する。
3. **リリース手順ドキュメント不足**: `docs/release-process.md` は概要のみで、具体的な手順が不足している。`scripts/dist/README.md` にリリース手順を記載する。

## 要件 (Requirements)

### 必須要件

1. **`scripts/tools/` → `scripts/dist/` リネーム**
   - ディレクトリ名を `scripts/tools/` から `scripts/dist/` に変更する
   - 配下の全6スクリプト (`dev`, `build`, `release`, `publish`, `install-tools`, `bootstrap-tools`) を移動する
   - 各スクリプト内のエコーメッセージの `[scripts/tools/...]` を `[scripts/dist/...]` に更新する
   - 旧 `scripts/tools/` ディレクトリを削除する

2. **`compatibility-matrix.yaml` の削除**
   - `tools/metadata/compatibility-matrix.yaml` を削除する

3. **影響を受けるドキュメントの更新**
   - 以下のファイル内の `scripts/tools` → `scripts/dist` への参照更新:
     - `prompts/phases/000-foundation/ideas/main/007-DevTools-Platform-Structure.md`
     - `prompts/phases/000-foundation/plans/main/008-DevTools-Platform-Structure.md`
     - `docs/release-process.md`
   - 以下のファイル内の `compatibility-matrix.yaml` 関連記述の削除:
     - `prompts/phases/000-foundation/ideas/main/007-DevTools-Platform-Structure.md`
     - `prompts/phases/000-foundation/plans/main/008-DevTools-Platform-Structure.md`

4. **`scripts/dist/README.md` の作成**
   - アプリケーションのリリース手順を記載する
   - 内容:
     - 前提条件（Go, GoReleaser 等）
     - ビルド手順 (`scripts/dist/build`)
     - リリース成果物の作成手順 (`scripts/dist/release`)
     - 公開手順 (`scripts/dist/publish`)
     - アーティファクトフロー図
     - 各スクリプトの概要と使い方

### 任意要件

- `docs/release-process.md` の内容を `scripts/dist/README.md` へ統合し、`docs/release-process.md` は削除を検討

## 実現方針 (Implementation Approach)

### 影響範囲一覧

| カテゴリ | 対象ファイル | 変更内容 |
|---------|-------------|---------|
| スクリプト移動 | `scripts/tools/*` (6ファイル) | `scripts/dist/` へ移動＋エコーメッセージ修正 |
| ファイル削除 | `tools/metadata/compatibility-matrix.yaml` | 削除 |
| ドキュメント更新 | `007-DevTools-Platform-Structure.md` | `scripts/tools` → `scripts/dist`, `compatibility-matrix` 記述削除 |
| ドキュメント更新 | `008-DevTools-Platform-Structure.md` | `scripts/tools` → `scripts/dist`, `compatibility-matrix` 記述削除 |
| ドキュメント更新 | `docs/release-process.md` | `scripts/tools` → `scripts/dist` |
| 新規作成 | `scripts/dist/README.md` | リリース手順ドキュメント |

### `scripts/dist/README.md` の構成案

```markdown
# Distribution Scripts

## Overview
CLI tools distribution pipeline scripts.

## Prerequisites
- Go 1.21+
- GoReleaser (optional, for automated releases)

## Scripts
| Script | Description |
|--------|-------------|
| build | Build CLI tools from features |
| release | Create release artifacts |
| publish | Publish to GitHub Releases / Homebrew / Scoop |
| dev | Launch development environments |
| install-tools | Install developer tools locally |
| bootstrap-tools | Initial setup for new developers |

## Release Workflow
1. Build: `./scripts/dist/build <tool-id>`
2. Release: `./scripts/dist/release <tool-id> <version>`
3. Publish: `./scripts/dist/publish <tool-id> <version>`

## Artifact Flow
features/ → tools/manifests/ → build → dist/ → release → packaging/ → publish → Homebrew/Scoop/GitHub
```

## 検証シナリオ (Verification Scenarios)

1. `scripts/dist/` に全6スクリプト + `README.md` が存在すること
2. `scripts/tools/` ディレクトリが存在しないこと
3. `tools/metadata/compatibility-matrix.yaml` が存在しないこと
4. 全スクリプトのエコーメッセージが `[scripts/dist/...]` になっていること
5. 3つのドキュメント内で `scripts/tools` の文字列が残っていないこと
6. 2つのドキュメント内で `compatibility-matrix` の文字列が残っていないこと（ファイルパスとしての参照）
7. `./scripts/process/build.sh` が引き続き成功すること

## テスト項目 (Testing for the Requirements)

| 要件 | 検証方法 |
|---|---|
| ディレクトリ移動 | `[ -d scripts/dist ] && [ ! -d scripts/tools ]` |
| ファイル削除 | `[ ! -f tools/metadata/compatibility-matrix.yaml ]` |
| スクリプト存在 | `ls scripts/dist/` で7ファイル確認 (6スクリプト + README.md) |
| エコーメッセージ修正 | `grep -r "scripts/tools" scripts/dist/` が0件 |
| ドキュメント更新 | `grep -r "scripts/tools" docs/ prompts/phases/000-foundation/` が0件 |
| ビルド成功 | `./scripts/process/build.sh` |
