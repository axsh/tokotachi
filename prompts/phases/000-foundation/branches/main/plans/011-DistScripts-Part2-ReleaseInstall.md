# 011-DistScripts-Part2-ReleaseInstall

> **Source Specification**: [009-DistScripts-Implementation.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/009-DistScripts-Implementation.md)

## Goal Description

Part 1 で構築した `_lib.sh` と `build` を基盤に、`release` (アーカイブ作成 + チェックサム) と `install-tools` (ローカルインストール) を実装する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.3: `release` スクリプト | Proposed Changes > `release` |
| Req.3: tar.gz/zip アーカイブ作成 | `release` > アーカイブループ |
| Req.3: sha256 チェックサム | `release` > チェックサム生成 |
| Req.3: 出力先 dist/<tool-id>/<version>/ | `release` > 出力パス |
| Req.5: `install-tools` スクリプト | Proposed Changes > `install-tools` |
| Req.5: --all で全ツール | `install-tools` > 引数処理 |
| Req.5: ネイティブバイナリのみ | `install-tools` > プラットフォーム検出 |
| Req.5: 未ビルド時の自動ビルド | `install-tools` > 自動 build 呼び出し |

---

## Proposed Changes

### scripts/dist — release スクリプト

#### [MODIFY] [release](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/release)
*   **Description**: スタブをリリース成果物作成ロジックに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 引数チェック**:
    ```bash
    if [[ $# -lt 2 ]]; then
      echo "Usage: $0 <tool-id> <version>"
      echo "Example: $0 devctl v1.0.0"
      exit 1
    fi
    TOOL_ID="$1"; VERSION="$2"
    ```
*   **Logic — ビルド済みバイナリ確認**:
    ```bash
    BUILD_DIR="${REPO_ROOT}/dist/${TOOL_ID}"
    if [[ ! -d "$BUILD_DIR" ]] || [[ -z "$(ls -A "$BUILD_DIR" 2>/dev/null)" ]]; then
      fail "No built binaries found in ${BUILD_DIR}/. Run 'build ${TOOL_ID}' first."
      exit 1
    fi
    ```
*   **Logic — アーカイブ作成ループ**:
    ```bash
    RELEASE_DIR="${REPO_ROOT}/dist/${TOOL_ID}/${VERSION}"
    mkdir -p "$RELEASE_DIR"
    BINARY_NAME="$(get_field "$TOOL_ID" "['binary_name']")"

    while read -r os arch; do
      local ext=""; [[ "$os" == "windows" ]] && ext=".exe"
      local binary="${BUILD_DIR}/${BINARY_NAME}_${os}_${arch}${ext}"
      local archive_name="${BINARY_NAME}_${os}_${arch}"

      if [[ ! -f "$binary" ]]; then
        warn "Binary not found: $binary (skipping)"
        continue
      fi

      info "Creating archive for ${os}/${arch}..."

      # 一時ディレクトリでバイナリ名を binary_name に変更してアーカイブ
      local tmp_dir; tmp_dir="$(mktemp -d)"
      cp "$binary" "${tmp_dir}/${BINARY_NAME}${ext}"

      if [[ "$os" == "windows" ]]; then
        # Windows: zip
        (cd "$tmp_dir" && zip -q "${RELEASE_DIR}/${archive_name}.zip" "${BINARY_NAME}${ext}")
        pass "${archive_name}.zip"
      else
        # Linux/macOS: tar.gz
        (cd "$tmp_dir" && tar czf "${RELEASE_DIR}/${archive_name}.tar.gz" "${BINARY_NAME}${ext}")
        pass "${archive_name}.tar.gz"
      fi

      rm -rf "$tmp_dir"
    done < <(get_platforms "$TOOL_ID")
    ```
*   **Logic — チェックサム生成**:
    ```bash
    info "Generating checksums..."
    (cd "$RELEASE_DIR" && sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt)
    pass "checksums.txt created"

    echo ""
    pass "Release artifacts: ${RELEASE_DIR}/"
    ls -la "$RELEASE_DIR/"
    ```

