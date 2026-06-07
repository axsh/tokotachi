# 001-Intake-Quality-Improvements

> **Source Specification**: prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/001-Intake-Quality-Improvements.md

## Goal Description

Agentic Memory Intake パイプラインの品質改善。5 つの要件 (R1-R5) を実装する。
- R1: `branch_package` を文字列からオブジェクトに拡張し、パス安全な id を分離
- R2: 常駐ポリシーに `raw_notes` の 1 項目 1 命題ガイダンスを追加
- R3: `created_at` の表示粒度を仕様として文書化 (コード変更なし)
- R4: `tt agent intake show --redact` で provenance を秘匿化
- R5: `tt agent status` の出力を拡充

## User Review Required

- R1 は `IntakeEvent.BranchPackage` の型を `string` -> `*BranchPackageInfo` に変更する破壊的変更。既存のテストデータ (1 件) は削除する。
- SQLite index の `branch_package` カラムは `key` 値を格納する形に変更するが、テーブル定義 (カラム名) は既存のまま維持する。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: branch_package のパス安全化 | Step 1: types.go, branch.go, supplement.go, index.go |
| R2: raw_notes のベストプラクティス | Step 2: architecture-memory.md |
| R3: created_at の表示粒度統一 | 仕様文書化のみ。コード変更なし |
| R4: --redact オプション | Step 3: show.go, agent_intake.go |
| R5: status 出力拡充 | Step 4: status.go, agent_status.go |

## Proposed Changes

### 1. Types & Branch Package

#### [MODIFY] [types.go](file:///features/tt/internal/agent/types.go)
*   **Description**: `BranchPackage` を `string` から `*BranchPackageInfo` に変更
*   **Technical Design**:
    ```go
    // BranchPackageInfo holds the structured branch package identifier.
    type BranchPackageInfo struct {
        Key       string `json:"key"`        // "owner/repo:branch:merge_base"
        ID        string `json:"id"`         // "BR-{branch_slug}-{merge_base_short8}"
        Branch    string `json:"branch"`     // raw branch name
        MergeBase string `json:"merge_base"` // full merge_base hash
    }
    ```
    `IntakeEvent.BranchPackage` フィールドを変更:
    ```go
    // Before:
    BranchPackage string `json:"branch_package,omitempty"`
    // After:
    BranchPackage *BranchPackageInfo `json:"branch_package,omitempty"`
    ```

#### [MODIFY] [branch.go](file:///features/tt/internal/agent/notify/branch.go)
*   **Description**: `DeriveBranchPackage` の戻り値を `*agent.BranchPackageInfo` に変更。slug 化関数を追加。
*   **Technical Design**:
    ```go
    // DeriveBranchPackage computes the structured branch package identifier.
    func DeriveBranchPackage(git *agent.GitInfo, executor GitExecutor) *agent.BranchPackageInfo {
        if git == nil || git.Branch == "" || git.Branch == "HEAD" {
            return nil
        }
        repoID := deriveRepoID(executor)
        mergeBase := git.MergeBase
        key := fmt.Sprintf("%s:%s:%s", repoID, git.Branch, mergeBase)
        slug := Slugify(git.Branch)
        short := mergeBase
        if len(short) > 8 {
            short = short[:8]
        }
        id := fmt.Sprintf("BR-%s-%s", slug, short)
        return &agent.BranchPackageInfo{
            Key:       key,
            ID:        id,
            Branch:    git.Branch,
            MergeBase: mergeBase,
        }
    }
    ```
*   **Logic** (`Slugify` 関数):
    ```go
    // Slugify converts a branch name to a path-safe slug.
    // Rules:
    //   - Replace characters outside [A-Za-z0-9._-] with "-"
    //   - Collapse consecutive "-" to one
    //   - Trim leading/trailing "-"
    //   - Max length 64 chars; if exceeded, use first 56 + "-" + 7-char hash
    func Slugify(name string) string
    ```
    1. `regexp.MustCompile("[^A-Za-z0-9._-]")` で不正文字を `-` に置換
    2. `regexp.MustCompile("-{2,}")` で連続 `-` を圧縮
    3. `strings.Trim(result, "-")` で先頭末尾の `-` を削除
    4. `len(result) > 64` の場合: `result[:56] + "-" + sha256(name)[:7]`

#### [MODIFY] [supplement.go](file:///features/tt/internal/agent/notify/supplement.go)
*   **Description**: L26 の `DeriveBranchPackage` 呼び出しの代入先を調整
*   **Logic**: 戻り値が `*BranchPackageInfo` に変わるため、代入がそのまま動作する (型変更のみ)

#### [MODIFY] [index.go](file:///features/tt/internal/agent/storage/index.go)
*   **Description**: `Store` で `event.BranchPackage.Key` を格納するように変更
*   **Logic**:
    ```go
    // L121 の変更:
    // Before:
    event.BranchPackage,
    // After:
    branchPackageKey(event.BranchPackage),
    ```
    ヘルパー関数:
    ```go
    func branchPackageKey(bp *agent.BranchPackageInfo) string {
        if bp == nil {
            return ""
        }
        return bp.Key
    }
    ```

