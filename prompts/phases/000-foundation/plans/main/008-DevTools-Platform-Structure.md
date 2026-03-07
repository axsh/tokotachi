# 008-DevTools-Platform-Structure

> **Source Specification**: [007-DevTools-Platform-Structure.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/007-DevTools-Platform-Structure.md)

## Goal Description

開発支援ツール基盤のリポジトリ構成を構築する。具体的には、CLIツールの配布に必要な**新規ディレクトリ群**（`tools/`, `packaging/`, `releases/`, `scripts/dist/`）と**メタデータファイル群**（YAML定義、インストーラーテンプレート、リリースチャンネル設定等）を作成し、既存リポジトリに `.gitignore` 更新と `feature.yaml` 追加を行う。

## User Review Required

> [!IMPORTANT]
> - 本計画は**ファイル・ディレクトリの作成**が中心であり、Goコードの実装は含まない
> - `scripts/dist/` 配下のスクリプトは**スタブ（ヘッダーのみ）**として作成し、実際のロジック実装は別計画とする
> - インストーラーテンプレート（Homebrew Formula, Scoop manifest, bootstrap）は**テンプレート構造のみ**を定義し、Go template変数のプレースホルダーを使用する
> - 既存の `features/devctl/` に `feature.yaml` を追加する（他の feature には追加しない）

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.1: Feature中心のアーキテクチャ / `feature.yaml` | Proposed Changes > features/devctl > feature.yaml |
| Req.2: 責務の分離 (6レイヤーのディレクトリ分離) | Proposed Changes > 全新規ディレクトリ |
| Req.3: クロスプラットフォーム配布 | Proposed Changes > tools/manifests > devctl.yaml (platforms定義) |
| Req.4: ツールチェーン非依存 (プリコンパイル済みバイナリ) | Proposed Changes > packaging/goreleaser (アーキテクチャ設計のみ、実行ロジックは別計画) |
| Req.5: 配布メタデータ管理 | Proposed Changes > tools/manifests |
| Req.6: インストーラーテンプレート | Proposed Changes > tools/installers |
| Req.7: リリースチャンネル管理 | Proposed Changes > releases/channels |
| 任意: GoReleaserビルド自動化 | Proposed Changes > packaging/goreleaser |
| 任意: チェックサムポリシー | Proposed Changes > packaging/checksums |
| 任意: アーカイブレイアウト規則 | Proposed Changes > packaging/archives |
| 検証1: ディレクトリ階層 | Verification Plan > 構造検証スクリプト |
| 検証2: .gitignore に dist/ | Proposed Changes > .gitignore |
| 検証3: tools.yaml のYAMLパース | Verification Plan > YAML検証 |
| 検証4: feature.yaml のフィールド | Verification Plan > YAML検証 |
| 検証5: devctl.yaml のプラットフォーム網羅 | Verification Plan > YAML検証 |
| 検証6: インストーラーテンプレートの存在 | Verification Plan > 構造検証スクリプト |
| 検証7: スクリプト実行権限 | Verification Plan > 構造検証スクリプト |

---

## Proposed Changes

### .gitignore

#### [MODIFY] [.gitignore](file:///c:/Users/yamya/myprog/tokotachi/.gitignore)
*   **Description**: `dist/` と `build/` をignore対象に追加
*   **Technical Design**:
    *   既存の内容（`bin/`, `tmp/`, `work/*`, `.DS_Store`, `Thumbs.db`）は保持
    *   `# Distribution artifacts` セクションとして `dist/` を追加
    *   `# Build artifacts` セクションとして `build/` を追加

---

### tools/manifests — ツール登録メタデータ

#### [NEW] [tools.yaml](file:///c:/Users/yamya/myprog/tokotachi/tools/manifests/tools.yaml)
*   **Description**: グローバルツールレジストリ。配布対象となるCLIツールを一覧で定義する
*   **Technical Design**:
    ```yaml
    tools:
      - id: devctl
        feature_path: features/devctl
        type: go-cli
        binary_name: devctl
        release: true
    ```
    *   将来 `featurectl` 等が追加される際は、この一覧にエントリを追加する

