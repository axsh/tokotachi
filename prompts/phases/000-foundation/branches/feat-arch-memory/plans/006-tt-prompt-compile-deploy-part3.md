# 006-tt-prompt-compile-deploy-part3

> **Source Specification**: [006-tt-prompt-compile-deploy.md](file://prompts/phases/000-foundation/branches/feat-arch-memory/ideas/006-tt-prompt-compile-deploy.md)

## Goal Description

Part 1-2 で構築した `tt prompt` サブコマンド群を、既存のシェルスクリプトとプロンプトマニフェストに統合する。スクリプトの書き換え、architecture-maintainer スキルの簡潔化、prompt-update プロシージャの追加、ドキュメント更新を行う。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R5. スクリプト書き換え | Proposed Changes > scripts/prompt/* |
| R8. architecture-maintainer 簡潔化 | Proposed Changes > prompts/manifest/code_content/capabilities/* |
| R9. prompt-update プロシージャ | 既に作成済み（仕様書レビュー時に作成） |
| INV-013 ドキュメント更新 | Proposed Changes > docs/manual/tt-user-manual.md |

## Proposed Changes

### scripts/prompt/（スクリプト書き換え）

#### [MODIFY] [_resolve_tool.sh](file://scripts/prompt/_resolve_tool.sh)
*   **Description**: `agentctl` を探索するロジックを `tt` を探索するロジックに書き換える。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # Common tool discovery logic for tt prompt wrapper scripts.
    # Source this file, then use $TOOL variable.

    _resolve_tt() {
        if [ -n "${TT_TOOL:-}" ]; then
            echo "$TT_TOOL"
            return 0
        fi
        if command -v tt &>/dev/null; then
            echo "tt"
            return 0
        fi
        # Check project-local bin/
        local script_dir
        script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        local project_root
        project_root="$(cd "$script_dir/../.." && pwd)"
        local local_bin="$project_root/bin/tt"
        if [ -x "$local_bin" ]; then
            echo "$local_bin"
            return 0
        fi
        echo "Skipping coding agent settings update: tt tool not found." >&2
        echo "Set TT_TOOL env var, add tt to PATH, or place it in bin/" >&2
        return 1
    }

    TOOL="$(_resolve_tt)" || exit 1
    ```
*   **Logic**: `AGENTCTL` -> `TT_TOOL`, `agentctl` -> `tt`, `bin/agentctl` -> `bin/tt` に全面変更。

#### [MODIFY] [compile.sh](file://scripts/prompt/compile.sh)
*   **Description**: `agentctl compile` 呼び出しを `tt prompt compile` に変更。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # scripts/prompt/compile.sh -- tt prompt compile wrapper
    set -euo pipefail

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/_resolve_tool.sh"

    exec "$TOOL" prompt compile "$@"
    ```

#### [MODIFY] [deploy.sh](file://scripts/prompt/deploy.sh)
*   **Description**: `agentctl deploy` 呼び出しを `tt prompt deploy` に変更。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # scripts/prompt/deploy.sh -- tt prompt deploy wrapper
    set -euo pipefail

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/_resolve_tool.sh"

    exec "$TOOL" prompt deploy "$@"
    ```

#### [NEW] [update.sh](file://scripts/prompt/update.sh)
*   **Description**: `tt prompt update` のラッパースクリプト。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # scripts/prompt/update.sh -- tt prompt update wrapper
    set -euo pipefail

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/_resolve_tool.sh"

    exec "$TOOL" prompt update "$@"
    ```

---

### prompts/manifest/code_content/capabilities/（スキル簡潔化）

#### [MODIFY] [skill.md](file://prompts/manifest/code_content/capabilities/architecture-maintainer/skill.md)
*   **Description**: compile + deploy の2ステップを `update --force --target "{{target:name}}"` の1ステップに統合。
*   **Technical Design**: L37-39 を以下に変更:
    ```markdown
    - If you added, modified, or deleted any memory documents in `prompts/memory/` (including content or frontmatter changes),
      you MUST run `./scripts/prompt/update.sh --force --target "{{target:name}}"` to compile and deploy the memory documents
      (this converts the information in the `prompts/memory/` folder into settings for the coding agent and deploys them).
    ```
*   **Logic**: 2行の手順（compile -> deploy）を1行に統合。`{{target:name}}` テンプレート変数により、コンパイル時に各ターゲット名に展開される。

---

### prompts/manifest/code_content/procedures/（プロシージャ確認）

#### 既に作成済み: [prompt-update.md](file://prompts/manifest/code_content/procedures/prompt-update.md)
*   仕様書レビュー時に作成済み。追加の変更は不要。

---

### docs/manual/（ドキュメント更新）

#### [MODIFY] [tt-user-manual.md](file://docs/manual/tt-user-manual.md)
*   **Description**: `tt prompt` サブコマンドグループのドキュメントを追加。
*   **更新内容**:
    - `tt prompt compile` コマンドの説明、オプション、使用例
    - `tt prompt deploy` コマンドの説明、オプション、使用例
    - `tt prompt update` コマンドの説明、オプション、使用例
    - ターゲット名称解決（前方部分一致、エイリアス）の説明
    - `TT_TARGET` 環境変数の説明

---

### prompts/manifest/code_content/procedures/（既存プロシージャ更新）

#### [MODIFY] [arch-correct.md](file://prompts/manifest/code_content/procedures/arch-correct.md)
*   **Description**: Step 7 の `./scripts/prompt/deploy.sh --force` を `./scripts/prompt/update.sh --force --target "{{target:name}}"` に更新。
*   **Technical Design**: L73-74 を以下に変更:
    ```markdown
    ### 7. Recompile Generated Files
    If frontmatter was changed or a new document was added:
    - Run `./scripts/prompt/update.sh --force --target "{{target:name}}"` to recompile and deploy the updated configuration.
    ```

## Step-by-Step Implementation Guide

### Phase 1: スクリプト書き換え（R5）

1.  **`_resolve_tool.sh` 更新**: `agentctl` -> `tt` に書き換え。
2.  **`compile.sh` 更新**: `exec "$TOOL" compile` -> `exec "$TOOL" prompt compile` に変更。
3.  **`deploy.sh` 更新**: `exec "$TOOL" deploy` -> `exec "$TOOL" prompt deploy` に変更。
4.  **`update.sh` 新規作成**: `tt prompt update` のラッパーを作成。
5.  **実行権限確認**: `chmod +x scripts/prompt/update.sh`
6.  **Git コミット**: `feat: rewrite prompt scripts to use tt command`

### Phase 2: プロンプトマニフェスト更新（R8）

7.  **`architecture-maintainer/skill.md` 更新**: compile+deploy -> update に簡潔化。`{{target:name}}` テンプレート変数を使用。
8.  **`arch-correct.md` 更新**: Step 7 の deploy コマンドを update に変更。
9.  **Git コミット**: `docs: simplify architecture-maintainer skill with tt prompt update`

### Phase 3: ドキュメント更新

10. **`tt-user-manual.md` 更新**: `tt prompt` セクションを追加。
11. **Git コミット**: `docs: add tt prompt commands to user manual`

### Phase 4: コンパイル・デプロイ検証

12. **全体ビルド + 単体テスト**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

13. **実際の compile + deploy 実行**:
    ```bash
    ./scripts/prompt/update.sh --force --target all
    ```
    全ターゲットに対してコンパイル＆デプロイが正常に完了することを確認する。

14. **デプロイ結果の確認**:
    - `.agent/rules/`, `.agent/skills/`, `.agent/workflows/` 以下のファイルが正しく生成されていることを確認
    - `.agent/.meta/last_update.yaml` が作成されていることを確認
    - `.cursor/`, `.claude/`, `.agents/` の各ターゲットディレクトリも同様に確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests** (共通機能のリグレッション確認):
    ```bash
    ./scripts/process/integration_test.sh --categories "common"
    ```

3.  **End-to-End 検証** (スクリプト経由):
    ```bash
    # compile のテスト
    ./scripts/prompt/compile.sh --dry-run

    # update のテスト（全ターゲット）
    ./scripts/prompt/update.sh --force --target all

    # 再実行でスキップされることの確認
    ./scripts/prompt/update.sh --target all
    ```
    *   **Log Verification**: 2回目の実行で「No changes detected. Skipping update.」と表示されることを確認。

### テスト項目のセルフレビュー

1.  **網羅性の検証**: Part 3 の検証は主にスクリプト経由のE2E実行で行う。スクリプト書き換えの正確性は、実際の compile/deploy 実行結果で検証する。
2.  **証拠の十分性**: デプロイ結果のファイル生成確認、メタデータファイルの存在確認で検証する。
3.  **迂回排除**: スクリプトが内部で正しく `tt prompt` を呼び出していることを、`--dry-run` の出力で確認する。

## 総合判定

Part 1-3 の全てが完了した後、以下の総合検証を実施する:

1.  **全体ビルド**: `./scripts/process/build.sh --skip-frontend --skip-etc`
2.  **統合テスト**: `./scripts/process/integration_test.sh --categories "common"`
3.  **フルサイクル検証**:
    ```bash
    # Clean state からの update
    ./scripts/prompt/update.sh --force --target all

    # 個別ターゲットの update
    ./scripts/prompt/update.sh --target anti

    # compile の dry-run
    ./scripts/prompt/compile.sh --target cursor --dry-run

    # deploy の確認
    ./scripts/prompt/deploy.sh --target claude-code --force
    ```
4.  **テンプレート変数の検証**: コンパイルされたスキルファイル内で `{{target:name}}` が正しく展開されていることを確認。
