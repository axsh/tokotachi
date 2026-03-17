# 000-GoModuleVersioning

> **Source Specification**: [000-GoModuleVersioning.md](file://prompts/phases/000-foundation/ideas/fix-module-versioning/000-GoModuleVersioning.md)

## Goal Description

GitHub Release のタグ形式を `tt-vX.Y.Z` から `vX.Y.Z` に変更し、Go モジュールシステムが認識できるようにする。これにより外部プロジェクトが `go get github.com/axsh/tokotachi@v0.4.3` のようにバージョンを指定してインポートできるようになる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. タグ形式を `vX.Y.Z` に統一 | Proposed Changes > `publish.sh` |
| 2. `get_current_version()` のタグ検索ロジック修正 | Proposed Changes > `github-upload.sh` |
| 3. `go get` による取得の実現 | 要件 1, 2 の結果として自動的に達成 |
| 4. `go.mod` のモジュールパスは変更しない | 変更対象外（変更しないことで対応） |
| 5. Go モジュールプロキシへの反映確認（任意） | 手動検証で対応 |

## Proposed Changes

### リリーススクリプト (scripts/dist)

#### [MODIFY] [publish.sh](file://scripts/dist/publish.sh)

*   **Description**: タグ生成ロジックからツールIDプレフィックスを削除し、`vX.Y.Z` 形式に統一する
*   **Technical Design**:
    *   L41: `TAG` 変数の生成式を変更
    *   `TITLE` は引き続き `${TOOL_ID} ${VERSION}` を維持（GitHub Release の表示名であり、Go モジュールに影響しない）
*   **Logic**:
    *   **変更前**: `TAG="${TOOL_ID}-${VERSION}"` → `tt-v0.4.3`
    *   **変更後**: `TAG="${VERSION}"` → `v0.4.3`

---

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)

*   **Description**: `get_current_version()` 関数のタグ検索ロジックを新しいタグ形式に対応させる
*   **Technical Design**:
    *   `get_current_version()` 関数内の jq クエリと結果処理を変更
*   **Logic**:
    *   **タグ検索クエリの変更**:
        *   変更前: `startswith("${tool_id}-v")` で `tt-v` プレフィックスのタグを検索
        *   変更後: `test("^v[0-9]")` で `v` で始まるセマンティックバージョンタグを検索
    *   **バージョン抽出の変更**:
        *   変更前: `echo "${tag#${tool_id}-}"` でプレフィックスを除去して `v0.4.3` を取得
        *   変更後: `echo "$tag"` でタグをそのまま返す（プレフィックス除去が不要）

## Step-by-Step Implementation Guide

1.  [x] **`publish.sh` のタグ生成を修正**:
    *   `scripts/dist/publish.sh` の L41 を編集
    *   `TAG="${TOOL_ID}-${VERSION}"` → `TAG="${VERSION}"`
2.  [x] **`github-upload.sh` の `get_current_version()` を修正**:
    *   `scripts/dist/github-upload.sh` の L45-53 を編集
    *   jq クエリ: `startswith("${tool_id}-v")` → `test("^v[0-9]")`
    *   バージョン抽出: プレフィックス除去コード（L51-52）を削除し、`echo "$tag"` に変更
3.  [x] **ビルド検証**:
    *   `./scripts/process/build.sh` でプロジェクト全体のビルドと単体テストが通ることを確認
4.  [x] **コードレビュー検証**:
    *   `grep` で変更箇所を確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    プロジェクト全体のビルドと単体テストを実行し、既存機能に回帰がないことを確認する。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **変更箇所の正確性チェック**:
    変更後のファイルの該当箇所を `grep` で確認する。
    ```bash
    grep -n 'TAG=' scripts/dist/publish.sh
    grep -n 'test(' scripts/dist/github-upload.sh
    grep -n 'tool_id' scripts/dist/github-upload.sh
    ```
    *   **期待結果**:
        *   `publish.sh`: `TAG="${VERSION}"` が存在し、`TOOL_ID` を含むタグ生成がないこと
        *   `github-upload.sh`: `startswith` が存在せず、`test("^v[0-9]")` が存在すること
        *   `github-upload.sh`: `get_current_version()` 関数内に `tool_id` プレフィックス除去コードがないこと

### Manual Verification

本変更はシェルスクリプトのみの修正であり、実際の GitHub Release 作成を伴うため、完全な E2E 検証はリリース時に行う。

*   **手順**: 次回 `./scripts/dist/github-upload.sh tt vX.Y.Z` を実行した際に、GitHub Release のタグが `vX.Y.Z` 形式であることを確認する

## Documentation

変更対象となる仕様書やドキュメントはありません。