#### [NEW] [devctl.yaml](file:///c:/Users/yamya/myprog/tokotachi/tools/manifests/devctl.yaml)
*   **Description**: devctl の個別配布マニフェスト。プラットフォーム、リリース形式、インストーラー対応を定義する
*   **Technical Design**:
    ```yaml
    id: devctl
    feature_path: features/devctl
    type: go-cli
    binary_name: devctl
    main_package: ./cmd/devctl

    platforms:
      - os: linux
        arch: amd64
      - os: linux
        arch: arm64
      - os: darwin
        arch: amd64
      - os: darwin
        arch: arm64
      - os: windows
        arch: amd64

    release:
      archive: zip
      checksum: true

    install:
      homebrew: true
      scoop: true
    ```

---

### tools/installers — インストーラーテンプレート

#### [NEW] [devctl.rb.tmpl](file:///c:/Users/yamya/myprog/tokotachi/tools/installers/homebrew/Formula/devctl.rb.tmpl)
*   **Description**: Homebrew Formula テンプレート。リリース時に GoReleaser もしくは publish スクリプトがこのテンプレートから実際の `.rb` ファイルを生成する
*   **Technical Design**:
    ```ruby
    class Devctl < Formula
      desc "{{ .Description }}"
      homepage "{{ .Homepage }}"
      version "{{ .Version }}"

      on_macos do
        if Hardware::CPU.arm?
          url "{{ .BaseURL }}/devctl_darwin_arm64.tar.gz"
          sha256 "{{ .SHA256.DarwinArm64 }}"
        else
          url "{{ .BaseURL }}/devctl_darwin_amd64.tar.gz"
          sha256 "{{ .SHA256.DarwinAmd64 }}"
        end
      end

      on_linux do
        if Hardware::CPU.arm?
          url "{{ .BaseURL }}/devctl_linux_arm64.tar.gz"
          sha256 "{{ .SHA256.LinuxArm64 }}"
        else
          url "{{ .BaseURL }}/devctl_linux_amd64.tar.gz"
          sha256 "{{ .SHA256.LinuxAmd64 }}"
        end
      end

      def install
        bin.install "devctl"
      end

      test do
        system "#{bin}/devctl", "version"
      end
    end
    ```

#### [NEW] [devctl.json.tmpl](file:///c:/Users/yamya/myprog/tokotachi/tools/installers/scoop/devctl.json.tmpl)
*   **Description**: Scoop bucket マニフェストテンプレート
*   **Technical Design**:
    ```json
    {
      "version": "{{ .Version }}",
      "description": "{{ .Description }}",
      "homepage": "{{ .Homepage }}",
      "license": "{{ .License }}",
      "architecture": {
        "64bit": {
          "url": "{{ .BaseURL }}/devctl_windows_amd64.zip",
          "hash": "{{ .SHA256.WindowsAmd64 }}"
        }
      },
      "bin": "devctl.exe",
      "checkver": {
        "github": "{{ .GitHubRepo }}"
      },
      "autoupdate": {
        "architecture": {
          "64bit": {
            "url": "{{ .BaseURL }}/devctl_windows_amd64.zip"
          }
        }
      }
    }
    ```

