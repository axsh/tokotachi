# 005-Far-Knowledge-Skillification-Part2

> **Source Specification**: [005-Far-Knowledge-Skillification.md](../ideas/005-Far-Knowledge-Skillification.md)

## Goal Description

Part2 では Wrapper スクリプトの整理 (廃止・改名・新設) と、プロンプト/ワークフロー/capability/policy の改名・内容改修を実装する。

## User Review Required

> [!WARNING]
> `record-architecture-knowledge.md` -> `record-far-knowledge.md` の改名は、既存のワークフローから参照されているため、全参照箇所の更新が必須。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: Wrapper notify.sh -> record.sh | Proposed Changes > scripts/code/agent/ |
| R2: capability 改名 (record-architecture-knowledge -> record-far-knowledge) | Proposed Changes > capabilities/ |
| R2: capability 改名 (pre-push-architecture-check -> pre-push-knowledge-check) | Proposed Changes > capabilities/ |
| R2: policy 改名 (architecture-memory -> far-knowledge-memory) | Proposed Changes > policies/ |
| R3: systematize-far-knowledge ワークフロー新設 | Proposed Changes > procedures/ |
| R4: execute-implementation-plan Section 3.3 改訂 | Proposed Changes > procedures/ |
| Wrapper assist.sh 廃止 | Proposed Changes > scripts/ |
| Wrapper task.sh 廃止 | Proposed Changes > scripts/ |
| Wrapper knowledge.sh 新設 | Proposed Changes > scripts/ |

## Proposed Changes

### 1. Wrapper スクリプトの整理 (`scripts/code/agent/`)

#### [DELETE] [assist.sh](file://scripts/code/agent/assist.sh)
*   **Description**: `tt agent assist` 廃止に伴い削除。

#### [DELETE] [task.sh](file://scripts/code/agent/task.sh)
*   **Description**: `tt agent task` 廃止に伴い削除。

