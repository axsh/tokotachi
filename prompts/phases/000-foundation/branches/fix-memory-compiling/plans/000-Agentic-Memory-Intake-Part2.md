# 000-Agentic-Memory-Intake-Part2

> **Source Specification**: [000-Agentic-Memory-Intake.md](../ideas/000-Agentic-Memory-Intake.md)

## Goal Description

Part 1 で構築した `tt agent notify` の基盤に加え、以下を実装する:

1. 補助コマンド (`tt agent status`, `tt agent intake list`, `tt agent intake show`)
2. Wrapper スクリプトの移行と新規作成 (`scripts/code/`)
3. Compiler Ignore Hardening (R10)
4. 常駐ポリシー `architecture-memory.md` の新規作成

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R5: Wrapper スクリプト | Proposed Changes > scripts/code/ |
| R6: 補助コマンド (status, intake list/show) | Proposed Changes > cmd/agent_status.go, cmd/agent_intake.go, internal/agent/status/ |
| R10: Compiler Ignore Hardening | Proposed Changes > memory/frontmatter.go, memory/indexer.go |
| R5: `_resolve_tool.sh` graceful skip | Proposed Changes > scripts/code/_resolve_tool.sh |
| プロンプト設計: architecture-memory.md 新規作成 | Proposed Changes > policies/architecture-memory.md |

## Proposed Changes

### 補助コマンド -- ステータス (`features/tt/internal/agent/status/`)

#### [NEW] [status.go](file://features/tt/internal/agent/status/status.go)
*   **Description**: `tt agent status` のロジック -- pending/processed/failed の件数、oldest age、index health を表示
*   **Technical Design**:
    ```go
    package status

    // StatusReport holds the status summary.
    type StatusReport struct {
        PendingCount    int    `json:"pending_count"`
        ProcessedCount  int    `json:"processed_count"`
        FailedCount     int    `json:"failed_count"`
        IgnoredCount    int    `json:"ignored_count"`
        OldestPendingAge string `json:"oldest_pending_age,omitempty"` // human-readable duration
        IndexHealth     string `json:"index_health"` // "ok", "degraded", "unavailable"
        CurrentBranch   string `json:"current_branch,omitempty"`
    }

    // GetStatus computes the status report.
    // varDir is the path to prompts/memory/var/
    func GetStatus(varDir string) (*StatusReport, error)
    ```
*   **Logic**:
    *   `filepath.Walk(intakeDir+"/pending")` でファイル数をカウント
    *   同様に `processed/`, `failed/`, `ignored/` をカウント
    *   pending 内の最も古いファイルの `timestamps.created_at` から age を計算
    *   SQLite DB の存在確認と `PRAGMA integrity_check` で health を判定

#### [NEW] [list.go](file://features/tt/internal/agent/status/list.go)
*   **Description**: `tt agent intake list` のロジック -- 1 行 1 event のテーブル/JSON 表示
*   **Technical Design**:
    ```go
    package status

    // ListOptions holds filter options for listing events.
    type ListOptions struct {
        Status     string // "pending", "processed", "failed", "ignored"
        Agent      string
        Branch     string
        Query      string // FTS query
        PathPrefix string
        From       string // ISO8601
        To         string // ISO8601
        Format     string // "table" or "json"
        Limit      int
    }

    // ListItem represents a single event in list output.
    type ListItem struct {
        EventID     string `json:"event_id"`
        Agent       string `json:"agent"`
        Branch      string `json:"branch"`
        Status      string `json:"status"`
        TaskSummary string `json:"task_summary"`
        CreatedAt   string `json:"created_at"`
    }

    // List retrieves events matching the filter criteria.
    // Default scope is current branch.
    func List(varDir string, opts ListOptions) ([]ListItem, error)
    ```
*   **Logic**:
    *   SQLite index から `SELECT` クエリを組み立て
    *   FTS: `Query` が指定された場合 `intake_events_fts MATCH ?` で検索
    *   `PathPrefix`: `effective_changed_paths` の JSON 内で prefix マッチ
    *   `From`/`To`: `created_at` の範囲フィルタ
    *   既定スコープは `current branch` (git から取得)

#### [NEW] [show.go](file://features/tt/internal/agent/status/show.go)
*   **Description**: `tt agent intake show <event-id>` のロジック -- 完全な IntakeEvent を表示
*   **Technical Design**:
    ```go
    package status

    // Show retrieves a single IntakeEvent by event_id.
    // Reads the JSON file from the stored_at path.
    func Show(varDir, eventID string) (*agent.IntakeEvent, error)
    ```
*   **Logic**:
    *   SQLite index から `stored_at` パスを取得
    *   そのパスの JSON ファイルを読み込み、`IntakeEvent` にデシリアライズ
    *   ファイルが存在しない場合はエラー