#### [NEW] [install.sh.tmpl](file:///c:/Users/yamya/myprog/tokotachi/tools/installers/bootstrap/install.sh.tmpl)
*   **Description**: ブートストラップインストーラー (Linux/macOS用)。パッケージマネージャが使えない環境向け
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # Bootstrap installer for {{ .ToolName }}
    # Usage: curl -fsSL <url>/install.sh | bash

    set -euo pipefail

    TOOL_NAME="{{ .ToolName }}"
    VERSION="{{ .Version }}"
    BASE_URL="{{ .BaseURL }}"

    detect_platform() {
      local os arch
      os="$(uname -s | tr '[:upper:]' '[:lower:]')"
      arch="$(uname -m)"
      case "$arch" in
        x86_64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
      esac
      echo "${os}_${arch}"
    }

    main() {
      local platform archive_url install_dir
      platform="$(detect_platform)"
      archive_url="${BASE_URL}/${TOOL_NAME}_${platform}.tar.gz"
      install_dir="${HOME}/.local/bin"

      echo "Installing ${TOOL_NAME} ${VERSION} for ${platform}..."
      mkdir -p "$install_dir"
      curl -fsSL "$archive_url" | tar xz -C "$install_dir" "$TOOL_NAME"
      echo "Installed to ${install_dir}/${TOOL_NAME}"
    }

    main "$@"
    ```

#### [NEW] [install.ps1.tmpl](file:///c:/Users/yamya/myprog/tokotachi/tools/installers/bootstrap/install.ps1.tmpl)
*   **Description**: ブートストラップインストーラー (Windows用)
*   **Technical Design**:
    ```powershell
    # Bootstrap installer for {{ .ToolName }}
    # Usage: iwr -useb <url>/install.ps1 | iex

    $ToolName = "{{ .ToolName }}"
    $Version = "{{ .Version }}"
    $BaseURL = "{{ .BaseURL }}"

    $ArchiveURL = "${BaseURL}/${ToolName}_windows_amd64.zip"
    $InstallDir = "$env:LOCALAPPDATA\${ToolName}\bin"

    Write-Host "Installing ${ToolName} ${Version} for windows_amd64..."
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $TempFile = [System.IO.Path]::GetTempFileName() + ".zip"
    Invoke-WebRequest -Uri $ArchiveURL -OutFile $TempFile
    Expand-Archive -Path $TempFile -DestinationPath $InstallDir -Force
    Remove-Item $TempFile
    Write-Host "Installed to ${InstallDir}\${ToolName}.exe"
    ```

---

### tools/metadata — グローバル配布設定

#### [NEW] [release-policy.yaml](file:///c:/Users/yamya/myprog/tokotachi/tools/metadata/release-policy.yaml)
*   **Description**: リリースポリシー定義
*   **Technical Design**:
    ```yaml
    versioning: semver
    default_channel: stable
    pre_release_channels:
      - experimental
    release_approval: manual
    ```

#### [NEW] [naming.yaml](file:///c:/Users/yamya/myprog/tokotachi/tools/metadata/naming.yaml)
*   **Description**: アーティファクト命名規則
*   **Technical Design**:
    ```yaml
    binary: "{{ .ToolName }}"
    archive: "{{ .ToolName }}_{{ .OS }}_{{ .Arch }}"
    archive_extension:
      default: tar.gz
      windows: zip
    checksum_file: checksums.txt
    ```

---

### packaging — ビルドパッケージング定義

#### [NEW] [base.yaml](file:///c:/Users/yamya/myprog/tokotachi/packaging/goreleaser/base.yaml)
*   **Description**: GoReleaser共通設定
*   **Technical Design**:
    ```yaml
    # GoReleaser base configuration
    # This file defines shared settings for all tool builds.
    version: 2

    env:
      - CGO_ENABLED=0

    builds: []

    archives:
      - format: tar.gz
        format_overrides:
          - goos: windows
            format: zip
        name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
        files:
          - LICENSE
          - README.md

    checksum:
      name_template: checksums.txt
      algorithm: sha256
    ```

#### [NEW] [devctl.yaml](file:///c:/Users/yamya/myprog/tokotachi/packaging/goreleaser/devctl.yaml)
*   **Description**: devctl用GoReleaser設定
*   **Technical Design**:
    ```yaml
    # GoReleaser configuration for devctl
    # Extends base.yaml settings.
    project_name: devctl
    version: 2

    builds:
      - id: devctl
        dir: features/devctl
        main: ./cmd/devctl
        binary: devctl
        env:
          - CGO_ENABLED=0
        goos:
          - linux
          - darwin
          - windows
        goarch:
          - amd64
          - arm64
        ignore:
          - goos: windows
            goarch: arm64

    dist: dist/devctl
    ```

#### [NEW] [layout.yaml](file:///c:/Users/yamya/myprog/tokotachi/packaging/archives/layout.yaml)
*   **Description**: アーカイブレイアウト規則
*   **Technical Design**:
    ```yaml
    include:
      - LICENSE
      - README.md
    structure:
      root: "{{ .ToolName }}-{{ .Version }}"
    ```

#### [NEW] [policy.yaml](file:///c:/Users/yamya/myprog/tokotachi/packaging/checksums/policy.yaml)
*   **Description**: チェックサムポリシー
*   **Technical Design**:
    ```yaml
    algorithm: sha256
    filename: checksums.txt
    ```

---

### releases — リリースメタデータ

#### [NEW] [devctl.md](file:///c:/Users/yamya/myprog/tokotachi/releases/changelogs/devctl.md)
*   **Description**: devctl変更履歴。初期エントリのみ作成
*   **Technical Design**:
    ```markdown
    # devctl Changelog

    ## Unreleased

    - Initial release preparation
    ```

#### [NEW] [latest.md](file:///c:/Users/yamya/myprog/tokotachi/releases/notes/latest.md)
*   **Description**: 最新リリースノート（テンプレート）
*   **Technical Design**:
    ```markdown
    # Release Notes

    ## What's New

    _No releases yet._
    ```

#### [NEW] [release-note.md.tmpl](file:///c:/Users/yamya/myprog/tokotachi/releases/notes/templates/release-note.md.tmpl)
*   **Description**: リリースノートのテンプレートファイル
*   **Technical Design**:
    ```markdown
    # {{ .ToolName }} {{ .Version }}

    ## What's New

    {{ .Changelog }}

    ## Downloads

    | Platform | Architecture | Download |
    |----------|-------------|----------|
    {{ range .Assets -}}
    | {{ .OS }} | {{ .Arch }} | [{{ .Filename }}]({{ .URL }}) |
    {{ end }}

    ## Checksums

    SHA256 checksums are available in `checksums.txt`.
    ```

#### [NEW] [stable.yaml](file:///c:/Users/yamya/myprog/tokotachi/releases/channels/stable.yaml)
*   **Description**: stable チャンネル。リリース済みツールバージョンを管理
*   **Technical Design**:
    ```yaml
    # Stable release channel
    # Lists the latest stable versions of each tool.
    stable: {}
    ```

#### [NEW] [experimental.yaml](file:///c:/Users/yamya/myprog/tokotachi/releases/channels/experimental.yaml)
*   **Description**: experimental チャンネル
*   **Technical Design**:
    ```yaml
    # Experimental release channel
    # Lists the latest experimental/pre-release versions of each tool.
    experimental: {}
    ```

---

### features/devctl — feature.yaml 追加

#### [NEW] [feature.yaml](file:///c:/Users/yamya/myprog/tokotachi/features/devctl/feature.yaml)
*   **Description**: devctl の Feature メタデータ定義
*   **Technical Design**:
    ```yaml
    name: devctl
    kind: cli
    language: go
    entrypoint: ./cmd/devctl
    release:
      enabled: true
    install:
      expose_as: devctl
    ```
    *   フィールド定義:

    | フィールド | 値 | 説明 |
    |---|---|---|
    | `name` | `devctl` | 論理的なFeature名 |
    | `kind` | `cli` | CLIツール |
    | `language` | `go` | Go言語 |
    | `entrypoint` | `./cmd/devctl` | メインパッケージパス |
    | `release.enabled` | `true` | 配布対象 |
    | `install.expose_as` | `devctl` | インストール時バイナリ名 |

---

### docs — ドキュメント構造

#### [NEW] [docs/architecture/](file:///c:/Users/yamya/myprog/tokotachi/docs/architecture/)
*   **Description**: アーキテクチャドキュメント用ディレクトリ。`.gitkeep` を配置

#### [NEW] [docs/tooling/](file:///c:/Users/yamya/myprog/tokotachi/docs/tooling/)
*   **Description**: ツーリングドキュメント用ディレクトリ。`.gitkeep` を配置

#### [NEW] [docs/release-process.md](file:///c:/Users/yamya/myprog/tokotachi/docs/release-process.md)
*   **Description**: リリースプロセスの概要ドキュメント
*   **Technical Design**:
    ```markdown
    # Release Process

    ## Overview

    This document describes the release process for CLI tools.

    ## Steps

    1. Build: `scripts/dist/build`
    2. Release: `scripts/dist/release`
    3. Publish: `scripts/dist/publish`

    See each script for detailed usage.
    ```

---

### scripts/dist — スクリプトスタブ

各スクリプトは **bash スタブ**（ヘルプ表示のみ）として作成する。実際のロジック実装は別計画とする。

#### [NEW] [dev](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/dev)
*   **Description**: 開発環境起動スクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/dev] Development environment launcher (not yet implemented)"
    echo "Usage: $0 <feature-name>"
    exit 1
    ```