#### [NEW] [record.sh](file://scripts/code/agent/record.sh)
*   **Description**: `notify.sh` を `record.sh` に改名。内部で `tt agent record` を呼び出す。新規フラグ対応。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # scripts/code/agent/record.sh -- tt agent record wrapper
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/../_resolve_tool.sh"

    TT_ARGS=()
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --agent)            TT_ARGS+=(--agent "$2");            shift 2 ;;
        --summary)          TT_ARGS+=(--summary "$2");          shift 2 ;;
        --summary-file)     TT_ARGS+=(--summary-file "$2");     shift 2 ;;
        --note)             TT_ARGS+=(--note "$2");             shift 2 ;;
        --notes-file)       TT_ARGS+=(--notes-file "$2");       shift 2 ;;
        --changed-path)     TT_ARGS+=(--changed-path "$2");     shift 2 ;;
        --changed-paths-from-git) TT_ARGS+=(--changed-paths-from-git); shift ;;
        # 既存フラグ
        --architecture-impact)    TT_ARGS+=(--architecture-impact);    shift ;;
        --memory-related)         TT_ARGS+=(--memory-related);         shift ;;
        --prompt-related)         TT_ARGS+=(--prompt-related);         shift ;;
        --agent-behavior-related) TT_ARGS+=(--agent-behavior-related); shift ;;
        --requires-immediate-action) TT_ARGS+=(--requires-immediate-action); shift ;;
        # 新規フラグ (R1)
        --design-pattern)         TT_ARGS+=(--design-pattern);         shift ;;
        --convention)             TT_ARGS+=(--convention);             shift ;;
        --lesson-learned)         TT_ARGS+=(--lesson-learned);         shift ;;
        --preference)             TT_ARGS+=(--preference);             shift ;;
        # メタデータ
        --client-request-id) TT_ARGS+=(--client-request-id "$2"); shift 2 ;;
        --dry-run)          TT_ARGS+=(--dry-run);               shift ;;
        --print-payload)    TT_ARGS+=(--print-payload);         shift ;;
        *)
          echo "[ERROR] Unknown argument: $1" >&2
          exit 1
          ;;
      esac
    done

    exec "$TOOL" agent record "${TT_ARGS[@]}"
    ```

#### [DELETE] [notify.sh](file://scripts/code/agent/notify.sh)
*   **Description**: `record.sh` に移行したため削除。

#### [MODIFY] [intake.sh](file://scripts/code/agent/intake.sh)
*   **Description**: `processed` サブコマンドを追加。
*   **Technical Design**:
    ```bash
    # case に追加:
    processed)
        if [[ $# -lt 1 ]]; then
          echo "Usage: intake.sh processed <event-id>" >&2
          exit 1
        fi
        EVENT_ID="$1"
        exec "$TOOL" agent intake processed "$EVENT_ID"
        ;;
    ```
*   **Logic**: Usage 表示も `list|show|processed` に更新。

#### [NEW] [knowledge.sh](file://scripts/code/agent/knowledge.sh)
*   **Description**: `tt agent knowledge` の Wrapper。
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # scripts/code/agent/knowledge.sh -- tt agent knowledge wrapper
    set -euo pipefail
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "$SCRIPT_DIR/../_resolve_tool.sh"

    if [[ $# -lt 1 ]]; then
      echo "Usage: knowledge.sh <add|append|list|split|merge|rename|move> [OPTIONS]" >&2
      exit 1
    fi

    SUBCMD="$1"
    shift

    TT_ARGS=()
    case "$SUBCMD" in
      add)
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --category-path)  TT_ARGS+=(--category-path "$2");  shift 2 ;;
            --title)          TT_ARGS+=(--title "$2");          shift 2 ;;
            --description)    TT_ARGS+=(--description "$2");    shift 2 ;;
            --content-file)   TT_ARGS+=(--content-file "$2");   shift 2 ;;
            --source-events)  TT_ARGS+=(--source-events "$2");  shift 2 ;;
            *) echo "[ERROR] Unknown argument for add: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge add "${TT_ARGS[@]}"
        ;;
      append)
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --category-path)  TT_ARGS+=(--category-path "$2");  shift 2 ;;
            --title)          TT_ARGS+=(--title "$2");          shift 2 ;;
            --content-file)   TT_ARGS+=(--content-file "$2");   shift 2 ;;
            --source-events)  TT_ARGS+=(--source-events "$2");  shift 2 ;;
            *) echo "[ERROR] Unknown argument for append: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge append "${TT_ARGS[@]}"
        ;;
      list)
        exec "$TOOL" agent knowledge list
        ;;
      split)
        CATEGORY_PATH="$1"; shift
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --into)  shift; while [[ $# -gt 0 && "$1" != --* ]]; do TT_ARGS+=(--into "$1"); shift; done ;;
            --plan)  TT_ARGS+=(--plan "$2");  shift 2 ;;
            *) echo "[ERROR] Unknown argument for split: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge split "$CATEGORY_PATH" "${TT_ARGS[@]}"
        ;;
      merge)
        # merge <cat1> <cat2> --into <new> --plan <file>
        CATS=()
        while [[ $# -gt 0 && "$1" != --* ]]; do CATS+=("$1"); shift; done
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --into)  TT_ARGS+=(--into "$2");  shift 2 ;;
            --plan)  TT_ARGS+=(--plan "$2");  shift 2 ;;
            --title) TT_ARGS+=(--title "$2"); shift 2 ;;
            *) echo "[ERROR] Unknown argument for merge: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge merge "${CATS[@]}" "${TT_ARGS[@]}"
        ;;
      rename)
        OLD_PATH="$1"; NEW_PATH="$2"; shift 2
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --title) TT_ARGS+=(--title "$2"); shift 2 ;;
            *) echo "[ERROR] Unknown argument for rename: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge rename "$OLD_PATH" "$NEW_PATH" "${TT_ARGS[@]}"
        ;;
      move)
        while [[ $# -gt 0 ]]; do
          case "$1" in
            --from) TT_ARGS+=(--from "$2"); shift 2 ;;
            --to)   TT_ARGS+=(--to "$2");   shift 2 ;;
            *) echo "[ERROR] Unknown argument for move: $1" >&2; exit 1 ;;
          esac
        done
        exec "$TOOL" agent knowledge move "${TT_ARGS[@]}"
        ;;
      *)
        echo "[ERROR] Unknown subcommand: $SUBCMD. Use 'add', 'append', 'list', 'split', 'merge', 'rename', or 'move'." >&2
        exit 1
        ;;
    esac
    ```

---

### 2. capability / policy の改名・改修

#### [NEW] [record-far-knowledge.md](file://prompts/manifest/code_content/capabilities/record-far-knowledge.md)
*   **Description**: `record-architecture-knowledge.md` を改名・改修。
*   **Logic**:
    *   タイトル: "Record Far-Knowledge" (旧: "Record Architecture Knowledge")
    *   description: 「遠方知識全般」を対象とする旨を明記
    *   コマンド参照: `./scripts/code/agent/notify.sh` -> `./scripts/code/agent/record.sh` に全て変更
    *   フラグガイド: 既存5フラグ + 新規4フラグ (`--design-pattern`, `--convention`, `--lesson-learned`, `--preference`) の使い分けを追加
    *   距離判定ガイドライン (仕様 R1 の「距離判定のガイドライン」) をスキル本文に含める

#### [DELETE] [record-architecture-knowledge.md](file://prompts/manifest/code_content/capabilities/record-architecture-knowledge.md)
*   **Description**: `record-far-knowledge.md` に移行したため削除。

#### [NEW] [pre-push-knowledge-check.md](file://prompts/manifest/code_content/capabilities/pre-push-knowledge-check.md)
*   **Description**: `pre-push-architecture-check.md` を改名・改修。
*   **Logic**:
    *   タイトル: "Pre-Push Knowledge Check" (旧: "Pre-Push Architecture Check")
    *   対象を「アーキテクチャ」から「遠方知識全般」に拡大
    *   Step 4 の参照先を `record-far-knowledge` に変更

#### [DELETE] [pre-push-architecture-check.md](file://prompts/manifest/code_content/capabilities/pre-push-architecture-check.md)

#### [NEW] [far-knowledge-memory.md](file://prompts/manifest/code_content/policies/far-knowledge-memory.md)
*   **Description**: `architecture-memory.md` を改名・改修。
*   **Logic**:
    *   タイトル: "Far-Knowledge Memory Policy" (旧: "Architecture Memory Policy")
    *   ポリシー対象を「アーキテクチャ」から「遠方知識全般」に拡大
    *   コマンド参照を `record.sh` に統一

#### [DELETE] [architecture-memory.md](file://prompts/manifest/code_content/policies/architecture-memory.md)

---

### 3. ワークフロー/プロシージャの改修

#### [MODIFY] [execute-implementation-plan.md](file://prompts/manifest/code_content/procedures/execute-implementation-plan.md)
*   **Description**: Section 3.3 を改訂。
*   **Logic**:
    *   旧: `### 3.3 アーキテクチャメモリの記録 (Architecture Memory Intake)` + `record-architecture-knowledge` 参照
    *   新: `### 3.3 遠方知識の記録 (Far-Knowledge Recording)` + `record-far-knowledge` 参照 + 「体系化・スキル化は別途 systematize-far-knowledge ワークフローで実施する」の注記追加

#### [NEW] [systematize-far-knowledge.md](file://prompts/manifest/code_content/procedures/systematize-far-knowledge.md)
*   **Description**: 体系化ワークフローの新規作成。
*   **Technical Design**:
    ```yaml
    ---
    apiVersion: agent.meta/v1
    kind: procedure
    id: systematize-far-knowledge
    title: "遠方知識の体系化"
    description: >-
      pending intake events を確認し、遠方知識をカテゴリ化・体系化・スキル化する。
      実行タイミング: ユーザーが明示的に指示した時。
    triggers:
      - "slash_command"
    ---
    ```
*   **Logic**: 仕様 R3 のステップ1-10をワークフロー手順として記述:
    1. `intake.sh list --status pending` で pending events 確認
    2. 0件ならスキップ報告
    3. `knowledge.sh list` で既存カテゴリツリー確認
    4. 各 pending event の距離判定 (LLM)
    5. カテゴリ判定 -> `knowledge.sh add` or `knowledge.sh append`
    6. 再整理判断 -> 必要なら `knowledge.sh split/merge/rename/move`
    7. `intake.sh processed <event-id>` で processed 移行
    8. カテゴリ別にスキル化検討 (LLM) -> capability ファイル生成
    9. `prompts/memory/branches/<branch-package-id>/skills/` に配置
    10. `update.sh` でデプロイ

---

## Step-by-Step Implementation Guide

1.  **Wrapper スクリプトの廃止**:
    *   `scripts/code/agent/assist.sh` を削除
    *   `scripts/code/agent/task.sh` を削除

2.  **notify.sh -> record.sh の改名**:
    *   `scripts/code/agent/record.sh` を新規作成 (新規フラグ対応)
    *   `scripts/code/agent/notify.sh` を削除

3.  **intake.sh の拡張**:
    *   `scripts/code/agent/intake.sh` に `processed` サブコマンドを追加

4.  **knowledge.sh の新設**:
    *   `scripts/code/agent/knowledge.sh` を新規作成

5.  **capability の改名・改修**:
    *   `prompts/manifest/code_content/capabilities/record-far-knowledge.md` を作成
    *   `prompts/manifest/code_content/capabilities/record-architecture-knowledge.md` を削除
    *   `prompts/manifest/code_content/capabilities/pre-push-knowledge-check.md` を作成
    *   `prompts/manifest/code_content/capabilities/pre-push-architecture-check.md` を削除

6.  **policy の改名・改修**:
    *   `prompts/manifest/code_content/policies/far-knowledge-memory.md` を作成
    *   `prompts/manifest/code_content/policies/architecture-memory.md` を削除

7.  **ワークフローの改修**:
    *   `prompts/manifest/code_content/procedures/execute-implementation-plan.md` の Section 3.3 を改訂
    *   `prompts/manifest/code_content/procedures/systematize-far-knowledge.md` を新規作成

8.  **compile + deploy 確認**:
    *   `./scripts/code/prompt/update.sh --dry-run` でエラーがないこと確認
    *   `./scripts/code/prompt/update.sh` で実際にデプロイ
    *   `.agents/workflows/systematize-far-knowledge.md` が生成されること確認
    *   `.agents/skills/record-far-knowledge/SKILL.md` が生成されること確認

9.  **Verification Plan の実行** (後述)

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests** (Part1 変更との統合確認):
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Prompt Compile & Deploy**:
    ```bash
    ./scripts/code/prompt/update.sh --dry-run
    ./scripts/code/prompt/update.sh
    ```
    *   **Log Verification**:
        *   `record-far-knowledge` スキルがデプロイされること
        *   `pre-push-knowledge-check` スキルがデプロイされること
        *   `systematize-far-knowledge` ワークフローがデプロイされること
        *   旧名 (`record-architecture-knowledge` 等) のファイルがデプロイ先に残っていないこと

3.  **Wrapper スクリプトの動作確認** (Part1 ビルド後):
    ```bash
    ./scripts/code/agent/record.sh --agent antigravity --summary "test" --note "test note" --design-pattern --dry-run
    ./scripts/code/agent/intake.sh list --status pending
    ./scripts/code/agent/knowledge.sh list
    ```

4.  **最終全体検証**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```

### GUI E2E Tests

GUI関連の変更なし。E2E テスト不要。

## Documentation

#### [MODIFY] [005-Far-Knowledge-Skillification.md](file://prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/005-Far-Knowledge-Skillification.md)
*   **更新内容**: 実装後に検証結果を追記