---

### コマンド層 (`features/tt/cmd/`)

#### [NEW] [agent_status.go](file://features/tt/cmd/agent_status.go)
*   **Description**: `tt agent status` コマンド
*   **Technical Design**:
    ```go
    package cmd

    var agentStatusCmd = &cobra.Command{
        Use:   "status",
        Short: "Show agent intake status summary",
        RunE:  runAgentStatus,
    }
    ```
*   **Logic**:
    *   `status.GetStatus(varDir)` を呼び出し、結果を JSON またはテーブルで出力

#### [NEW] [agent_intake.go](file://features/tt/cmd/agent_intake.go)
*   **Description**: `tt agent intake list` / `tt agent intake show` コマンド
*   **Technical Design**:
    ```go
    package cmd

    var agentIntakeCmd = &cobra.Command{
        Use:   "intake",
        Short: "Manage intake events",
    }

    var agentIntakeListCmd = &cobra.Command{
        Use:   "list",
        Short: "List intake events",
        RunE:  runAgentIntakeList,
    }

    var agentIntakeShowCmd = &cobra.Command{
        Use:   "show [event-id]",
        Short: "Show a single intake event",
        Args:  cobra.ExactArgs(1),
        RunE:  runAgentIntakeShow,
    }

    // Filter flags
    var (
        intakeListStatus     string
        intakeListAgent      string
        intakeListBranch     string
        intakeListQuery      string
        intakeListPathPrefix string
        intakeListFrom       string
        intakeListTo         string
        intakeListFormat     string // "table" or "json"
    )
    ```
*   **Logic**:
    *   `list`: フィルタフラグを `ListOptions` にマッピングし `status.List()` を呼び出し
    *   `show`: 引数の event-id を `status.Show()` に渡し、結果を JSON で出力

---

### Wrapper スクリプト (`scripts/code/`)

#### [NEW] [_resolve_tool.sh](file://scripts/code/_resolve_tool.sh)
*   **Description**: 共通 tt binary 解決ヘルパー。graceful skip 対応版
*   **Technical Design**:
    ```bash
    #!/usr/bin/env bash
    # Common tool discovery logic for tt wrapper scripts.
    # Source this file, then use $TOOL variable.
    # If tt is not found, exits with 0 (graceful skip) and prints a skip message.

    _resolve_tt() {
        if [ -n "${TT_TOOL:-}" ]; then
            echo "$TT_TOOL"
            return 0
        fi
        if command -v tt &>/dev/null; then
            echo "tt"
            return 0
        fi
        local script_dir
        script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        local project_root
        project_root="$(cd "$script_dir/../.." && pwd)"
        local local_bin="$project_root/bin/tt"
        if [ -x "$local_bin" ]; then
            echo "$local_bin"
            return 0
        fi
        echo "[SKIP] tt tool is not installed. This is expected in some environments." >&2
        echo "[SKIP] Do NOT attempt to install tt. Simply skip this step and continue." >&2
        exit 0
    }

    TOOL="$(_resolve_tt)"
    ```
*   **Logic**:
    *   検索順序: `$TT_TOOL` env -> PATH -> `bin/tt`
    *   見つからない場合は `exit 0` (graceful skip)。stderr に `[SKIP]` メッセージ
    *   `exit 1` ではなく `exit 0` を使用し、Coding Agent が「解くべきエラー」と誤認しないようにする

#### [NEW] [notify.sh](file://scripts/code/agent/notify.sh)
*   **Description**: Coding Agent オプションを `tt agent notify` オプションに明示的にマッピングする shim
*   **Technical Design**: 仕様書 R5 に記載の実装例をそのまま採用:
    ```bash
    #!/usr/bin/env bash
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
        --architecture-impact)    TT_ARGS+=(--architecture-impact);    shift ;;
        --memory-related)         TT_ARGS+=(--memory-related);         shift ;;
        --prompt-related)         TT_ARGS+=(--prompt-related);         shift ;;
        --agent-behavior-related) TT_ARGS+=(--agent-behavior-related); shift ;;
        --requires-immediate-action) TT_ARGS+=(--requires-immediate-action); shift ;;
        --client-request-id) TT_ARGS+=(--client-request-id "$2"); shift 2 ;;
        --dry-run)          TT_ARGS+=(--dry-run);               shift ;;
        --print-payload)    TT_ARGS+=(--print-payload);         shift ;;
        *)
          echo "[ERROR] Unknown argument: $1" >&2
          exit 1
          ;;
      esac
    done

    exec "$TOOL" agent notify "${TT_ARGS[@]}"
    ```
*   **Logic**: 引数を明示的にパースし、TT_ARGS 配列を構築して `tt agent notify` に渡す。`"$@"` パススルーは禁止