#### [NEW] [build](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/build)
*   **Description**: CLIツールビルドスクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/build] CLI tool builder (not yet implemented)"
    echo "Usage: $0 <tool-id>"
    exit 1
    ```

#### [NEW] [release](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/release)
*   **Description**: リリース成果物作成スクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/release] Release artifact creator (not yet implemented)"
    echo "Usage: $0 <tool-id> <version>"
    exit 1
    ```

#### [NEW] [publish](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/publish)
*   **Description**: リリース公開スクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/publish] Release publisher (not yet implemented)"
    echo "Usage: $0 <tool-id> <version>"
    exit 1
    ```

#### [NEW] [install-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/install-tools)
*   **Description**: 開発者ツールインストールスクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/install-tools] Developer tool installer (not yet implemented)"
    echo "Usage: $0 [--all | <tool-id>...]"
    exit 1
    ```

#### [NEW] [bootstrap-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/bootstrap-tools)
*   **Description**: 初期セットアップスクリプト（スタブ）
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    echo "[scripts/dist/bootstrap-tools] Initial setup script (not yet implemented)"
    echo "Usage: $0"
    exit 1
    ```

---

## Step-by-Step Implementation Guide

### [x] Step 1: `.gitignore` の更新

*   `.gitignore` に `dist/` と `build/` エントリを追加する

### [x] Step 2: `tools/` ディレクトリ一式の作成

1. `tools/manifests/tools.yaml` を作成（グローバルレジストリ）
2. `tools/manifests/devctl.yaml` を作成（devctl個別マニフェスト）
3. `tools/metadata/release-policy.yaml` を作成
4. `tools/metadata/naming.yaml` を作成

### [x] Step 3: `tools/installers/` の作成

1. `tools/installers/homebrew/Formula/devctl.rb.tmpl` を作成
2. `tools/installers/scoop/devctl.json.tmpl` を作成
3. `tools/installers/bootstrap/install.sh.tmpl` を作成
4. `tools/installers/bootstrap/install.ps1.tmpl` を作成

### [x] Step 4: `packaging/` ディレクトリの作成

1. `packaging/goreleaser/base.yaml` を作成
2. `packaging/goreleaser/devctl.yaml` を作成
3. `packaging/archives/layout.yaml` を作成
4. `packaging/checksums/policy.yaml` を作成

### [x] Step 5: `releases/` ディレクトリの作成

1. `releases/changelogs/devctl.md` を作成
2. `releases/notes/latest.md` を作成
3. `releases/notes/templates/release-note.md.tmpl` を作成
4. `releases/channels/stable.yaml` を作成
5. `releases/channels/experimental.yaml` を作成

### [x] Step 6: `features/devctl/feature.yaml` の追加

*   `features/devctl/feature.yaml` を作成

### [x] Step 7: `docs/` ディレクトリの整備

1. `docs/architecture/.gitkeep` を作成
2. `docs/tooling/.gitkeep` を作成
3. `docs/release-process.md` を作成

### [x] Step 8: `scripts/dist/` スクリプトスタブの作成

1. 6つのスクリプトスタブ (`dev`, `build`, `release`, `publish`, `install-tools`, `bootstrap-tools`) を作成
2. 各スクリプトに実行権限を付与: `chmod +x scripts/dist/*`

### [x] Step 9: 検証

1. 構造検証スクリプトを実行してディレクトリ・ファイルの存在を確認
2. YAML構文の検証
3. `.gitignore` の確認

---

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    既存のビルドとテストが影響を受けていないことを確認する。
    ```bash
    ./scripts/process/build.sh
    ```

2. **構造検証**:
    仕様で定義されたすべてのファイルとディレクトリが存在することを確認する。
    ```bash
    # ディレクトリ存在確認
    for dir in \
      tools/manifests tools/installers/homebrew/Formula tools/installers/scoop \
      tools/installers/bootstrap tools/metadata \
      packaging/goreleaser packaging/archives packaging/checksums \
      releases/changelogs releases/notes/templates releases/channels \
      scripts/dist docs/architecture docs/tooling; do
      [ -d "$dir" ] && echo "OK: $dir" || echo "FAIL: $dir"
    done

    # 必須ファイル存在確認
    for file in \
      tools/manifests/tools.yaml tools/manifests/devctl.yaml \
      tools/metadata/release-policy.yaml \
      tools/metadata/naming.yaml \
      tools/installers/homebrew/Formula/devctl.rb.tmpl \
      tools/installers/scoop/devctl.json.tmpl \
      tools/installers/bootstrap/install.sh.tmpl \
      tools/installers/bootstrap/install.ps1.tmpl \
      packaging/goreleaser/base.yaml packaging/goreleaser/devctl.yaml \
      packaging/archives/layout.yaml packaging/checksums/policy.yaml \
      releases/changelogs/devctl.md releases/notes/latest.md \
      releases/notes/templates/release-note.md.tmpl \
      releases/channels/stable.yaml releases/channels/experimental.yaml \
      features/devctl/feature.yaml \
      docs/release-process.md \
      scripts/dist/dev scripts/dist/build scripts/dist/release \
      scripts/dist/publish scripts/dist/install-tools scripts/dist/bootstrap-tools; do
      [ -f "$file" ] && echo "OK: $file" || echo "FAIL: $file"
    done

    # .gitignore確認
    grep -q "dist/" .gitignore && echo "OK: dist/ in .gitignore" || echo "FAIL: dist/ not in .gitignore"
    grep -q "build/" .gitignore && echo "OK: build/ in .gitignore" || echo "FAIL: build/ not in .gitignore"

    # スクリプト実行権限確認 (Windowsでは不完全な場合あり)
    for script in scripts/dist/*; do
      [ -x "$script" ] && echo "OK: $script is executable" || echo "WARN: $script may not be executable"
    done
    ```

3. **YAML構文検証**:
    すべてのYAMLファイルがパース可能であることを確認する。Python の PyYAML を使用する。
    ```bash
    python3 -c "
    import yaml, sys, glob
    files = glob.glob('tools/**/*.yaml', recursive=True) + \
            glob.glob('packaging/**/*.yaml', recursive=True) + \
            glob.glob('releases/**/*.yaml', recursive=True) + \
            glob.glob('features/devctl/feature.yaml')
    ok = True
    for f in files:
        try:
            with open(f) as fh:
                yaml.safe_load(fh)
            print(f'OK: {f}')
        except Exception as e:
            print(f'FAIL: {f}: {e}')
            ok = False
    sys.exit(0 if ok else 1)
    "
    ```

---

## Documentation

#### [NEW] [docs/release-process.md](file:///c:/Users/yamya/myprog/tokotachi/docs/release-process.md)
*   **更新内容**: リリースプロセスの概要を新規作成（Step 7で作成済み）
