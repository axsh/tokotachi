# 010-DistScripts-Part1-BuildFoundation

> **Source Specification**: [009-DistScripts-Implementation.md](file:///c:/Users/yamya/myprog/tokotachi/prompts/phases/000-foundation/ideas/main/009-DistScripts-Implementation.md)

## Goal Description

配布スクリプトの基盤となる共通ライブラリ `_lib.sh` と、コアスクリプト `build` を実装する。これにより `./scripts/dist/build devctl` でクロスプラットフォームビルドが可能になる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Req.1: `_lib.sh` 共通ライブラリ | Proposed Changes > `_lib.sh` |
| Req.1: YAML読み取り関数 | `_lib.sh` > `yaml_get()` |
| Req.1: 色付き出力関数 | `_lib.sh` > `info()`, `pass()`, `fail()`, `warn()` |
| Req.1: プラットフォーム検出関数 | `_lib.sh` > `detect_os()`, `detect_arch()` |
| Req.1: マニフェスト読み取り関数 | `_lib.sh` > `manifest_path()`, `get_field()`, `get_platforms()` |
| Req.1: ツール一覧取得関数 | `_lib.sh` > `get_all_tool_ids()` |
| Req.2: `build` スクリプト | Proposed Changes > `build` |
| Req.2: マニフェスト読み取り + クロスコンパイル | `build` > main ロジック |
| Req.2: 出力先 dist/<tool-id>/ | `build` > 出力パス |
| Req.2: GOOS/GOARCH/CGO_ENABLED=0 設定 | `build` > ビルドループ |
| Req.2: 成功/失敗の色付き表示 | `build` > 結果サマリー |

---

## Proposed Changes

### scripts/dist — 共通ライブラリ

#### [NEW] [_lib.sh](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/_lib.sh)
*   **Description**: 全配布スクリプトが source する共通ライブラリ
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # Common library for distribution scripts
    # Usage: source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 色付き出力関数**:
    ```bash
    info()  { echo -e "\033[0;34m[INFO]\033[0m $*"; }
    pass()  { echo -e "\033[0;32m[PASS]\033[0m $*"; }
    fail()  { echo -e "\033[1;31m[FAIL]\033[0m $*"; }
    warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }
    ```
*   **Logic — YAML値取得**:
    ```bash
    # yaml_get <file> <python-expression>
    # 例: yaml_get "tools/manifests/devctl.yaml" "['binary_name']"
    # → "devctl"
    yaml_get() {
      local file="$1" expr="$2"
      python -c "import yaml,sys; d=yaml.safe_load(open('${file}')); print(d${expr})" 2>/dev/null
    }
    ```
*   **Logic — プラットフォーム検出**:
    ```bash
    detect_os() {
      case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unknown" ;;
      esac
    }
    detect_arch() {
      case "$(uname -m)" in
        x86_64)        echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
      esac
    }
    ```
*   **Logic — マニフェスト操作**:
    ```bash
    manifest_path() { echo "${REPO_ROOT}/tools/manifests/${1}.yaml"; }

    # get_field <tool-id> <python-expression>
    get_field() { yaml_get "$(manifest_path "$1")" "$2"; }

    # get_platforms <tool-id>
    # 出力: "linux amd64\nlinux arm64\ndarwin amd64\n..."
    get_platforms() {
      local file; file="$(manifest_path "$1")"
      python -c "
    import yaml
    d = yaml.safe_load(open('${file}'))
    for p in d.get('platforms', []):
        print(p['os'], p['arch'])
    "
    }

    # get_all_tool_ids
    # tools/manifests/tools.yaml から全ツールIDを取得
    get_all_tool_ids() {
      yaml_get "${REPO_ROOT}/tools/manifests/tools.yaml" "" | \
      python -c "
    import yaml,sys
    d = yaml.safe_load(open('${REPO_ROOT}/tools/manifests/tools.yaml'))
    for t in d.get('tools', []):
        print(t['id'])
    "
    }
    ```
*   **Logic — 共通初期化**:
    ```bash
    # SCRIPT_DIR, REPO_ROOT を自動設定
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")" && pwd)"
    REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
    ```

---

### scripts/dist — build スクリプト

#### [MODIFY] [build](file:///c:/Users/yamya/myprog/tokotachi/scripts/dist/build)
*   **Description**: スタブを実際のクロスコンパイルロジックに置き換える
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    set -euo pipefail
    source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"
    ```
*   **Logic — 引数チェック**:
    ```bash
    if [[ $# -lt 1 ]]; then
      echo "Usage: $0 <tool-id>"
      echo "Example: $0 devctl"
      exit 1
    fi
    TOOL_ID="$1"
    ```
*   **Logic — マニフェスト読み取り**:
    ```bash
    MANIFEST="$(manifest_path "$TOOL_ID")"
    if [[ ! -f "$MANIFEST" ]]; then
      fail "Manifest not found: $MANIFEST"
      exit 1
    fi
    FEATURE_PATH="$(get_field "$TOOL_ID" "['feature_path']")"
    BINARY_NAME="$(get_field "$TOOL_ID" "['binary_name']")"
    MAIN_PACKAGE="$(get_field "$TOOL_ID" "['main_package']")"
    ```
*   **Logic — クロスコンパイルループ**:
    ```bash
    DIST_DIR="${REPO_ROOT}/dist/${TOOL_ID}"
    mkdir -p "$DIST_DIR"

    total=0; passed=0; failed=0

    while read -r os arch; do
      total=$((total + 1))
      # Windows はバイナリに .exe を付与
      local ext=""; [[ "$os" == "windows" ]] && ext=".exe"
      local output="${DIST_DIR}/${BINARY_NAME}_${os}_${arch}${ext}"

      info "Building ${TOOL_ID} for ${os}/${arch}..."
      if CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
         go build -o "$output" "./${FEATURE_PATH}/${MAIN_PACKAGE}"; then
        pass "${os}/${arch} → $(basename "$output")"
        passed=$((passed + 1))
      else
        fail "${os}/${arch} build failed"
        failed=$((failed + 1))
      fi
    done < <(get_platforms "$TOOL_ID")

    # 結果サマリー
    echo ""
    if [[ $failed -eq 0 ]]; then
      pass "All ${total} builds succeeded. Output: ${DIST_DIR}/"
    else
      fail "${failed}/${total} builds failed."
      exit 1
    fi
    ```

---

## Step-by-Step Implementation Guide

### Step 1: `_lib.sh` の作成

*   `scripts/dist/_lib.sh` を新規作成
*   色付き出力関数 (`info`, `pass`, `fail`, `warn`) を実装
*   YAML読み取り関数 (`yaml_get`) を実装
*   プラットフォーム検出関数 (`detect_os`, `detect_arch`) を実装
*   マニフェスト操作関数 (`manifest_path`, `get_field`, `get_platforms`, `get_all_tool_ids`) を実装
*   REPO_ROOT 自動設定ロジックを実装

### Step 2: `build` スクリプトの実装

*   既存スタブを置き換え
*   `_lib.sh` を source
*   引数チェック → マニフェスト読み取り → クロスコンパイルループ → 結果サマリーを実装

### Step 3: 検証

*   `./scripts/process/build.sh` で既存テストが通ることを確認
*   `./scripts/dist/build devctl` でクロスビルド実行
*   `dist/devctl/` に5バイナリが生成されることを確認
*   エラーケース (`./scripts/dist/build nonexistent`) の確認

---

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **クロスビルド検証**:
    ```bash
    ./scripts/dist/build devctl
    ls -la dist/devctl/
    # 期待: devctl_linux_amd64, devctl_linux_arm64, devctl_darwin_amd64, devctl_darwin_arm64, devctl_windows_amd64.exe
    ```

3.  **エラーハンドリング検証**:
    ```bash
    ./scripts/dist/build nonexistent; echo "exit: $?"
    # 期待: exit: 1
    ./scripts/dist/build; echo "exit: $?"
    # 期待: exit: 1 + usage表示
    ```

---

## 継続計画について

本計画は Part 1 / 3 です。後続の計画:

- **Part 2** (`011-DistScripts-Part2-ReleaseInstall.md`): `release` + `install-tools` スクリプトの実装
- **Part 3** (`012-DistScripts-Part3-PublishDevBootstrap.md`): `publish` + `dev` + `bootstrap-tools` スクリプトの実装