#### [NEW] [status.sh](file://scripts/code/agent/status.sh)
*   **Description**: `tt agent status` の wrapper
*   **Technical Design**: `_resolve_tool.sh` を source し、引数を明示的にマッピング

#### [NEW] [intake.sh](file://scripts/code/agent/intake.sh)
*   **Description**: `tt agent intake list` / `tt agent intake show` の wrapper
*   **Technical Design**: `_resolve_tool.sh` を source し、サブコマンドと引数を明示的にマッピング

#### 既存 Wrapper の移行

以下の既存ファイルを `scripts/prompt/` から `scripts/code/prompt/` に移行する:

#### [NEW] [compile.sh](file://scripts/code/prompt/compile.sh)
*   **Description**: `scripts/prompt/compile.sh` の移行。引数を明示的にマッピング

#### [NEW] [deploy.sh](file://scripts/code/prompt/deploy.sh)
*   **Description**: `scripts/prompt/deploy.sh` の移行

#### [NEW] [update.sh](file://scripts/code/prompt/update.sh)
*   **Description**: `scripts/prompt/update.sh` の移行

> **注意**: 既存の `scripts/prompt/` は互換性のために一定期間残すか、全てのプロンプトが `scripts/code/` を参照するように更新された後に削除する。本計画では新規作成のみ行い、既存は削除しない。

---

### Compiler Ignore Hardening (R10)

#### [MODIFY] [frontmatter.go](file://features/tt/internal/prompt/memory/frontmatter.go)
*   **Description**: `ParseAllMemoryDocs` に `var/` と `schemas/` のスキップ処理を追加
*   **Technical Design**:
    ```go
    // shouldSkipMemoryPath returns true if the path should be excluded from
    // memory document parsing. Paths under var/ and schemas/ are excluded.
    func shouldSkipMemoryPath(path string) bool {
        normalized := filepath.ToSlash(path)
        return strings.Contains(normalized, "/var/") ||
               strings.Contains(normalized, "/schemas/")
    }
    ```
*   **Logic**:
    *   `ParseAllMemoryDocs` 内のファイルループ冒頭で `shouldSkipMemoryPath(file)` を呼び、true なら `continue`
    *   `isGeneratedFile` チェックの前に配置 (先に安価なパスチェックで除外)

#### [MODIFY] [indexer.go](file://features/tt/internal/prompt/memory/indexer.go)
*   **Description**: `GenerateIndex` にも同様のスキップ処理を追加 (必要な場合)
*   **Technical Design**: `ParseAllMemoryDocs` で既にフィルタ済みのため、indexer 側は変更不要の可能性が高い。ただし `indexer.go` 内で独自にファイルを走査している箇所がないか確認し、あれば同様のガードを追加

---

### 常駐ポリシー (プロンプト manifest)

#### [NEW] [architecture-memory.md](file://prompts/manifest/code_content/policies/architecture-memory.md)
*   **Description**: Coding Agent がメモリ関連の作業時に参照する常駐ポリシー。旧版は設計変更に伴い削除済みのため新規作成
*   **Technical Design**:
    ```markdown
    ---
    id: architecture-memory
    title: Architecture Memory Policy
    kind: policy
    scope: always
    ---

    Before changing architecture-sensitive code, read prompts/memory/index.md.
    After such changes, update the relevant architecture document.
    If unsure where to write, append to prompts/memory/inbox.md.

    When architecture-impacting or agent-memory-relevant knowledge may have been created,
    run `./scripts/code/agent/notify.sh --agent {{target:name}}` once per coherent task boundary.

    Use notify only to store long-term memory candidates.
    Do not edit canonical memory documents for intake.
    Do not run `./scripts/code/agent/assist.sh`, `./scripts/code/prompt/compile.sh`, or `./scripts/code/prompt/update.sh`
    unless the user explicitly asks for consolidation or deployment.
    ```

---

### ディレクトリレイアウト (Git 管理)

#### [MODIFY] [project.yaml](file://prompts/manifest/project.yaml)
*   **Description**: `memory_docs` のグロブパターンを確認。現在 `prompts/memory/**/*.md` で `var/` 配下も拾ってしまうため、コード側 (frontmatter.go) でフィルタする方針とし、project.yaml 自体は変更しない。
*   **注意**: project.yaml のパターンは変更せず、コード側で除外する理由: グロブパターンで `!var/` のような除外構文は `doublestar` ライブラリでは直接サポートされないため、確実なコード側フィルタを優先する。

## Step-by-Step Implementation Guide

### Step 1: Compiler Ignore Hardening (TDD: Red -> Green)

1.  `features/tt/internal/prompt/memory/frontmatter_test.go` にテストケースを追加:
    *   `var/` 配下のファイルがスキップされること
    *   `schemas/` 配下のファイルがスキップされること
    *   通常の `*.md` はスキップされないこと
