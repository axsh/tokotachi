# 012-DistScripts-Part3-PublishDevBootstrap

> **Source Specification**: [009-DistScripts-Implementation.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/009-DistScripts-Implementation.md)

## Goal Description

Part 1-2 を基盤に、`publish` (GitHub Releases 公開)、`dev` (開発環境ラッパー)、`bootstrap-tools` (初期セットアップ) を実装して配布スクリプト群を完成させる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.4: `publish` スクリプト | Proposed Changes > `publish` |
| Req.4: `gh` CLI で GitHub Releases | `publish` > gh release create |
| Req.4: リリースノート読み取り | `publish` > notes 処理 |
| Req.4: Homebrew/Scoop マニフェスト生成 | `publish` > テンプレート処理 |
| Req.4: `gh` 未導入時のエラー | `publish` > 前提条件チェック |
| Req.6: `dev` スクリプト | Proposed Changes > `dev` |
| Req.6: devctl up ラッパー | `dev` > メインロジック |
| Req.6: devctl 未ビルド時の自動インストール | `dev` > 自動 install-tools |
| Req.7: `bootstrap-tools` スクリプト | Proposed Changes > `bootstrap-tools` |
| Req.7: Go 存在確認 | `bootstrap-tools` > 前提条件チェック |
| Req.7: 全ツールビルド+インストール | `bootstrap-tools` > メインフロー |

---

## Proposed Changes

### scripts/dist — publish スクリプト

#### [MODIFY] [publish](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/publish)
*   **Description**: スタブを GitHub Releases 公開ロジックに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 引数チェックと前提条件**:
    ```bash
    if [[ $# -lt 2 ]]; then
      echo "Usage: $0 <tool-id> <version>"
      echo "Example: $0 devctl v1.0.0"
      exit 1
    fi
    TOOL_ID="$1"; VERSION="$2"

    # gh CLI の存在確認
    if ! command -v gh &>/dev/null; then
      fail "GitHub CLI (gh) is not installed."
      echo "  Install: https://cli.github.com/"
      exit 1
    fi

    # gh の認証確認
    if ! gh auth status &>/dev/null; then
      fail "GitHub CLI is not authenticated. Run 'gh auth login' first."
      exit 1
    fi
    ```
*   **Logic — リリース成果物の確認**:
    ```bash
    RELEASE_DIR="${REPO_ROOT}/dist/${TOOL_ID}/${VERSION}"
    if [[ ! -d "$RELEASE_DIR" ]] || [[ -z "$(ls -A "$RELEASE_DIR" 2>/dev/null)" ]]; then
      fail "No release artifacts found in ${RELEASE_DIR}/"
      echo "  Run './scripts/dist/release ${TOOL_ID} ${VERSION}' first."
      exit 1
    fi
    ```
*   **Logic — GitHub Release の作成とアップロード**:
    ```bash
    TAG="${TOOL_ID}-${VERSION}"  # 例: devctl-v1.0.0
    TITLE="${TOOL_ID} ${VERSION}"

    # リリースノートの読み取り
    NOTES_FILE="${REPO_ROOT}/releases/notes/latest.md"
    if [[ ! -f "$NOTES_FILE" ]]; then
      warn "No release notes found. Using auto-generated notes."
      NOTES_FLAG="--generate-notes"
    else
      NOTES_FLAG="--notes-file ${NOTES_FILE}"
    fi

    info "Creating GitHub Release: ${TAG}..."
    gh release create "$TAG" \
      --title "$TITLE" \
      $NOTES_FLAG \
      "${RELEASE_DIR}"/*

    pass "Published ${TOOL_ID} ${VERSION} to GitHub Releases"
    ```
*   **Logic — Homebrew/Scoop マニフェスト情報の表示**:
    ```bash
    echo ""
    info "=== Next Steps ==="
    echo "  Homebrew: Update tools/installers/homebrew/Formula/devctl.rb"
    echo "  Scoop:    Update tools/installers/scoop/devctl.json"
    echo ""
    echo "  SHA256 checksums are in: ${RELEASE_DIR}/checksums.txt"
    ```

---

### scripts/dist — dev スクリプト

#### [MODIFY] [dev](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/dev)
*   **Description**: スタブを devctl up ラッパーに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 引数チェック**:
    ```bash
    if [[ $# -lt 1 ]]; then
      echo "Usage: $0 <feature-name>"
      echo "Example: $0 devctl"
      exit 1
    fi
    FEATURE_NAME="$1"
    shift  # 残りの引数を devctl up に渡す
    ```