### 2. Architecture Memory Policy (R2)

#### [MODIFY] [architecture-memory.md](file:///prompts/manifest/code_content/policies/architecture-memory.md)
*   **Description**: raw_notes の 1 項目 1 命題ガイダンスを追加
*   **追加テキスト** (ファイル末尾に追記):
    ```
    When writing --note values, keep each note as a single proposition or fact.
    Do not pack multiple concepts into one note.
    Bad:  --note "Pipeline: A -> B -> C -> D"
    Good: --note "A validates input before B" --note "B normalizes text" --note "C stores to disk"
    ```

### 3. Show --redact (R4)

#### [MODIFY] [show.go](file:///features/tt/internal/agent/status/show.go)
*   **Description**: `RedactProvenance` 関数を追加
*   **Technical Design**:
    ```go
    const redactedValue = "<redacted>"

    // RedactProvenance replaces provenance fields with <redacted>.
    // Does NOT modify the original event. Returns a copy.
    func RedactProvenance(event *agent.IntakeEvent) *agent.IntakeEvent {
        copy := *event
        copy.Provenance = agent.Provenance{
            Hostname: redactedValue,
            User:     redactedValue,
            Cwd:      redactedValue,
        }
        return &copy
    }
    ```
*   **Logic**: ポインタの中身をコピーして provenance だけ差し替え。元のイベント構造体は変更しない。

#### [MODIFY] [agent_intake.go](file:///features/tt/cmd/agent_intake.go)
*   **Description**: `--redact` フラグの追加
*   **Technical Design**:
    ```go
    var intakeShowRedact bool

    // init() に追加:
    agentIntakeShowCmd.Flags().BoolVar(&intakeShowRedact, "redact", false, "Redact provenance fields")
    ```
    `runAgentIntakeShow` で redact を適用:
    ```go
    func runAgentIntakeShow(cmd *cobra.Command, args []string) error {
        // ...existing code...
        event, err := status.Show(varDir, args[0])
        // ...error handling...
        if intakeShowRedact {
            event = status.RedactProvenance(event)
        }
        // ...marshal and print...
    }
    ```

### 4. Status 出力拡充 (R5)

#### [MODIFY] [status.go](file:///features/tt/internal/agent/status/status.go)
*   **Description**: `StatusReport` 構造体と `GetStatus` を全面改修
*   **Technical Design**:
    ```go
    // StatusCounts holds event counts by status.
    type StatusCounts struct {
        Pending   int `json:"pending"`
        Processed int `json:"processed"`
        Failed    int `json:"failed"`
        Ignored   int `json:"ignored"`
    }

    // StatusReport holds the status summary.
    type StatusReport struct {
        MemoryRoot          string        `json:"memory_root"`
        CurrentBranch       string        `json:"current_branch"`
        Counts              StatusCounts  `json:"counts"`
        CurrentBranchCounts *StatusCounts `json:"current_branch_counts,omitempty"`
        OldestPending       string        `json:"oldest_pending,omitempty"`
        IndexHealth         string        `json:"index_health"`
    }
    ```
*   **Logic** (`GetStatus` 引数追加):
    ```go
    // GetStatus computes the status report.
    // memoryRoot: "prompts/memory" (for display)
    // varDir:     "prompts/memory/var" (for file counting)
    // currentBranch: from git (caller provides)
    func GetStatus(memoryRoot, varDir, currentBranch string) (*StatusReport, error)
    ```
    1. `Counts`: 全ブランチの合計 (既存ロジック流用)
    2. `CurrentBranchCounts`: SQLite index から `WHERE branch = currentBranch` で集計。index が存在しない場合は `nil`。
    3. `OldestPending`: 最古 pending イベントの `created_at` を ISO8601 秒精度 (`2006-01-02T15:04:05Z`) で返す。人間可読の age 形式は廃止。
    4. `IndexHealth`:
        - `os.Stat(dbPath)` が `IsNotExist` -> `"missing"`
        - `sql.Open` + `PRAGMA integrity_check` が失敗 -> `"error"`
        - 正常 -> `"ok"`

#### [MODIFY] [agent_status.go](file:///features/tt/cmd/agent_status.go)
*   **Description**: `runAgentStatus` で `currentBranch` を取得して `GetStatus` に渡す
*   **Logic**:
    ```go
    func runAgentStatus(cmd *cobra.Command, args []string) error {
        memoryRoot := filepath.Join("prompts", "memory")
        varDir := filepath.Join(memoryRoot, "var")
        branch := getCurrentBranch()

        report, err := status.GetStatus(memoryRoot, varDir, branch)
        // ...
    }

    func getCurrentBranch() string {
        cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
        out, err := cmd.Output()
        if err != nil {
            return ""
        }
        return strings.TrimSpace(string(out))
    }
    ```

### 5. 旧データの削除

