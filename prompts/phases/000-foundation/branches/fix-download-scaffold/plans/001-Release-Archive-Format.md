# 001-Release-Archive-Format

> **Source Specification**: prompts/phases/000-foundation/branches/fix-download-scaffold/ideas/001-Release-Archive-Format.md

## Goal Description
リリースビルド時（`scripts/dist/tool/internal/package.sh` の実行時）に生成されるアーカイブ（`.tar.gz` と `.zip`）を展開した際、フラットに展開されるのではなく、バージョン番号を含んだフォルダ（例: `tt-v1.0.0/`）が作成されてその中に展開されるようにパッケージング方式を変更します。

## User Review Required
None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| リリースアーカイブを展開した際、ツールIDとバージョン番号を含むディレクトリ（例: `{TOOL_ID}-{VERSION}`、具体的な値として `tt-v1.0.0` など）が作成され、その配下にすべてのファイルが展開されること。 | Proposed Changes > scripts/dist/tool/internal/package.sh |
| 対象アーカイブ形式は、`tar.gz` および `zip`（Windows targets）の両方とする。 | Proposed Changes > scripts/dist/tool/internal/package.sh |
| 既存のビルドプロセスや、生成されるチェックサム生成処理（`checksums.txt`）に悪影響を与えないこと。 | Proposed Changes > scripts/dist/tool/internal/package.sh および Verification Plan |

## Proposed Changes

### Packaging Scripts

#### [MODIFY] [package.sh](file://scripts/dist/tool/internal/package.sh)
*   **Description**: 圧縮前に一時ディレクトリ内に `{TOOL_ID}-{VERSION}` ディレクトリを作成し、ファイルをそこに格納してからディレクトリ自体を圧縮するように変更します。
*   **Technical Design**:
    *   変数 `pkg_dir_name="${TOOL_ID}-${VERSION}"` と `pkg_dir="${tmp_dir}/${pkg_dir_name}"` を追加します。
    *   各ファイルのコピー先を `tmp_dir` から `pkg_dir` に変更します。
    *   zip 圧縮時のコマンドを `(cd "$tmp_dir" && zip -rq "${RELEASE_DIR}/${archive_name}.zip" "${pkg_dir_name}")` に変更します（`-r` オプションを追加し、ディレクトリごと圧縮）。
    *   PowerShell 圧縮時（フォールバック）のパス設定を `win_src="$(cygpath -w "${pkg_dir}")"` に変更し、親ディレクトリから指定して圧縮するようにします。
    *   tar 圧縮時のコマンドを `(cd "$tmp_dir" && tar $tar_opts "${RELEASE_DIR}/${archive_name}.tar.gz" "${pkg_dir_name}")` に変更します。
*   **Logic**:
    *   一時ディレクトリ（`tmp_dir`）配下に中間ディレクトリ `pkg_dir_name`（値は `{TOOL_ID}-{VERSION}`）を作成し、そこに成果物を配置した上で、圧縮対象として `pkg_dir_name` 自体を渡すことで、展開時にそのディレクトリが作成されるようにします。

## Step-by-Step Implementation Guide

1.  **一時フォルダ配下の中間フォルダ生成とコピー先の変更**:
    *   `scripts/dist/tool/internal/package.sh` のバイナリコピー前に中間ディレクトリを定義・作成する処理を追加します。
    *   バイナリおよびWindows用インストール関連ファイルのコピー先を、定義した中間ディレクトリに変更します。
2.  **ZIP/TAR圧縮コマンドの修正**:
    *   Linux/Mac環境での `zip` 圧縮時に `-r` オプションを追加し、中間ディレクトリを指定して圧縮するように変更します。
    *   Windows環境のPowerShellフォールバック処理において、`-Path` 引数に中間ディレクトリの絶対パスを指定して圧縮するように変更します。
    *   Linux/Mac環境での `tar` 圧縮時に中間ディレクトリを指定して圧縮するように変更します。
3.  **ビルドおよび検証の実行**:
    *   後述の Verification Plan に従い検証を行います。

## Verification Plan

### Automated Verification
※本スクリプト（`package.sh`）はリリース時のパッケージングスクリプトであり、製品バイナリの一部ではないため、テストコードはありません。検証はリリース自動化スクリプトの実行とアーカイブ内容の確認により行います。

1.  **Build & Unit Tests**:
    ビルドスクリプトを実行し、ビルドプロセスに問題がないことを確認します。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration & Manual Verification (Script Check)**:
    手動でのリリースクリプトの実行と、生成された成果物の展開テストを行います。
    *   変更が正しく機能しているかを検証するため、一時的にダミーバージョンを使用してパッケージングを行います。
    *   以下のコマンドで、`tt` ツールに対するテストアーカイブを作成します。
        ```bash
        # バイナリがビルドされていることを確認
        ./scripts/dist/tool/internal/build.sh tt
        # パッケージングを実行
        ./scripts/dist/tool/internal/package.sh tt v9.9.9-test
        ```
    *   `dist/tt/v9.9.9-test/` ディレクトリ配下に `tt_*.tar.gz` や `tt_*.zip` が生成されていることを確認します。
    *   生成された `checksums.txt` の内容を確認し、チェックサムが正しく生成されていることを検証します。
    *   生成されたアーカイブを一時ディレクトリに展開し、最上位に `tt-v9.9.9-test/` ディレクトリが存在し、その中に `tt` や `README.md` などのファイルが格納されていることを確認します。
        *   例（WindowsでのZip展開確認）:
            ```bash
            mkdir -p tmp/unzipped
            unzip -q dist/tt/v9.9.9-test/tt_windows_amd64.zip -d tmp/unzipped
            ls -la tmp/unzipped/tt-v9.9.9-test
            ```
        *   例（LinuxでのTar展開確認）:
            ```bash
            mkdir -p tmp/untarred
            tar -xzf dist/tt/v9.9.9-test/tt_linux_amd64.tar.gz -C tmp/untarred
            ls -la tmp/untarred/tt-v9.9.9-test
            ```

### 総合判定プロセス (Post-Test Comprehensive Verdict)
*   すべての検証手順が正常に完了し、展開されたファイルが正しくバージョン付きフォルダに含まれていること、かつ `checksums.txt` が正しく記述されていることを確認した上で、総合判定結果を作成します。

## Documentation
None.