*   **Logic — devctl の存在確認と自動インストール**:
    ```bash
    DEVCTL="${REPO_ROOT}/bin/devctl"
    NATIVE_OS="$(detect_os)"
    [[ "$NATIVE_OS" == "windows" ]] && DEVCTL="${DEVCTL}.exe"

    if [[ ! -x "$DEVCTL" ]]; then
      warn "devctl not found in bin/. Installing..."
      "${SCRIPT_DIR}/install-tools" devctl
    fi

    if [[ ! -x "$DEVCTL" ]]; then
      fail "Failed to install devctl"
      exit 1
    fi
    ```
*   **Logic — devctl up 実行**:
    ```bash
    info "Starting development environment for ${FEATURE_NAME}..."
    exec "$DEVCTL" up "$FEATURE_NAME" "$@"
    ```

---

### scripts/dist — bootstrap-tools スクリプト

#### [MODIFY] [bootstrap-tools](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/bootstrap-tools)
*   **Description**: スタブを初期セットアップロジックに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 前提条件チェック**:
    ```bash
    echo ""
    echo "╔══════════════════════════════════════╗"
    echo "║     Developer Tools Bootstrap        ║"
    echo "╚══════════════════════════════════════╝"
    echo ""

    # Go の存在確認
    if ! command -v go &>/dev/null; then
      fail "Go is not installed."
      echo "  Install: https://go.dev/dl/"
      exit 1
    fi
    pass "Go $(go version | awk '{print $3}') found"

    # Python の存在確認 (_lib.sh の YAML パースに必要)
    if ! command -v python &>/dev/null && ! command -v python3 &>/dev/null; then
      fail "Python is not installed (required for YAML parsing)."
      exit 1
    fi
    pass "Python found"
    ```
*   **Logic — 全ツールのビルドとインストール**:
    ```bash
    info "Building all tools..."
    TOOL_IDS=()
    while IFS= read -r id; do
      TOOL_IDS+=("$id")
    done < <(get_all_tool_ids)

    total=${#TOOL_IDS[@]}
    built=0; failed=0

    for tool_id in "${TOOL_IDS[@]}"; do
      info "Building ${tool_id}..."
      if "${SCRIPT_DIR}/build" "$tool_id"; then
        built=$((built + 1))
      else
        failed=$((failed + 1))
      fi
    done

    echo ""
    if [[ $failed -gt 0 ]]; then
      fail "${failed}/${total} tools failed to build."
      exit 1
    fi
    pass "All ${total} tools built successfully."

    info "Installing all tools..."
    "${SCRIPT_DIR}/install-tools" --all

    echo ""
    pass "Bootstrap complete! Tools installed to bin/"
    ls -la "${REPO_ROOT}/bin/"
    ```

---

## Step-by-Step Implementation Guide

### Step 1: `publish` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   引数チェック → gh CLI 存在確認 → リリース成果物確認 → gh release create → 情報表示

### Step 2: `dev` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   引数チェック → devctl 存在確認 (未インストール時は自動ビルド) → `devctl up` 実行

### Step 3: `bootstrap-tools` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   Go/Python 存在確認 → 全ツールビルド → install-tools --all → 結果サマリー

### Step 4: `scripts/dist/README.md` の更新

*   `_lib.sh` (共通ライブラリ) の説明を追加

### Step 5: 検証

*   `./scripts/process/build.sh` で既存テスト通過確認
*   `./scripts/dist/bootstrap-tools` で全フロー実行
*   各スクリプトのエラーハンドリング確認

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **publish — gh CLI 未導入時の検証** (gh がインストールされていない場合):
    ```bash
    ./scripts/dist/publish devctl v0.1.0; echo "exit: $?"
    # 期待: "[FAIL] GitHub CLI (gh) is not installed." + exit: 1
    ```

3.  **dev — 引数なし検証**:
    ```bash
    ./scripts/dist/dev; echo "exit: $?"
    # 期待: Usage表示 + exit: 1
    ```

4.  **bootstrap-tools — フルフロー検証**:
    ```bash
    rm -rf dist/ bin/
    ./scripts/dist/bootstrap-tools
    # 期待: 全ツールビルド + インストール + bin/ にバイナリ
    ./bin/devctl version
    # 期待: バージョン表示
    ```

5.  **エラーハンドリング検証**:
    ```bash
    ./scripts/dist/publish; echo "exit: $?"
    # 期待: exit: 1 + usage表示
    ```

---

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/README.md)
*   **更新内容**: `_lib.sh` (共通ライブラリ) の説明を Scripts テーブルに追加

---

## 継続計画について

本計画は Part 3 / 3 です。これで配布スクリプト群の実装が完了します。

- **Part 1** (`010-DistScripts-Part1-BuildFoundation.md`): `_lib.sh` + `build` (前提)
- **Part 2** (`011-DistScripts-Part2-ReleaseInstall.md`): `release` + `install-tools` (前提)
- **Part 3** (本計画): `publish` + `dev` + `bootstrap-tools`