#### [DELETE] 旧テスト Intake Event
*   `prompts/memory/var/intake/pending/2026/06/07/E-01KTHKB1PPM6XRNB14NK48K2KX.json`
*   `prompts/memory/var/intake/index.db`, `index.db-wal`, `index.db-shm`
*   `prompts/memory/var/logs/agent-notify.ndjson`

## Step-by-Step Implementation Guide

### Step 1: R1 - BranchPackageInfo (TDD) [x]

1. [x] `types.go` に `BranchPackageInfo` 構造体を追加。`IntakeEvent.BranchPackage` の型を `*BranchPackageInfo` に変更。
2. [x] `branch.go` に `Slugify` 関数を追加。
3. [x] `branch_test.go` にテストを追加:
    - `TestSlugify`: 基本変換、連続ダッシュ圧縮、先頭末尾トリム、64 文字制限
    - `TestDeriveBranchPackage`: 戻り値が `*BranchPackageInfo` であること、ID にパス不安全文字がないこと
4. [x] `branch.go` の `DeriveBranchPackage` を `*BranchPackageInfo` を返すように変更。
5. [x] `supplement.go` L26 の代入を確認 (型が合うのでそのまま)。
6. [x] `index.go` に `branchPackageKey` ヘルパーを追加。`Store` の L121 を変更。
7. [x] 既存テスト (`supplement_test.go`) のコンパイルエラーを修正。
8. [x] テスト実行、Green を確認。
9. [x] `git add && git commit`

### Step 2: R2 - ポリシー更新 [x]

1. [x] `prompts/manifest/code_content/policies/architecture-memory.md` に raw_notes ガイダンスを追記。
2. [x] `git add && git commit`

### Step 3: R4 - --redact [x]

1. [x] `show_test.go` にテストを追加:
    - `TestRedactProvenance`: provenance フィールドが `<redacted>` になること、元イベント不変、他フィールド保持
2. [x] `show.go` に `RedactProvenance` 関数を追加。
3. [x] `agent_intake.go` に `--redact` フラグを追加。`runAgentIntakeShow` で適用。
4. [x] テスト実行、Green を確認。
5. [x] `git add && git commit`

### Step 4: R5 - status 出力拡充 [x]

1. [x] `status_test.go` のテストを更新:
    - `TestGetStatus_NewFormat`: 新しい構造 (`MemoryRoot`, `CurrentBranch`, `Counts`, `CurrentBranchCounts`) が返ること
    - `TestGetStatus_IndexMissing`: index なしの場合 `IndexHealth` が `missing`
    - `TestGetStatus_IndexOk`: index ありの場合 `IndexHealth` が `ok`
    - `TestGetStatus_OldestPendingISO`: `OldestPending` が ISO8601 秒精度であること
    - `TestGetStatus_BranchCounts`: ブランチ別カウントが正しいこと
2. [x] `status.go` を全面改修: `StatusReport` 構造体変更、`GetStatus` 引数追加。
3. [x] `agent_status.go` を更新: `getCurrentBranch` 追加、`GetStatus` 呼び出し変更。
4. [x] テスト実行、Green を確認。
5. [x] `git add && git commit`

### Step 5: 旧データ削除 & クリーンアップ [x]

1. [x] 旧テスト Intake Event、index.db、audit log を削除。
2. [x] `.gitignore` に `prompts/memory/var/` が含まれていることを確認 (既に含まれていた)。

### Step 6: ビルドと検証 [x]

1. [x] `./scripts/process/build.sh --backend-only` PASSED (21s)
2. [x] 動作確認: notify, show, show --redact, status 全て期待通りの出力

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --backend-only
    ```

2.  **Prompt Compile** (R2 のポリシー変更確認):
    ```bash
    ./bin/tt.exe prompt compile --apply
    ```

3.  **手動動作確認** (自動テスト補完):
    R1 の動作確認として、ビルド後に以下を実行し出力形式を確認:
    ```bash
    ./bin/tt.exe agent notify --agent antigravity --summary "R1 test" --note "single proposition test"
    ./bin/tt.exe agent status
    ./bin/tt.exe agent intake list
    ./bin/tt.exe agent intake show <event-id>
    ./bin/tt.exe agent intake show <event-id> --redact
    ```

### テスト項目セルフレビュー

**1. 網羅性**: R1 (slug化 + BranchPackageInfo 構造体) + R4 (redact) + R5 (status 拡充) の全要件が単体テストでカバーされている。R2 はコード変更なし、R3 は仕様文書化のみ。

**2. 証拠の十分性**: 各テストは「期待する値が返る」「期待する構造である」を検証している。

**3. 迂回排除**: `TestSlugify` でパス不安全文字が残らないことを正規表現で検証。`TestRedactProvenance` で元イベントが変更されないことを検証。

**4. 依存関係**: `Slugify` -> `DeriveBranchPackage` -> `SupplementEnvironment` -> `HandleNotify` の順でボトムアップにテスト。

## Documentation

#### [MODIFY] [001-Intake-Quality-Improvements.md](file:///prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/001-Intake-Quality-Improvements.md)
*   **更新内容**: 実装完了後、各 R の完了状態を更新
