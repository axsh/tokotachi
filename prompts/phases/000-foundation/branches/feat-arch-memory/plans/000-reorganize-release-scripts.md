# 000-reorganize-release-scripts

> **Source Specification**: [000-reorganize-release-scripts.md](../../ideas/000-reorganize-release-scripts.md)

## Goal Description

`scripts/dist` 配下のリリース関連スクリプトと開発環境用スクリプトのディレクトリ構造を整理し、ツールリリース・コンテンツリリース・開発環境構築の責務を分離します。
また、Windows向けパッケージ作成時に、Windows用インストーラ（`install.ps1`, `uninstall.ps1`）を自動的に同梱するようにパッケージ作成処理を修正します。

## User Review Required

*   **開発用スクリプトの移動**: 開発環境構築用のスクリプトが `scripts/dist/` 配下から `scripts/dev/` 配下に移動します。本番リリース担当以外の一般の開発者も影響を受ける変更であるため、ドキュメント等の参照先が変更される点にご留意ください。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 実行エントリポイントの明確化 | Proposed Changes > `scripts/dist/tool/release.sh` |
| リリース対象（ツール・コンテンツ）の構造分離 | Proposed Changes > `scripts/dist/` ディレクトリ設計 |
| 開発環境構築スクリプトの再配置 | Proposed Changes > `scripts/dev/` |
| Windows向け配布パッケージへのインストーラ同梱 | Proposed Changes > `scripts/dist/tool/internal/package.sh` |
| 互換性と既存動作の維持 | Proposed Changes > スクリプト内のパス参照および共通ライブラリの修正 |

## Proposed Changes

### scripts/dist

#### [NEW] [release.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/release.sh)
*   **Description**: ツールリリースのメインエントリポイント（旧 `github-upload.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../../shared/_lib.sh` に修正。
    *   内部で呼び出すサブスクリプトのパスを以下に修正：
        *   `build.sh` -> `internal/build.sh`
        *   `release.sh` -> `internal/package.sh`
        *   `publish.sh` -> `internal/publish.sh`
*   **Logic**:
    ```bash
    # (変更例)
    source "$(dirname "${BASH_SOURCE[0]}")/../shared/_lib.sh"
    
    # 呼び出し部分のパス修正
    "${SCRIPT_DIR}/internal/build.sh" "$TOOL_ID"
    "${SCRIPT_DIR}/internal/package.sh" "$TOOL_ID" "$NEW_VERSION"
    "${SCRIPT_DIR}/internal/publish.sh" "$TOOL_ID" "$NEW_VERSION"
    ```

#### [NEW] [build.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/internal/build.sh)
*   **Description**: ツールのビルド処理（旧 `build.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../../../shared/_lib.sh` に修正。

