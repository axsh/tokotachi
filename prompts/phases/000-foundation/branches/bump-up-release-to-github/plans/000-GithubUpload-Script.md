# 000-GithubUpload-Script

> **Source Specification**: [000-GithubUpload-Script.md](file://prompts/phases/000-foundation/ideas/bump-up-release-to-github/000-GithubUpload-Script.md)

## Goal Description

`scripts/dist/` 配下に `github-upload.sh` スクリプトを新規作成し、build → release → publish を一括実行可能にする。加えて、既存スクリプトのファイル名に `.sh` 拡張子を付与し、README.md を更新する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `github-upload.sh` スクリプトの作成 | Proposed Changes > github-upload.sh |
| R2: バージョン指定オプション（絶対/増分/省略） | Proposed Changes > github-upload.sh > `parse_version_arg`, `compute_new_version` |
| R3: バージョン形式バリデーション | Proposed Changes > github-upload.sh > `validate_version_format` |
| R4: バージョンダウングレード防止 | Proposed Changes > github-upload.sh > `compare_versions` |
| R5: 既存スクリプトのリネーム | Proposed Changes > Rename Section |
| R6: README.md の更新 | Proposed Changes > README Section |

## Proposed Changes

### scripts/dist — スクリプトリネーム

#### [MODIFY] [build → build.sh](file://scripts/dist/build)
*   **Description**: ファイル名を `build` → `build.sh` にリネーム（`git mv`使用）
*   **内容変更**: なし（ファイル名のみ変更）

#### [MODIFY] [release → release.sh](file://scripts/dist/release)
*   **Description**: ファイル名を `release` → `release.sh` にリネーム
*   **内容変更**: なし

#### [MODIFY] [publish → publish.sh](file://scripts/dist/publish)
*   **Description**: ファイル名を `publish` → `publish.sh` にリネーム
*   **内容変更**: なし

#### [MODIFY] [dev → dev.sh](file://scripts/dist/dev)
*   **Description**: ファイル名を `dev` → `dev.sh` にリネーム
*   **内容変更**: `install-tools` への参照を `install-tools.sh` に更新

現在のコード (L27):
```bash
  "${SCRIPT_DIR}/install-tools" devctl
```
変更後:
```bash
  "${SCRIPT_DIR}/install-tools.sh" devctl
```

#### [MODIFY] [install-tools → install-tools.sh](file://scripts/dist/install-tools)
*   **Description**: ファイル名を `install-tools` → `install-tools.sh` にリネーム
*   **内容変更**: `build` への参照を `build.sh` に更新

現在のコード (L52):
```bash
    "${SCRIPT_DIR}/build" "$tool_id"
```
変更後:
```bash
    "${SCRIPT_DIR}/build.sh" "$tool_id"
```

#### [MODIFY] [bootstrap-tools → bootstrap-tools.sh](file://scripts/dist/bootstrap-tools)
*   **Description**: ファイル名を `bootstrap-tools` → `bootstrap-tools.sh` にリネーム
*   **内容変更**: `build` と `install-tools` への参照を更新

現在のコード (L47):
```bash
  if "${SCRIPT_DIR}/build" "$tool_id"; then
```
変更後:
```bash
  if "${SCRIPT_DIR}/build.sh" "$tool_id"; then
```

現在のコード (L63):
```bash
"${SCRIPT_DIR}/install-tools" --all
```
変更後:
```bash
"${SCRIPT_DIR}/install-tools.sh" --all
```

---

### scripts/dist — 新規スクリプト

#### [NEW] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **Description**: build → release → publish を一括実行するオーケストレーションスクリプト

*   **Technical Design**:

    ```bash
    #!/usr/bin/env bash
    # All-in-one: build → release → publish to GitHub
    # Usage: ./scripts/dist/github-upload.sh <tool-id> [version|+increment]
    # Examples:
    #   ./scripts/dist/github-upload.sh devctl v1.2.0
    #   ./scripts/dist/github-upload.sh devctl +v0.1.0
    #   ./scripts/dist/github-upload.sh devctl            # defaults to +v0.0.1
    ```

*   **Logic — 引数パース (`parse_args`)**:
    1. 第1引数 `TOOL_ID` を必須で受け取る。無い場合は usage を表示して exit 1
    2. 第2引数 `VERSION_ARG` はオプション:
        - 省略時: `VERSION_ARG="+v0.0.1"` をデフォルトセット
        - `+` で始まる場合: 増分モード (`MODE="increment"`)
        - それ以外: 絶対指定モード (`MODE="absolute"`)

*   **Logic — バージョン形式バリデーション (`validate_version_format`)**:
    ```bash
    # 正規表現: ^v[0-9]+\.[0-9]+\.[0-9]+$
    # 増分の場合は先頭の "+" を除去してからバリデーション
    # マッチしない場合はエラー:
    #   fail "Invalid version format: '${raw}'. Expected: v{N}.{N}.{N}"
    #   exit 1
    ```

*   **Logic — 現在バージョン取得 (`get_current_version`)**:
    ```bash
    # gh CLI で現在の最新リリースタグを取得
    # タグ形式: "{tool-id}-v{N}.{N}.{N}" (例: "devctl-v1.0.0")
    local tag
    tag=$(gh release list --limit 100 --json tagName --jq \
      "[.[] | select(.tagName | startswith(\"${TOOL_ID}-v\"))] | sort_by(.tagName) | last | .tagName // empty")

    if [[ -z "$tag" ]]; then
      echo "v0.0.0"  # 初回リリース
    else
      echo "${tag#${TOOL_ID}-}"  # プレフィックス除去 → "v1.0.0"
    fi
    ```

*   **Logic — バージョン算出 (`compute_new_version`)**:
    ```bash
    # バージョン文字列 "vX.Y.Z" をパース
    parse_semver() {
      local ver="${1#v}"
      IFS='.' read -r major minor patch <<< "$ver"
    }

    # 増分モードの場合:
    #   current の各コンポーネント + increment の各コンポーネント
    #   例: v1.2.3 + v0.1.0 → v1.3.3
    # 絶対モードの場合:
    #   VERSION_ARG をそのまま使用
    ```

*   **Logic — バージョン比較 (`compare_versions`)**:
    ```bash
    # version_le_or_eq: 新バージョン <= 現在バージョン ならエラー
    # major を比較 → minor を比較 → patch を比較
    #
    # エラーメッセージ:
    #   同一の場合: "Version ${new} is the same as current version ${current}."
    #   低い場合:   "Version ${new} is lower than current version ${current}."
    ```

*   **Logic — メイン実行フロー**:
    ```bash
    # 1. 引数パース
    # 2. バージョン形式バリデーション
    # 3. 現在バージョン取得
    # 4. 新バージョン算出
    # 5. バージョン比較（ダウングレードチェック）
    # 6. 確認表示
    info "Tool:    ${TOOL_ID}"
    info "Current: ${CURRENT_VERSION}"
    info "New:     ${NEW_VERSION}"
    # 7. build.sh 呼び出し
    "${SCRIPT_DIR}/build.sh" "$TOOL_ID"
    # 8. release.sh 呼び出し
    "${SCRIPT_DIR}/release.sh" "$TOOL_ID" "$NEW_VERSION"
    # 9. publish.sh 呼び出し
    "${SCRIPT_DIR}/publish.sh" "$TOOL_ID" "$NEW_VERSION"
    # 10. 完了メッセージ
    pass "Successfully uploaded ${TOOL_ID} ${NEW_VERSION} to GitHub!"
    ```

---

### README — ドキュメント更新

#### [MODIFY] [README.md](file://README.md)
*   **Description**: 「Build from Source」セクション (L140-146) のスクリプト参照を更新
*   **変更箇所**:

```diff
 # Bootstrap: build and install all tools
-./scripts/dist/bootstrap-tools
+./scripts/dist/bootstrap-tools.sh

 # Or build individually
-./scripts/dist/build devctl
-./scripts/dist/install-tools devctl
+./scripts/dist/build.sh devctl
+./scripts/dist/install-tools.sh devctl
```

#### [MODIFY] [scripts/dist/README.md](file://scripts/dist/README.md)
*   **Description**: スクリプト一覧テーブル、Release Workflow、Artifact Flow を更新し、`github-upload.sh` を追記
*   **変更箇所**:

1. **Scripts テーブル** — 全スクリプト名を `.sh` 付きに更新 + `github-upload.sh` 行を追加:

```markdown
| Script | Description | Usage |
|--------|-------------|-------|
| `_lib.sh` | Common library (sourced by all scripts) | — |
| `build.sh` | Build CLI tools from features | `./scripts/dist/build.sh <tool-id>` |
| `release.sh` | Create release artifacts | `./scripts/dist/release.sh <tool-id> <version>` |
| `publish.sh` | Publish to GitHub Releases | `./scripts/dist/publish.sh <tool-id> <version>` |
| `github-upload.sh` | All-in-one: build + release + publish | `./scripts/dist/github-upload.sh <tool-id> [version]` |
| `dev.sh` | Launch development environments | `./scripts/dist/dev.sh <feature-name>` |
| `install-tools.sh` | Install developer tools locally | `./scripts/dist/install-tools.sh [--all \| <tool-id>...]` |
| `bootstrap-tools.sh` | Initial setup for new developers | `./scripts/dist/bootstrap-tools.sh` |
```

2. **Release Workflow セクション** — コマンド例を `.sh` 付きに更新 + Quick Release 追記:

```markdown
### Quick Release (All-in-one)

Build, release, and publish in a single command:

\```bash
# Patch release (+0.0.1)
./scripts/dist/github-upload.sh devctl

# Specific version
./scripts/dist/github-upload.sh devctl v2.0.0

# Increment version
./scripts/dist/github-upload.sh devctl +v0.1.0
\```
```

3. **Artifact Flow** — `github-upload.sh` を図に追加:

```
features/
     ↓
tools/manifests/
     ↓
scripts/dist/build.sh
     ↓
dist/
     ↓
scripts/dist/release.sh
     ↓
packaging/
     ↓
scripts/dist/publish.sh
     ↓
Homebrew / Scoop / GitHub Releases

── Or all-in-one ──
scripts/dist/github-upload.sh → build.sh → release.sh → publish.sh
```

## Step-by-Step Implementation Guide

1.  **既存スクリプトのリネーム**:
    *   `git mv scripts/dist/build scripts/dist/build.sh`
    *   `git mv scripts/dist/release scripts/dist/release.sh`
    *   `git mv scripts/dist/publish scripts/dist/publish.sh`
    *   `git mv scripts/dist/dev scripts/dist/dev.sh`
    *   `git mv scripts/dist/install-tools scripts/dist/install-tools.sh`
    *   `git mv scripts/dist/bootstrap-tools scripts/dist/bootstrap-tools.sh`

2.  **スクリプト内の相互参照を更新**:
    *   `dev.sh` 内: `"${SCRIPT_DIR}/install-tools"` → `"${SCRIPT_DIR}/install-tools.sh"`
    *   `install-tools.sh` 内: `"${SCRIPT_DIR}/build"` → `"${SCRIPT_DIR}/build.sh"`
    *   `bootstrap-tools.sh` 内: `"${SCRIPT_DIR}/build"` → `"${SCRIPT_DIR}/build.sh"`, `"${SCRIPT_DIR}/install-tools"` → `"${SCRIPT_DIR}/install-tools.sh"`

3.  **`github-upload.sh` を新規作成**:
    *   `scripts/dist/github-upload.sh` を作成
    *   `_lib.sh` をソースし、共通関数（`info`, `pass`, `fail`, `warn`, `REPO_ROOT` 等）を利用
    *   上記 Technical Design / Logic セクションに基づいて実装
    *   `chmod +x scripts/dist/github-upload.sh` で実行権限を付与

4.  **トップフォルダ `README.md` を更新**:
    *   Build from Source セクションのスクリプト参照を `.sh` 付きに更新

5.  **`scripts/dist/README.md` を更新**:
    *   Scripts テーブル、Release Workflow、Artifact Flow を上記の通り更新

6.  **構文チェック**:
    *   `bash -n scripts/dist/github-upload.sh` で構文エラーがないことを確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **全スクリプトの構文チェック**:
    ```bash
    bash -n ./scripts/dist/github-upload.sh
    bash -n ./scripts/dist/build.sh
    bash -n ./scripts/dist/release.sh
    bash -n ./scripts/dist/publish.sh
    bash -n ./scripts/dist/dev.sh
    bash -n ./scripts/dist/install-tools.sh
    bash -n ./scripts/dist/bootstrap-tools.sh
    ```

3.  **スクリプト間参照の整合性**:
    ```bash
    # リネーム前の拡張子なしファイル名が参照されていないことを確認
    # _lib.sh 内の参照は除外（_lib.sh は他スクリプトから source されるだけで参照しない）
    grep -rn '"${SCRIPT_DIR}/build"' ./scripts/dist/ || echo "OK: no old build refs"
    grep -rn '"${SCRIPT_DIR}/install-tools"' ./scripts/dist/ || echo "OK: no old install-tools refs"
    grep -rn '"${SCRIPT_DIR}/bootstrap-tools"' ./scripts/dist/ || echo "OK: no old bootstrap-tools refs"
    ```

### Manual Verification

> [!IMPORTANT]
> 以下は実際の GitHub Releases への公開を伴うため、ユーザーの判断で実施する。

1. **ヘルプ表示確認**: `./scripts/dist/github-upload.sh` を引数なしで実行し、Usage が表示されることを確認
2. **不正バージョン拒否**: `./scripts/dist/github-upload.sh devctl v1.0` を実行しエラーになることを確認
3. **実際のアップロード**: ユーザー判断で実施

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: Build from Source セクションのスクリプト参照を `.sh` 拡張子付きに更新

#### [MODIFY] [scripts/dist/README.md](file://scripts/dist/README.md)
*   **更新内容**: スクリプト名の `.sh` 化、`github-upload.sh` の追記、Release Workflow・Artifact Flow の更新