2.  `features/tt/internal/prompt/memory/frontmatter.go` に `shouldSkipMemoryPath` を実装し、`ParseAllMemoryDocs` に組み込む
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 2: 補助コマンドのロジック (TDD: Red -> Green)

1.  `features/tt/internal/agent/status/status_test.go` を作成:
    *   pending/processed/failed/ignored ファイルを tmpdir に配置し、カウントが正しいこと
    *   空ディレクトリでカウント 0 であること
2.  `features/tt/internal/agent/status/status.go` を実装
3.  `features/tt/internal/agent/status/list_test.go` を作成:
    *   SQLite にテストデータを投入し、フィルタが正しく動作すること
    *   FTS 検索が動作すること
4.  `features/tt/internal/agent/status/list.go` を実装
5.  `features/tt/internal/agent/status/show_test.go` を作成
6.  `features/tt/internal/agent/status/show.go` を実装
7.  テスト実行、Green を確認
8.  `git add && git commit`

### Step 3: コマンド層 (agent_status.go, agent_intake.go)

1.  `features/tt/cmd/agent_status.go` を作成
2.  `features/tt/cmd/agent_intake.go` を作成
3.  `git add && git commit`

### Step 4: Wrapper スクリプトの作成

1.  `scripts/code/` ディレクトリ構造を作成
2.  `scripts/code/_resolve_tool.sh` を作成 (graceful skip 対応)
3.  `scripts/code/agent/notify.sh` を作成 (明示的引数マッピング)
4.  `scripts/code/agent/status.sh` を作成
5.  `scripts/code/agent/intake.sh` を作成
6.  `scripts/code/prompt/compile.sh`, `deploy.sh`, `update.sh` を移行作成
7.  全 wrapper に `chmod +x` を設定
8.  `git add && git commit`

### Step 5: 常駐ポリシーの作成

1.  `prompts/manifest/code_content/policies/architecture-memory.md` を新規作成
2.  `./bin/tt.exe prompt compile` を実行して問題がないことを確認
3.  `git add && git commit`

### Step 6: ビルドと検証

1.  Verification Plan を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests (agent 関連)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "common" --specify "AgentNotify|AgentIntake|AgentStatus"
    ```

3.  **Compiler Ignore 検証**:
    ```bash
    # prompts/memory/var/ にダミーファイルを配置した状態で compile を実行し、エラーにならないことを確認
    mkdir -p prompts/memory/var/intake/pending/2026/06/07
    echo '{}' > prompts/memory/var/intake/pending/2026/06/07/E-test.json
    ./bin/tt prompt compile
    rm -rf prompts/memory/var/intake/pending/2026/06/07/E-test.json
    ```

4.  **Wrapper 動作検証**:
    ```bash
    # notify.sh の引数パースとマッピングを確認
    ./scripts/code/agent/notify.sh --agent antigravity --summary "Test" --note "Note1" --dry-run --print-payload
    ```

### テスト項目設計のセルフレビュー

#### 網羅性の検証

*   **Compiler Ignore**: `var/` と `schemas/` のスキップ + 通常ファイルの通過 -> Yes
*   **Status**: ファイルカウント + 空ディレクトリ -> Yes
*   **List**: フィルタ各種 + FTS -> Yes
*   **Show**: 正常取得 + 不在エラー -> Yes
*   **Wrapper**: graceful skip + 引数マッピング -> Yes

#### ボトムアップ順序

テスト順序: frontmatter (Compiler Ignore) -> status/list/show -> cmd -> wrapper

#### 観点チェックリスト

| # | 観点 | カバー状況 |
|---|------|-----------|
| 1 | 正常系の動作確認 | status: カウント一致, list: フィルタ結果, show: 完全 event 取得 |
| 2 | 異常系・境界値 | show: 存在しない event_id, list: 空結果 |
| 3 | 外部連携の実動作 | SQLite index からの読み取り |
| 4 | データの一貫性 | notify -> list -> show の一貫性 |
| 5 | 状態遷移の検証 | Part 1 の notify で pending が増え、status で反映 |
| 6 | 設定・構成の反映 | Compiler Ignore: var/ と schemas/ が除外される |
| 7 | 副作用の確認 | Compiler: var/ のファイルがコンパイル出力に混入しない |

## 継続計画について

本計画 (Part 1 + Part 2) で仕様書の全要件 (R1-R12) をカバーしている。

## Documentation

#### [MODIFY] [project.yaml](file://prompts/manifest/project.yaml)
*   **更新内容**: 変更なし (コード側でフィルタする方針)

#### [NEW] [architecture-memory.md](file://prompts/manifest/code_content/policies/architecture-memory.md)
*   **更新内容**: 常駐ポリシーの新規作成 (本計画の一部)