#### [NEW] [package.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/internal/package.sh)
*   **Description**: リリース成果物（アーカイブ）の作成（旧 `release.sh` からリネーム）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../../../shared/_lib.sh` に修正。
    *   Windows向けの Zip パッケージ作成時に、同ディレクトリ内の `install.ps1` と `uninstall.ps1` を一時ディレクトリにコピーした上で圧縮対象に同梱するロジックを追加。
*   **Logic**:
    ```bash
    if [[ "$os" == "windows" ]]; then
      # 同ディレクトリにあるインストーラを一時フォルダにコピー
      cp "$(dirname "${BASH_SOURCE[0]}")/install.ps1" "${tmp_dir}/"
      cp "$(dirname "${BASH_SOURCE[0]}")/uninstall.ps1" "${tmp_dir}/"
      
      # Windows: zip archive (with fallback to PowerShell)
      if command -v zip &>/dev/null; then
        (cd "$tmp_dir" && zip -q "${RELEASE_DIR}/${archive_name}.zip" "${BINARY_NAME}${ext}" "install.ps1" "uninstall.ps1")
      else
        # Fallback: use PowerShell Compress-Archive for all files in tmp_dir
        win_src_pattern="$(cygpath -w "${tmp_dir}")\\*"
        win_dst="$(cygpath -w "${RELEASE_DIR}/${archive_name}.zip")"
        powershell -NoProfile -Command "Compress-Archive -Path '${win_src_pattern}' -DestinationPath '${win_dst}' -Force"
      fi
      pass "${archive_name}.zip"
    ```

#### [NEW] [publish.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/internal/publish.sh)
*   **Description**: リリース成果物のパブリッシュ処理（旧 `publish.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../../../shared/_lib.sh` に修正。

#### [NEW] [install.ps1](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/internal/install.ps1)
*   **Description**: Windows向けインストーラ（移動のみ）。

#### [NEW] [uninstall.ps1](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/tool/internal/uninstall.ps1)
*   **Description**: Windows向けアンインストーラ（移動のみ）。

#### [NEW] [_lib.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/shared/_lib.sh)
*   **Description**: 共通ライブラリ。
*   **Technical Design**:
    *   `_lib.sh` がサブフォルダに移動したため、呼び出し元のスクリプトのディレクトリに依存せず `REPO_ROOT` が正しく解決されるように、`git rev-parse --show-toplevel` を使用するようにロジックを修正。
*   **Logic**:
    ```bash
    # REPO_ROOT を git から堅牢に解決
    REPO_ROOT="$(git rev-parse --show-toplevel)"
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")" && pwd)"
    ```

#### [NEW] [README.md](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dist/README.md) （上書き）
*   **Description**: `scripts/dist` 配下の新しい構造、役割、使用方法について更新。

#### [DELETE] 旧 `scripts/dist/` 直下ファイル群
*   **Description**: 以下の古いファイルを削除します。
    *   `scripts/dist/github-upload.sh`
    *   `scripts/dist/build.sh`
    *   `scripts/dist/release.sh`
    *   `scripts/dist/publish.sh`
    *   `scripts/dist/install.ps1`
    *   `scripts/dist/uninstall.ps1`
    *   `scripts/dist/_lib.sh`
    *   `scripts/dist/bootstrap-tools.sh`
    *   `scripts/dist/install-tools.sh`
    *   `scripts/dist/dev.sh`

---

### scripts/dev

#### [NEW] [bootstrap.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dev/bootstrap.sh)
*   **Description**: 開発環境の初期セットアップ（旧 `bootstrap-tools.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../dist/shared/_lib.sh` に修正。
    *   内部で呼び出す `build.sh` および `install-tools.sh` のパスを新しい場所に修正。
*   **Logic**:
    ```bash
    # パス修正例
    "${REPO_ROOT}/scripts/dist/tool/internal/build.sh" "$tool_id"
    "${REPO_ROOT}/scripts/dev/install-tools.sh" --all
    ```

#### [NEW] [install-tools.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dev/install-tools.sh)
*   **Description**: 開発ツールのローカルインストール（旧 `install-tools.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../dist/shared/_lib.sh` に修正.
    *   内部で呼び出す `build.sh` のパスを `../dist/tool/internal/build.sh` に修正。

#### [NEW] [dev.sh](file:///c:/Users/yamya/myprog/tokotachi/work/feat-arch-memory/scripts/dev/dev.sh)
*   **Description**: 開発環境の起動（旧 `dev.sh`）。
*   **Technical Design**:
    *   `_lib.sh` のインポート先を `../dist/shared/_lib.sh` に修正。
    *   内部で呼び出す `install-tools.sh` のパスを `../dev/install-tools.sh` に修正。

---

## Step-by-Step Implementation Guide

1.  **共通ライブラリ `_lib.sh` の配置変更と修正**:
    *   `scripts/dist/_lib.sh` を `scripts/dist/shared/_lib.sh` に移動します。
    *   移動した `_lib.sh` 内の `REPO_ROOT` 解決ロジックを `git rev-parse --show-toplevel` を使ったロジックに変更します。
2.  **ツールリリース用スクリプトの移動と修正**:
    *   `scripts/dist/github-upload.sh` を `scripts/dist/tool/release.sh` に移動します。
    *   `scripts/dist/build.sh` を `scripts/dist/tool/internal/build.sh` に移動します。
    *   `scripts/dist/release.sh` を `scripts/dist/tool/internal/package.sh` に移動・リネームします。
    *   `scripts/dist/publish.sh` を `scripts/dist/tool/internal/publish.sh` に移動します。
    *   `scripts/dist/install.ps1` を `scripts/dist/tool/internal/install.ps1` に移動します。
    *   `scripts/dist/uninstall.ps1` を `scripts/dist/tool/internal/uninstall.ps1` に移動します。
    *   移動した各スクリプトで `source` している `_lib.sh` のパスを修正します。
    *   `release.sh` (エントリポイント) が内部で呼び出す `build.sh`, `package.sh`, `publish.sh` のパスを修正します。
    *   `package.sh` で Windows パッケージ作成時に `install.ps1` / `uninstall.ps1` を同梱するロジック（Proposed Changes に記載のコード）を追加します。
3.  **開発環境構築スクリプトの移動と修正**:
    *   `scripts/dist/bootstrap-tools.sh` を `scripts/dev/bootstrap.sh` に移動します。
    *   `scripts/dist/install-tools.sh` を `scripts/dev/install-tools.sh` に移動します。
    *   `scripts/dist/dev.sh` を `scripts/dev/dev.sh` に移動します。
    *   移動した各スクリプトで `source` している `_lib.sh` のパスを修正します。
    *   内部で呼び出している他スクリプトのパス（`scripts/dist/tool/internal/build.sh` など）を修正します。
4.  **README.mdの更新**:
    *   `scripts/dist/README.md` を新しい構造に合わせて上書き更新します。
5.  **不要になった旧スクリプトの削除**:
    *   移動前の元の場所に残っている古いスクリプトファイルを削除します。

---

## Verification Plan

### Automated Verification

本変更はシェルスクリプトの配置整理およびパッケージングの同梱修正であり、GoコードのロジックやWebviewのUIには直接影響しません。しかし、既存のビルドパイプラインにエラーが発生していないことを自動テストで担保します。

1.  **Build & Unit Tests**:
    ビルドスクリプトを実行し、プロジェクト全体が正常にビルド・テストできることを確認します。
    ```bash
    ./scripts/process/build.sh
    ```

### Manual Verification

整理したスクリプトの動作検証は手動（スクリプト実行）により行います。

1.  **開発環境セットアップの確認**:
    移動後の `bootstrap.sh` を実行してツールが正しくビルド・インストールされることを確認します。
    ```bash
    # 開発環境セットアップの実行
    ./scripts/dev/bootstrap.sh
    
    # ローカルにインストールされたツールの実行確認
    ./bin/tt --version
    ```

2.  **パッケージング処理の検証**:
    個別にビルド・パッケージ作成スクリプトを実行し、Windowsパッケージを作成して同梱物を検証します。
    ```bash
    # 1. ttツールのビルド
    ./scripts/dist/tool/internal/build.sh tt
    
    # 2. パッケージング処理の実行
    ./scripts/dist/tool/internal/package.sh tt v9.9.9
    ```
    *   `dist/tt/v9.9.9/` 配下に `tt_windows_amd64.zip` が生成されていることを確認します。
    *   生成された `tt_windows_amd64.zip` を解凍（または `unzip -l` 等で一覧確認）し、直下に以下の3つのファイルが同梱されていることを確認します。
        *   `tt.exe`
        *   `install.ps1`
        *   `uninstall.ps1`

3.  **エントリポイント（一括リリース）スクリプトの動作確認 (Dry-Run)**:
    エントリポイント `release.sh` が正しく内部スクリプトを呼び出せているか確認します。
    ```bash
    # バージョン取得やビルドが走ることを確認 (GitHub Releasesへのpublish手前のghコマンド等は未認証や対象外で止まる、あるいはエラーになる想定)
    ./scripts/dist/tool/release.sh tt v9.9.9
    ```

### GUI E2E Tests (非適用)
*   **理由**: 今回の変更はすべてバックエンド/開発用のシェルスクリプトであり、Webview等のUI変更は一切含まれていないため、GUI E2Eテストは適用外とします。

---

## Documentation

以下のドキュメントを本計画に合わせて更新します。

#### [MODIFY] [README.md](file:///c:/Users/yamya\myprog\tokotachi\work\feat-arch-memory\scripts\dist\README.md)
*   **更新内容**: `scripts/dist` 配下の新しいディレクトリ構造と、ユーザー（オペレータ）が使うスクリプト、および開発用スクリプトの移動先についてのドキュメントを更新。