---

### scripts/dist — install-tools スクリプト

#### [MODIFY] [install-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/install-tools)
*   **Description**: スタブをローカルインストールロジックに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 引数処理**:
    ```bash
    if [[ $# -lt 1 ]]; then
      echo "Usage: $0 [--all | <tool-id>...]"
      echo "Example: $0 devctl"
      echo "         $0 --all"
      exit 1
    fi

    # ツールIDリストを決定
    TOOL_IDS=()
    if [[ "$1" == "--all" ]]; then
      while IFS= read -r id; do
        TOOL_IDS+=("$id")
      done < <(get_all_tool_ids)
    else
      TOOL_IDS=("$@")
    fi
    ```
*   **Logic — インストールループ**:
    ```bash
    NATIVE_OS="$(detect_os)"
    NATIVE_ARCH="$(detect_arch)"
    BIN_DIR="${REPO_ROOT}/bin"
    mkdir -p "$BIN_DIR"

    for tool_id in "${TOOL_IDS[@]}"; do
      BINARY_NAME="$(get_field "$tool_id" "['binary_name']")"
      local ext=""; [[ "$NATIVE_OS" == "windows" ]] && ext=".exe"
      local src="${REPO_ROOT}/dist/${tool_id}/${BINARY_NAME}_${NATIVE_OS}_${NATIVE_ARCH}${ext}"

      # 未ビルドなら自動でビルド
      if [[ ! -f "$src" ]]; then
        warn "${tool_id} not built yet. Building..."
        "${SCRIPT_DIR}/build" "$tool_id"
      fi

      if [[ -f "$src" ]]; then
        cp "$src" "${BIN_DIR}/${BINARY_NAME}${ext}"
        chmod +x "${BIN_DIR}/${BINARY_NAME}${ext}"
        pass "Installed ${BINARY_NAME}${ext} → bin/"
      else
        fail "Failed to install ${tool_id}: binary not found"
      fi
    done
    ```

---

## Step-by-Step Implementation Guide

### Step 1: `release` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   引数チェック → ビルド済みバイナリ確認 → アーカイブ作成ループ → チェックサム生成を実装

### Step 2: `install-tools` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   引数処理 (`--all` / 個別) → インストールループ (未ビルド時自動ビルド含む) を実装

### Step 3: 検証

*   `./scripts/process/build.sh` で既存テスト通過確認
*   `./scripts/dist/build devctl` → `./scripts/dist/release devctl v0.1.0` でアーカイブ + チェックサム確認
*   `./scripts/dist/install-tools devctl` でローカルインストール確認
*   `./bin/devctl version` で動作確認

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **リリース成果物検証**:
    ```bash
    ./scripts/dist/build devctl
    ./scripts/dist/release devctl v0.1.0
    ls -la dist/devctl/v0.1.0/
    # 期待: devctl_linux_amd64.tar.gz, devctl_linux_arm64.tar.gz,
    #       devctl_darwin_amd64.tar.gz, devctl_darwin_arm64.tar.gz,
    #       devctl_windows_amd64.zip, checksums.txt
    ```

3.  **チェックサム検証**:
    ```bash
    cd dist/devctl/v0.1.0 && sha256sum -c checksums.txt
    ```

4.  **ローカルインストール検証**:
    ```bash
    ./scripts/dist/install-tools devctl
    ./bin/devctl version
    # 期待: devctl のバージョンが表示される
    ```

5.  **エラーハンドリング検証**:
    ```bash
    ./scripts/dist/release; echo "exit: $?"
    # 期待: exit: 1 + usage表示

    ./scripts/dist/install-tools; echo "exit: $?"
    # 期待: exit: 1 + usage表示
    ```

---

## 継続計画について

本計画は Part 2 / 3 です。

- **Part 1** (`010-DistScripts-Part1-BuildFoundation.md`): `_lib.sh` + `build` ✅ (前提として完了している必要あり)
- **Part 3** (`012-DistScripts-Part3-PublishDevBootstrap.md`): `publish` + `dev` + `bootstrap-tools`
