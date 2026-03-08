# 001-GitHub-Upload-VersionBump-Fix

> **Source Specification**: [001-GitHub-Upload-VersionBump-Fix.md](file://prompts/phases/000-foundation/ideas/bump-up-release-to-github/001-GitHub-Upload-VersionBump-Fix.md)

## Goal Description

`scripts/dist/github-upload.sh` の引数解析ロジックを改修し、tool-id 省略時の自動検出と `--dry-run` オプションを追加する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| 1. tool-id 省略時の自動判定 | Proposed Changes > `_lib.sh` (`resolve_single_tool_id`) |
| 2. 引数のスマート解析 | Proposed Changes > `github-upload.sh` (引数解析セクション) |
| 3. 曖昧さの排除（複数ツール時エラー） | Proposed Changes > `_lib.sh` (`resolve_single_tool_id`) |
| 4. 後方互換性 | Proposed Changes > `github-upload.sh` (ARGS=2 の分岐) |
| 5. `--dry-run` オプション | Proposed Changes > `github-upload.sh` (dry-run セクション) |
| 6. 他のスクリプトへの波及なし | 変更対象外（`build.sh`, `release.sh`, `publish.sh` は変更しない） |

## Proposed Changes

### Distribution Scripts

#### [MODIFY] [_lib.sh](file://scripts/dist/_lib.sh)
*   **Description**: `is_version_arg` ヘルパー関数と `resolve_single_tool_id` 関数を追加する。これらは `github-upload.sh` から呼び出される汎用関数。
*   **Technical Design**:
    ```bash
    # バージョンパターン判定
    # 引数が vN.N.N または +vN.N.N 形式にマッチする場合 true を返す
    is_version_arg() {
      [[ "$1" =~ ^\+?v[0-9]+\.[0-9]+\.[0-9]+$ ]]
    }

    # ツールが1つだけ登録されている場合、その tool-id を返す
    # 複数登録されている場合はエラーメッセージを出して exit 1
    resolve_single_tool_id() {
      local ids count
      ids=$(get_all_tool_ids | tr -d '\r')
      count=$(echo "$ids" | wc -l | tr -d ' ')

      if [[ $count -eq 1 ]]; then
        echo "$ids"
      else
        fail "Multiple tools registered. Please specify tool-id explicitly."
        echo "  Available tools:" >&2
        echo "$ids" | sed 's/^/    - /' >&2
        exit 1
      fi
    }
    ```
*   **Logic**:
    *   `is_version_arg`: 正規表現 `^\+?v[0-9]+\.[0-9]+\.[0-9]+$` にマッチするかテスト
    *   `resolve_single_tool_id`: `get_all_tool_ids` の出力行数をカウントし、1行なら出力、複数行ならエラー
    *   Windows 環境での `\r` 混入を考慮して `tr -d '\r'` を適用

---

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **Description**: 引数解析セクション (L108-L130) を3段階のスマート解析に置き換え、`--dry-run` オプションの処理を追加する。
*   **Technical Design**:

    **Part A: `--dry-run` フラグ抽出（L108-L118 の既存 Usage セクションを置換）**

    引数ループで `--dry-run` を抽出し、残りを `ARGS` 配列に格納:

    ```bash
    DRY_RUN=false
    ARGS=()
    for arg in "$@"; do
      if [[ "$arg" == "--dry-run" ]]; then
        DRY_RUN=true
      else
        ARGS+=("$arg")
      fi
    done
    ```

    **Part B: スマート引数解析（既存の L120-L130 を置換）**

    `ARGS` の要素数に応じた分岐:

    ```bash
    case ${#ARGS[@]} in
      0)
        # tool-id 自動解決、デフォルト patch bump
        TOOL_ID=$(resolve_single_tool_id)
        VERSION_ARG="+v0.0.1"
        ;;
      1)
        if is_version_arg "${ARGS[0]}"; then
          # 第1引数がバージョン → tool-id を自動解決
          TOOL_ID=$(resolve_single_tool_id)
          VERSION_ARG="${ARGS[0]}"
        else
          # 第1引数が tool-id → デフォルト patch bump
          TOOL_ID="${ARGS[0]}"
          VERSION_ARG="+v0.0.1"
        fi
        ;;
      2)
        # 従来通り: $1=tool-id, $2=バージョン
        TOOL_ID="${ARGS[0]}"
        VERSION_ARG="${ARGS[1]}"
        ;;
      *)
        echo "Usage: $0 [--dry-run] [tool-id] [version|+increment]"
        exit 1
        ;;
    esac
    ```

    **Part C: Usage メッセージの更新**

    ファイル先頭のコメントと Usage メッセージを新しい引数仕様に合わせて更新:

    ```bash
    # Usage: ./scripts/dist/github-upload.sh [--dry-run] [tool-id] [version|+increment]
    ```

    **Part D: `--dry-run` ガード（バージョン比較後、Step 1 の前に挿入）**

    ```bash
    if [[ "$DRY_RUN" == true ]]; then
      echo ""
      info "=== Dry Run ==="
      info "Tool:    ${TOOL_ID}"
      info "Current: ${CURRENT_VERSION}"
      info "New:     ${NEW_VERSION}"
      info "Mode:    ${MODE}"
      info "No changes were made."
      exit 0
    fi
    ```

*   **Logic**:
    1.  全引数をループし、`--dry-run` を `DRY_RUN=true` として抽出、残りを `ARGS` 配列へ
    2.  `ARGS` 要素数で分岐:
        *   0個: `resolve_single_tool_id` で tool-id 解決、`+v0.0.1` をデフォルト
        *   1個: `is_version_arg` でパターン判定。マッチなら tool-id 自動解決＋その値をバージョンに。非マッチなら tool-id として扱い＋デフォルトバージョン
        *   2個: 従来通り `$1=tool-id`, `$2=バージョン`
        *   3個以上: Usage 表示して終了
    3.  既存のバージョン判定・取得・計算・比較ロジック（`validate_version_format`, `get_current_version`, `compute_incremented_version`, `compare_versions`）はそのまま使用
    4.  `DRY_RUN=true` の場合、確認メッセージ表示後に `exit 0` で終了（Step 1〜3 をスキップ）

## Step-by-Step Implementation Guide

1.  **`_lib.sh` にヘルパー関数を追加**:
    *   `scripts/dist/_lib.sh` の末尾に `is_version_arg` 関数を追加
    *   同じく末尾に `resolve_single_tool_id` 関数を追加

2.  **`github-upload.sh` のコメントと Usage を更新**:
    *   ファイル先頭のコメント (L2-L7) を新しい引数仕様に書き換え
    *   既存の Usage 表示セクション (L110-L118) を新しい形式に更新

3.  **`github-upload.sh` の引数解析を置き換え**:
    *   既存の引数解析セクション (L108-L130) を削除し、Part A (`--dry-run` 抽出) + Part B (スマート引数解析) に置き換え

4.  **`github-upload.sh` に `--dry-run` ガードを挿入**:
    *   バージョン比較 (`compare_versions`) の直後、確認表示セクションの前 (L152 付近) に Part D を挿入

5.  **`--dry-run` で動作確認**:
    *   `./scripts/dist/github-upload.sh --dry-run +v0.1.0` を実行して以下を確認:
        *   `Tool: devctl` と表示されること
        *   `Current:` が GitHub Releases の最新バージョンであること
        *   `New:` が Current の minor を +1 した値であること
        *   `Mode: increment` と表示されること
    *   `./scripts/dist/github-upload.sh --dry-run devctl +v0.1.0` を実行して同じ結果になることを確認
    *   `./scripts/dist/github-upload.sh --dry-run v1.0.0` を実行して `Mode: absolute` になることを確認
    *   `./scripts/dist/github-upload.sh --dry-run` を実行してデフォルト patch bump になることを確認

## Verification Plan

### Automated Verification

本変更はシェルスクリプトのみの修正であり、Go コードへの影響はない。Go のビルド・テストは不要。

`--dry-run` オプションを使って、実際のビルド・リリース・パブリッシュを行わずに引数解析ロジックの正しさを検証する。

**テスト1: tool-id 省略 + minor インクリメント**

```bash
./scripts/dist/github-upload.sh --dry-run +v0.1.0
```

*   **確認項目**: `Tool: devctl`, `Mode: increment`, `New:` が Current の minor +1

**テスト2: tool-id 省略 + 絶対バージョン**

```bash
./scripts/dist/github-upload.sh --dry-run v1.0.0
```

*   **確認項目**: `Tool: devctl`, `Mode: absolute`, `New: v1.0.0`

**テスト3: 明示的 tool-id（後方互換）**

```bash
./scripts/dist/github-upload.sh --dry-run devctl +v0.1.0
```

*   **確認項目**: テスト1 と同じ結果

**テスト4: 引数なし（デフォルト patch bump）**

```bash
./scripts/dist/github-upload.sh --dry-run
```

*   **確認項目**: `Tool: devctl`, `Mode: increment`, `New:` が Current の patch +1

## Documentation

#### [MODIFY] [github-upload.sh](file://scripts/dist/github-upload.sh)
*   **更新内容**: ファイル先頭のコメント（Usage/Examples）を新しい引数仕様に合わせて更新済み（実装に含まれる）
