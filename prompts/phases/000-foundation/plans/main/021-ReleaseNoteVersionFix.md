# 021-ReleaseNoteVersionFix

> **Source Specification**: [019-ReleaseNoteVersionFix.md](file://prompts/phases/000-foundation/ideas/main/019-ReleaseNoteVersionFix.md)

## Goal Description
リリースノート生成時に、Gitタグの認識不一致により差分ではなく全ての履歴が対象になってしまう不具合を解消し、さらに `github-upload.sh` におけるバージョンインクリメント処理をSemVer（セマンティックバージョニング）の桁繰り上げルールに準拠させ、末尾が `.0` のインクリメント指定（例: `+v0.1.0`）をエラーとして検出するバリデーションを追加する。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. タグ検出の一致と差分収集 | Proposed Changes > features/release-note/internal/git/history.go |
| 2. バージョンインクリメントにおけるSemVer準拠の繰り上がり | Proposed Changes > scripts/dist/github-upload.sh |
| 3. インクリメント指定フォーマットの制限（エラーチェック） | Proposed Changes > scripts/dist/github-upload.sh |

## Proposed Changes

### release-note (Go)

#### [MODIFY] [history_test.go](file://features/release-note/internal/git/history_test.go)
*   **Description**: `GetLatestReleaseTag` の挙動を検証するためのテストケースが不足しているため、今回のタグパーサー修正およびタグ形式判定の妥当性を確認するための単体テストを補強する。

#### [MODIFY] [history.go](file://features/release-note/internal/git/history.go)
*   **Description**: `GetLatestReleaseTag` において、`publish.sh` が作成するプレフィックスなしのタグ（例： `v0.4.5`）を正しく検出できるように、`jqExpr` でのタグ選択条件を修正する。
*   **Technical Design**:
    ```go
    // GetLatestReleaseTag returns the latest release tag.
    func (c *Collector) GetLatestReleaseTag(toolID string) (string, error)
    ```
*   **Logic**:
    *   `startswith("%s-v")` によるプレフィックス判定から、`test("^v[0-9]+")` による `vN.N.N` フォーマットの判定に変更する。
    *   これにより、`github-upload.sh` の `get_current_version` と整合し、最新タグが正しく `vX.Y.Z` として返されるようになる。

### scripts/dist

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **Description**:
    *   カウントアップ指定引数の厳密なバリデーション（`+v0.1.0` などのエラー制限）。
    *   インクリメント計算時に、上位バージョンが上がったら下位バージョンを `0` にリセットする処理。
*   **Technical Design**:
    *   `VERSION_ARG` 解析ブロックにて、`+` から始まる場合に正規表現 `^\+v([1-9][0-9]*|0\.[1-9][0-9]*|0\.0\.[1-9][0-9]*)$` を用いた制限チェックを追加。
    *   `compute_incremented_version` において、メジャーおよびマイナー増加時にそれ以下の桁をリセットする条件分岐を実装。
*   **Logic**:
    *   バリデーション: `+v1`、`+v0.1`、`+v0.0.1` などの末尾が `.0` ではない形式のみを許可。`+v0.1.0` や `+v1.0` などは `fail "Invalid increment format"` でエラー終了。
    *   計算ロジック:
        *   `inc_major > 0` の場合、`new_major = cur_major + inc_major`, `new_minor=0`, `new_patch=0`
        *   `inc_minor > 0` の場合、`new_minor = cur_minor + inc_minor`, `new_patch=0`
        *   `inc_patch > 0` の場合、`new_patch = cur_patch + inc_patch`

## Integration Tests (tests/)

> [!NOTE]
> 外部コマンドの挙動に依存するため、`release-note` に関しては単体テストおよび手動動作確認で検証します。既存の `tests/release-note/` 以下の統合テストについては、今回の変更による影響がない（デグレードがない）ことを確認するためのビルドテストとして活用します。

## Step-by-Step Implementation Guide

1.  **インクリメント入力フォーマットの制限追加**:
    *   [github-upload.sh](file://scripts/dist/github-upload.sh) に正規表現による厳密な `VERSION_ARG` のバリデーションチェックを追加する。
2.  **インクリメント計算のSemVerリセット処理実装**:
    *   [github-upload.sh](file://scripts/dist/github-upload.sh) の `compute_incremented_version` において、上位桁が上がった場合に下位桁を `0` にリセットする条件ロジックを追加する。
3.  **Goツール（`release-note`）のタグ判定処理修正**:
    *   [history.go](file://features/release-note/internal/git/history.go) の `GetLatestReleaseTag` において、`startswith` によるツール名プレフィックスマッチングから `v` プレフィックスの最新タグをソートして得る方式に修正する。
4.  **ビルド＆単体テストの実行**:
    *   `scripts/process/build.sh` を実行し、既存および新規テストが全て合格することを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "release-note"
    ```

### Manual Verification
1.  **インクリメント値バリデーションの動作確認**:
    *   `./scripts/dist/github-upload.sh tt +v0.1.0` を実行し、バリデーションエラーで即座に終了することを確認する。
    *   `./scripts/dist/github-upload.sh tt +v1.0` を実行し、同様にエラーとなることを確認する。
    *   `./scripts/dist/github-upload.sh tt +v0.1` は正常に実行されることを確認する。
2.  **バージョン計算のSemVer準拠動作確認**:
    （現在の最新バージョンが `v0.4.5` の場合）
    *   `+v0.1` を実行した時、計算結果が `v0.5.0` になることを確認する。
    *   `+v1` を実行した時、計算結果が `v1.0.0` になることを確認する。
    *   `+v0.0.1` を実行した時、計算結果が `v0.4.6` になることを確認する。
3.  **差分収集（マージコミット）の動作確認**:
    *   適当なコミットを行い、`./scripts/dist/github-upload.sh tt +v0.0.1` を実行し、出力されたログで `Latest release tag: v0.4.5`（あるいは最新のタグ名）が検出され、全ブランチではなく差分マージコミットのみが要約対象となっていることを確認する。

## Documentation

既存のドキュメントの更新はありません。
