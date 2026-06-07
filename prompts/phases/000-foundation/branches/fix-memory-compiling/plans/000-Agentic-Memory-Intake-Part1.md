# 000-Agentic-Memory-Intake-Part1

> **Source Specification**: [000-Agentic-Memory-Intake.md](../ideas/000-Agentic-Memory-Intake.md)

## Goal Description

`tt agent notify` コマンドと、その内部ロジック (バリデーション、正規化、ID/Hash 計算、atomic file write、SQLite index、監査ログ) を実装する。本 Part では以下を対象とする:

1. 共通型定義 (`IntakeEvent`, `NotifyResult` 等)
2. JSON Schema ファイル (3 種)
3. 内部ロジック層 (`features/tt/internal/agent/notify/`, `features/tt/internal/agent/storage/`)
4. コマンド層 (`features/tt/cmd/agent.go`, `features/tt/cmd/agent_notify.go`)
5. 新規依存パッケージ (`oklog/ulid/v2`, `modernc.org/sqlite`)

Part 2 では補助コマンド (status/intake list/show)、Wrapper スクリプト、Compiler Ignore、プロンプト新規作成を扱う。

## User Review Required

> [!IMPORTANT]
> **SQLite ライブラリの選択**: 仕様書では `modernc.org/sqlite` (pure Go) と `github.com/mattn/go-sqlite3` (CGo) の 2 候補が挙げられている。本計画では **`modernc.org/sqlite`** を採用する。理由: CGo は Windows/macOS/Linux 間のクロスコンパイルに gcc が必要でビルドが複雑になるため。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `tt agent notify` コマンドの実装 (JSON/CLI 2 経路, フラグ一覧) | Proposed Changes > cmd/agent.go, cmd/agent_notify.go |
| R2: 入力スキーマ (`agent-notify-payload.schema.json`) | Proposed Changes > schemas/ |
| R3: 保存スキーマ (`intake-event.schema.json`) | Proposed Changes > schemas/ |
| R4: ディレクトリレイアウトと Git 管理ポリシー | Proposed Changes > .gitignore, Step-by-Step > Step 1 |
| R5: Wrapper スクリプト | **Part 2 で対応** |
| R6: 補助コマンド | **Part 2 で対応** |
| R7: 結果スキーマとエラーハンドリング | Proposed Changes > schemas/, types.go, handler.go |
| R8: ID と Hash の戦略 | Proposed Changes > identity.go |
| R9: 内部処理パイプライン | Proposed Changes > handler.go |
| R10: Compiler Ignore Hardening | **Part 2 で対応** |
| R11: Branch Strategy | Proposed Changes > branch.go |
| R12: Metrics / Observability (MAY) | types.go のデータモデルにフィールドを含める |

## Proposed Changes

### JSON Schema (`prompts/memory/schemas/`)

#### [NEW] [agent-notify-payload.schema.json](file://prompts/memory/schemas/agent-notify-payload.schema.json)
*   **Description**: Coding Agent から送信されるペイロードの入力バリデーション用スキーマ
*   **Technical Design**:
    ```json
    {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "$id": "agent-notify-payload.schema.json",
      "type": "object",
      "required": ["version", "source_type", "agent", "task_summary", "raw_notes"],
      "properties": {
        "version": { "type": "integer", "const": 1 },
        "source_type": { "type": "string", "enum": ["coding_agent"] },
        "agent": {
          "type": "string",
          "enum": ["codex", "claude-code", "antigravity", "cursor", "unknown"]
        },
        "task_summary": { "type": "string", "minLength": 1, "maxLength": 500 },
        "raw_notes": {
          "type": "array", "minItems": 1, "maxItems": 32,
          "items": { "type": "string", "minLength": 1, "maxLength": 500 }
        },
        "changed_paths": { "type": "array", "items": { "type": "string" } },
        "flags": {
          "type": "object",
          "properties": {
            "architecture_impact": { "type": "boolean" },
            "memory_related": { "type": "boolean" },
            "prompt_related": { "type": "boolean" },
            "agent_behavior_related": { "type": "boolean" },
            "requires_immediate_action": { "type": "boolean" }
          },
          "additionalProperties": false
        },
        "client_request_id": { "type": "string" },
        "context": {
          "type": "object",
          "properties": {
            "session_id": { "type": "string" },
            "task_id": { "type": "string" },
            "wrapper_version": { "type": "string" }
          }
        },
        "extra": { "type": "object" }
      },
      "additionalProperties": false
    }
    ```

#### [NEW] [intake-event.schema.json](file://prompts/memory/schemas/intake-event.schema.json)
*   **Description**: `tt` が自動補完した完全な IntakeEvent の保存スキーマ
*   **Technical Design**: 入力ペイロードの全フィールドに加えて以下を追加:
    *   `event_id` (string, pattern: `^E-[0-9A-Z]{26}$`)
    *   `instance_id` (string)
    *   `content_hash` (string, pattern: `^sha256:[0-9a-f]{64}$`)
    *   `content_id` (string, pattern: `^RAWC-[0-9a-f]{64}$`)
    *   `git` (object: `branch`, `head_commit`, `is_dirty`, `merge_base`, `default_branch`)
    *   `scope` (string, enum: `["branch", "session"]`)
    *   `branch_package` (string)
    *   `effective_changed_paths` (array of string)
    *   `timestamps` (object: `created_at`, `stored_at`)
    *   `provenance` (object: `hostname`, `user`, `cwd`, `wrapper_version`)

#### [NEW] [agent-notify-result.schema.json](file://prompts/memory/schemas/agent-notify-result.schema.json)
*   **Description**: `tt agent notify` の結果スキーマ
*   **Technical Design**:
    ```json
    {
      "type": "object",
      "required": ["status", "code"],
      "properties": {
        "status": { "type": "string", "enum": ["accepted", "accepted_with_warnings", "rejected"] },
        "code": { "type": "string" },
        "mode": { "type": "string", "enum": ["deferred"] },
        "event_id": { "type": "string" },
        "instance_id": { "type": "string" },
        "content_hash": { "type": "string" },
        "content_id": { "type": "string" },
        "stored_at": { "type": "string" },
        "index_state": { "type": "string", "enum": ["ok", "degraded", "unavailable"] },
        "warnings": { "type": "array", "items": { "type": "string" } },
        "next_action": { "type": "string" },
        "message": { "type": "string" }
      }
    }
    ```

---

### 共通型定義 (`features/tt/internal/agent/`)

#### [NEW] [types.go](file://features/tt/internal/agent/types.go)
*   **Description**: `IntakeEvent`, `NotifyResult`, `NotifyPayload`, `GitInfo`, `Provenance`, `Timestamps`, `Flags`, `Context` 等の構造体定義
*   **Technical Design**:
    ```go
    package agent

    import "time"

    // NotifyPayload represents the input from coding agent
    type NotifyPayload struct {
        Version        int               `json:"version"`
        SourceType     string            `json:"source_type"`
        Agent          string            `json:"agent"`
        TaskSummary    string            `json:"task_summary"`
        RawNotes       []string          `json:"raw_notes"`
        ChangedPaths   []string          `json:"changed_paths,omitempty"`
        Flags          *Flags            `json:"flags,omitempty"`
        ClientRequestID string           `json:"client_request_id,omitempty"`
        Context        *PayloadContext   `json:"context,omitempty"`
        Extra          map[string]any    `json:"extra,omitempty"`
    }

    // Flags represents boolean flags for categorization
    type Flags struct {
        ArchitectureImpact     bool `json:"architecture_impact,omitempty"`
        MemoryRelated          bool `json:"memory_related,omitempty"`
        PromptRelated          bool `json:"prompt_related,omitempty"`
        AgentBehaviorRelated   bool `json:"agent_behavior_related,omitempty"`
        RequiresImmediateAction bool `json:"requires_immediate_action,omitempty"`
    }

    // PayloadContext holds session/task metadata
    type PayloadContext struct {
        SessionID      string `json:"session_id,omitempty"`
        TaskID         string `json:"task_id,omitempty"`
        WrapperVersion string `json:"wrapper_version,omitempty"`
    }

    // IntakeEvent is the full stored event (payload + computed fields)
    type IntakeEvent struct {
        NotifyPayload
        EventID              string       `json:"event_id"`
        InstanceID           string       `json:"instance_id"`
        ContentHash          string       `json:"content_hash"`
        ContentID            string       `json:"content_id"`
        Git                  *GitInfo     `json:"git,omitempty"`
        Scope                string       `json:"scope"`
        BranchPackage        string       `json:"branch_package,omitempty"`
        EffectiveChangedPaths []string    `json:"effective_changed_paths,omitempty"`
        Timestamps           Timestamps   `json:"timestamps"`
        Provenance           Provenance   `json:"provenance"`
    }

    // GitInfo holds git repository state
    type GitInfo struct {
        Branch        string `json:"branch"`
        HeadCommit    string `json:"head_commit"`
        IsDirty       bool   `json:"is_dirty"`
        MergeBase     string `json:"merge_base,omitempty"`
        DefaultBranch string `json:"default_branch,omitempty"`
    }

    // Timestamps holds creation and storage times
    type Timestamps struct {
        CreatedAt time.Time `json:"created_at"`
        StoredAt  time.Time `json:"stored_at"`
    }

    // Provenance holds environment info
    type Provenance struct {
        Hostname       string `json:"hostname"`
        User           string `json:"user"`
        Cwd            string `json:"cwd"`
        WrapperVersion string `json:"wrapper_version,omitempty"`
    }

    // NotifyResult is the command output
    type NotifyResult struct {
        Status     string   `json:"status"`
        Code       string   `json:"code"`
        Mode       string   `json:"mode,omitempty"`
        EventID    string   `json:"event_id,omitempty"`
        InstanceID string   `json:"instance_id,omitempty"`
        ContentHash string  `json:"content_hash,omitempty"`
        ContentID  string   `json:"content_id,omitempty"`
        StoredAt   string   `json:"stored_at,omitempty"`
        IndexState string   `json:"index_state,omitempty"`
        Warnings   []string `json:"warnings,omitempty"`
        NextAction string   `json:"next_action,omitempty"`
        Message    string   `json:"message,omitempty"`
    }

    // Exit code constants
    const (
        ExitOK                     = 0
        ExitJSONParseError         = 10
        ExitSchemaValidationError  = 11
        ExitUnsupportedVersion     = 12
        ExitAgentIDInvalid         = 20
        ExitStorageLockTimeout     = 30
        ExitStorageWriteFailed     = 31
        ExitPermissionDenied       = 40
    )

    // Result code constants
    const (
        CodeOK                    = "OK"
        CodeIndexDegraded         = "INDEX_DEGRADED"
        CodeNoGitRepository       = "NO_GIT_REPOSITORY"
        CodeJSONParseError        = "JSON_PARSE_ERROR"
        CodeSchemaValidationError = "SCHEMA_VALIDATION_ERROR"
        CodeUnsupportedVersion    = "UNSUPPORTED_SCHEMA_VERSION"
        CodeAgentIDInvalid        = "AGENT_ID_INVALID"
        CodeStorageLockTimeout    = "STORAGE_LOCK_TIMEOUT"
        CodeStorageWriteFailed    = "STORAGE_WRITE_FAILED"
        CodePermissionDenied      = "PERMISSION_DENIED"
    )
    ```

---

### 内部ロジック層 (`features/tt/internal/agent/notify/`)

#### [NEW] [schema.go](file://features/tt/internal/agent/notify/schema.go)
*   **Description**: JSON Schema によるペイロードバリデーション
*   **Technical Design**:
    ```go
    package notify

    // Validator validates a NotifyPayload against the JSON schema.
    type Validator struct {
        schemaPath string // path to agent-notify-payload.schema.json
    }

    // NewValidator creates a new Validator.
    // schemaPath is the absolute path to the schemas directory.
    func NewValidator(schemaPath string) (*Validator, error)

    // Validate validates the raw JSON bytes against the schema.
    // Returns nil if valid, or an error with details of violations.
    func (v *Validator) Validate(data []byte) error
    ```
*   **Logic**:
    *   `github.com/santhosh-tekuri/jsonschema/v6` (既に go.mod に存在) を使用
    *   `agent-notify-payload.schema.json` をロードし、入力 JSON をバリデーション
    *   version フィールドが 1 以外の場合は `UNSUPPORTED_SCHEMA_VERSION` エラー
    *   agent フィールドが enum 外の場合は `AGENT_ID_INVALID` エラー

#### [NEW] [normalize.go](file://features/tt/internal/agent/notify/normalize.go)
*   **Description**: テキスト正規化処理
*   **Technical Design**:
    ```go
    package notify

    // NormalizePayload normalizes text fields in-place:
    // 1. NFC normalization for task_summary and each raw_note
    // 2. Trim leading/trailing whitespace
    // 3. Compress consecutive whitespace to single space
    // 4. Remove empty notes from raw_notes after normalization
    func NormalizePayload(p *agent.NotifyPayload) error
    ```
*   **Logic**:
    *   `golang.org/x/text/unicode/norm` パッケージで NFC 正規化 (`norm.NFC.String(s)`)
    *   `strings.TrimSpace()` で前後の空白を除去
    *   正規表現 `regexp.MustCompile(`\s+`)` で連続空白を圧縮
    *   空文字列になった note はスライスから除外
    *   除外後に `raw_notes` が 0 件なら `SCHEMA_VALIDATION_ERROR` を返す

#### [NEW] [supplement.go](file://features/tt/internal/agent/notify/supplement.go)
*   **Description**: Git 情報・環境情報の自動補完
*   **Technical Design**:
    ```go
    package notify

    // SupplementEnvironment fills in Git info, provenance, and effective paths.
    // If not in a git repository, sets scope to "session" and adds NO_GIT_REPOSITORY warning.
    func SupplementEnvironment(event *agent.IntakeEvent, collectGitPaths bool) (warnings []string)
    ```
*   **Logic**:
    *   `git rev-parse --show-toplevel` でリポジトリルートを検出。失敗時は `scope="session"`, warning `NO_GIT_REPOSITORY`
    *   `git rev-parse --abbrev-ref HEAD` で branch を取得。`HEAD` (detached) の場合は `scope="session"`
    *   `git rev-parse HEAD` で head_commit を取得
    *   `git diff --name-only HEAD` + `git diff --name-only --cached HEAD` + `git ls-files --others --exclude-standard` で dirty paths を取得 (is_dirty フラグも設定)
    *   `--changed-paths-from-git` が有効な場合、dirty paths と `changed_paths` の union を `effective_changed_paths` に設定
    *   `os.Hostname()`, `os.UserHomeDir()` or `user.Current()`, `os.Getwd()` で provenance を補完

#### [NEW] [identity.go](file://features/tt/internal/agent/notify/identity.go)
*   **Description**: event_id (ULID), content_hash (SHA-256), content_id (RAWC-) の計算
*   **Technical Design**:
    ```go
    package notify

    // GenerateEventID generates a new ULID-based event ID with "E-" prefix.
    func GenerateEventID() (string, error)

    // ComputeContentHash computes SHA-256 of the canonical JSON representation.
    // Input fields: effective_changed_paths + flags + task_summary + raw_notes (normalized, sorted).
    func ComputeContentHash(event *agent.IntakeEvent) string

    // ComputeContentID computes the coarse fingerprint for semantic grouping.
    // Input fields: task_summary + raw_notes + normalized path prefix.
    // Excludes: branch, timestamps, wrapper_version.
    // Returns "RAWC-" + hex(sha256).
    func ComputeContentID(event *agent.IntakeEvent) string
    ```
*   **Logic**:
    *   `event_id`: `github.com/oklog/ulid/v2` を使用。`ulid.New(ulid.Timestamp(time.Now()), cryptoRandReader)` で生成し `"E-" + ulid.String()` で prefix
    *   `content_hash`: 正規化後の JSON バイト列 (`effective_changed_paths` + `flags` + `task_summary` + `raw_notes` をソート・結合) の SHA-256。`"sha256:" + hex.EncodeToString(hash)`
    *   `content_id`: `task_summary` + `raw_notes` (ソート) + path prefix (各 path のディレクトリ部分だけ取得し、重複除去・ソート) の SHA-256。`"RAWC-" + hex.EncodeToString(hash)`

#### [NEW] [branch.go](file://features/tt/internal/agent/notify/branch.go)
*   **Description**: branch_package の導出とスコープ管理
*   **Technical Design**:
    ```go
    package notify

    // DeriveBranchPackage computes the branch_package identifier.
    // Format: "{repo_id}:{branch_name}:{merge_base}"
    // repo_id is derived from the git remote origin URL.
    // merge_base is computed as: git merge-base HEAD {default_branch}
    func DeriveBranchPackage(git *agent.GitInfo) string

    // DeriveScope determines scope based on git availability.
    // Returns "branch" if in a git repo with a named branch.
    // Returns "session" if no git, or detached HEAD.
    func DeriveScope(git *agent.GitInfo) string
    ```
*   **Logic**:
    *   `git remote get-url origin` でリモート URL を取得し、`owner/repo` 形式に正規化して `repo_id` とする
    *   `git merge-base HEAD main` (default_branch) で merge_base を取得。失敗時は空文字列
    *   `branch_package = fmt.Sprintf("%s:%s:%s", repoID, branchName, mergeBase)`
    *   detached HEAD (`branch == "HEAD"`) の場合は `scope="session"`

#### [NEW] [handler.go](file://features/tt/internal/agent/notify/handler.go)
*   **Description**: パイプライン全体のオーケストレーション
*   **Technical Design**:
    ```go
    package notify

    // Handler orchestrates the full notify pipeline.
    type Handler struct {
        validator  *Validator
        fileStore  storage.FileStore
        index      storage.Index
        auditLog   storage.AuditLog
    }

    // NewHandler creates a new Handler with the given storage backends.
    func NewHandler(schemasDir, varDir string) (*Handler, error)

    // HandleNotify processes a notify request end-to-end.
    // Steps: parse -> validate -> normalize -> supplement -> derive IDs -> store -> index -> audit -> return result
    func (h *Handler) HandleNotify(inputJSON []byte, collectGitPaths bool) (*agent.NotifyResult, int)
    ```
*   **Logic** (R9 パイプライン):
    1. `json.Unmarshal(inputJSON, &payload)` -- 失敗時は `JSON_PARSE_ERROR`, exit 10
    2. `validator.Validate(inputJSON)` -- 失敗時は `SCHEMA_VALIDATION_ERROR`, exit 11
    3. `NormalizePayload(&payload)` -- NFC, trim, drop empty
    4. `IntakeEvent` を構築し `SupplementEnvironment(&event, collectGitPaths)` -- Git 情報補完
    5. `DeriveBranchPackage`, `DeriveScope` -- branch_package, scope 設定
    6. `GenerateEventID()` -- event_id 生成。`instance_id = event_id` (Intake 段階)
    7. `ComputeContentHash`, `ComputeContentID` -- hash 計算
    8. `Timestamps{CreatedAt: now, StoredAt: now}` を設定
    9. `fileStore.Write(event)` -- atomic file write。失敗時は `STORAGE_WRITE_FAILED`, exit 31
    10. `index.Store(event)` -- SQLite index。失敗時は warning `INDEX_DEGRADED` (ファイル保存は成功)
    11. `auditLog.Append(event, result)` -- NDJSON 追記
    12. `client_request_id` の冪等性チェック: index に同一 `client_request_id` が存在する場合、既存の `event_id` を返す

---

### 保存層 (`features/tt/internal/agent/storage/`)

#### [NEW] [filestore.go](file://features/tt/internal/agent/storage/filestore.go)
*   **Description**: pending ディレクトリへの atomic file write
*   **Technical Design**:
    ```go
    package storage

    // FileStore handles atomic file writes to the pending directory.
    type FileStore struct {
        baseDir string // e.g. "prompts/memory/var/intake"
    }

    // NewFileStore creates a new FileStore.
    func NewFileStore(baseDir string) *FileStore

    // Write atomically writes an IntakeEvent to pending/{YYYY}/{MM}/{DD}/E-{ULID}.json.
    // Steps: marshal JSON -> write to _tmp/ -> fsync -> rename to pending/
    func (fs *FileStore) Write(event *agent.IntakeEvent) (storedAt string, err error)
    ```
*   **Logic**:
    *   `storedAt` のパス計算: `pending/{event.Timestamps.CreatedAt.Format("2006/01/02")}/E-{event.EventID[2:]}.json`
    *   `os.MkdirAll(tmpDir, 0755)` で `_tmp/` ディレクトリを作成
    *   `os.CreateTemp(tmpDir, "intake-*.json")` で一時ファイルを作成
    *   `json.MarshalIndent(event, "", "  ")` で JSON 化して書き込み
    *   `file.Sync()` で fsync
    *   `file.Close()`
    *   `os.MkdirAll(filepath.Dir(finalPath), 0755)` で最終パスのディレクトリを作成
    *   `os.Rename(tmpPath, finalPath)` で atomic rename

#### [NEW] [index.go](file://features/tt/internal/agent/storage/index.go)
*   **Description**: SQLite index の管理
*   **Technical Design**:
    ```go
    package storage

    // Index manages the SQLite index for intake events.
    type Index struct {
        db *sql.DB
    }

    // NewIndex opens or creates the SQLite database at the given path.
    // Enables WAL mode and creates tables if not exist.
    func NewIndex(dbPath string) (*Index, error)

    // Store inserts event metadata into the index.
    // If client_request_id already exists, returns the existing event_id (idempotency).
    func (idx *Index) Store(event *agent.IntakeEvent) (existingEventID string, err error)

    // Close closes the database connection.
    func (idx *Index) Close() error
    ```
*   **Logic**:
    *   テーブル: `intake_events` (event_id PK, content_hash, content_id, agent, branch, scope, branch_package, status DEFAULT 'pending', client_request_id UNIQUE, task_summary, stored_at, created_at)
    *   FTS テーブル: `intake_events_fts` (event_id, task_summary, raw_notes_text) -- `fts5` 使用
    *   `PRAGMA journal_mode=WAL;`
    *   `PRAGMA busy_timeout=5000;` -- lock timeout 5 秒
    *   冪等性: `INSERT INTO intake_events ... ON CONFLICT(client_request_id) DO NOTHING` + `SELECT event_id WHERE client_request_id = ?`

#### [NEW] [audit.go](file://features/tt/internal/agent/storage/audit.go)
*   **Description**: NDJSON 監査ログの追記
*   **Technical Design**:
    ```go
    package storage

    // AuditLog appends audit entries as NDJSON.
    type AuditLog struct {
        logPath      string // agent-notify.ndjson
        errorLogPath string // agent-notify-error.ndjson
    }

    // NewAuditLog creates a new AuditLog.
    func NewAuditLog(logDir string) *AuditLog

    // Append appends an audit entry to the appropriate log file.
    func (al *AuditLog) Append(event *agent.IntakeEvent, result *agent.NotifyResult) error
    ```
*   **Logic**:
    *   成功 (`status == "accepted"` or `"accepted_with_warnings"`) は `agent-notify.ndjson` に追記
    *   失敗 (`status == "rejected"`) は `agent-notify-error.ndjson` に追記
    *   各行は `{"timestamp": "...", "event_id": "...", "status": "...", "code": "...", ...}` の JSON

---

### コマンド層 (`features/tt/cmd/`)

#### [NEW] [agent.go](file://features/tt/cmd/agent.go)
*   **Description**: `tt agent` サブコマンドグループの定義
*   **Technical Design**:
    ```go
    package cmd

    import "github.com/spf13/cobra"

    var agentCmd = &cobra.Command{
        Use:   "agent",
        Short: "Manage agent memory and notifications",
    }

    func init() {
        rootCmd.AddCommand(agentCmd)
    }
    ```

#### [NEW] [agent_notify.go](file://features/tt/cmd/agent_notify.go)
*   **Description**: `tt agent notify` コマンド。JSON 経路と CLI フラグ経路の 2 入力を処理
*   **Technical Design**:
    ```go
    package cmd

    var agentNotifyCmd = &cobra.Command{
        Use:   "notify",
        Short: "Submit a memory intake notification",
        RunE:  runAgentNotify,
    }

    // CLI flags
    var (
        notifyFile         string
        notifyStdin        bool
        notifyAgent        string
        notifySummary      string
        notifySummaryFile  string
        notifyNotes        []string  // --note can be repeated
        notifyNotesFile    string
        notifyChangedPaths []string  // --changed-path can be repeated
        notifyFromGit      bool
        notifyFlagArch     bool
        notifyFlagMem      bool
        notifyFlagPrompt   bool
        notifyFlagAgent    bool
        notifyFlagUrgent   bool
        notifyClientReqID  string
        notifyDryRun       bool
        notifyPrintPayload bool
    )
    ```
*   **Logic**:
    *   `--file` / `--stdin` 指定時: ファイルまたは stdin から JSON を読み、`handler.HandleNotify(jsonBytes, notifyFromGit)` を呼ぶ
    *   CLI フラグ経路: `NotifyPayload` を構築し、`json.Marshal` して `handler.HandleNotify()` に渡す
        *   `--summary-file` 指定時はファイルから summary を読む
        *   `--notes-file` 指定時は 1 行 1 note として読む
        *   `--architecture-impact` 等のフラグは `Flags` 構造体にマッピング
    *   両方指定時はエラー (`"--file/--stdin and CLI flags are mutually exclusive"`)
    *   結果を JSON で stdout に出力
    *   exit code は `NotifyResult` の `Code` からマッピング

---

### ディレクトリレイアウト

#### [NEW] [.gitignore](file://prompts/memory/.gitignore)
*   **Description**: `prompts/memory/var/` を Git 管理対象外にする
*   **Technical Design**:
    ```
    var/
    ```

---

### 依存パッケージ

#### [MODIFY] [go.mod](file://features/tt/go.mod)
*   **Description**: 新規依存パッケージの追加
*   **Technical Design**:
    *   `github.com/oklog/ulid/v2` -- ULID 生成
    *   `modernc.org/sqlite` -- pure Go SQLite ドライバ
    *   `golang.org/x/text` -- NFC 正規化 (既に indirect で存在するが direct に昇格)

## Step-by-Step Implementation Guide

### Step 1: ディレクトリレイアウトとスキーマの準備

1.  `prompts/memory/.gitignore` を作成し `var/` を記載
2.  `prompts/memory/schemas/` ディレクトリを作成
3.  `agent-notify-payload.schema.json`, `intake-event.schema.json`, `agent-notify-result.schema.json` の 3 ファイルを作成
4.  `git add && git commit`

### Step 2: 依存パッケージの追加

1.  `features/tt/go.mod` に `github.com/oklog/ulid/v2`, `modernc.org/sqlite`, `golang.org/x/text` を追加
2.  `go mod tidy` を実行
3.  `git add && git commit`

### Step 3: 共通型定義 (TDD: Red -> Green)

1.  `features/tt/internal/agent/types.go` を作成 (上記の構造体定義)
2.  `features/tt/internal/agent/types_test.go` を作成 -- exit code 定数・code 定数が期待値と一致すること、JSON marshal/unmarshal のラウンドトリップを確認
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 4: スキーマバリデーション (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/schema_test.go` を作成 -- テーブル駆動テスト:
    *   有効なペイロード -> nil
    *   必須フィールド欠損 (`raw_notes` 欠落) -> error
    *   型不一致 (`version: "1"` instead of `1`) -> error
    *   境界値 (`raw_notes` 0 件, 33 件, `task_summary` 501 文字) -> error
    *   `agent` が enum 外 -> error
2.  `features/tt/internal/agent/notify/schema.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 5: テキスト正規化 (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/normalize_test.go` を作成 -- テーブル駆動テスト:
    *   NFC 正規化 (濁点分離 -> 合成済み)
    *   空白圧縮
    *   前後 trim
    *   空ノート除外 (正規化後に空になるケース)
    *   全ノートが空になる -> error
2.  `features/tt/internal/agent/notify/normalize.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 6: ID/Hash 計算 (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/identity_test.go` を作成 -- テーブル駆動テスト:
    *   ULID 一意性 (100 回生成して全て異なる)
    *   `content_hash` 同一性 (同じ入力 -> 同じ hash)
    *   `content_id` の branch/timestamp 非依存性 (異なる branch/timestamp でも同じ content -> 同じ content_id)
    *   `event_id` の "E-" prefix
    *   `content_hash` の "sha256:" prefix
    *   `content_id` の "RAWC-" prefix
2.  `features/tt/internal/agent/notify/identity.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 7: Branch Strategy (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/branch_test.go` を作成 -- テーブル駆動テスト:
    *   正常な Git 環境 -> `scope="branch"`, `branch_package` が `owner/repo:branch:mergebase` 形式
    *   Git なし -> `scope="session"`
    *   detached HEAD -> `scope="session"`
    *   merge_base 取得失敗 -> branch_package の末尾が空
2.  `features/tt/internal/agent/notify/branch.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 8: 環境情報補完 (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/supplement_test.go` を作成
2.  `features/tt/internal/agent/notify/supplement.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 9: Storage -- FileStore (TDD: Red -> Green)

1.  `features/tt/internal/agent/storage/filestore_test.go` を作成 -- テーブル駆動テスト:
    *   正常: atomic write 後にファイルが存在し、JSON が valid
    *   パス構造: `pending/YYYY/MM/DD/E-{ULID}.json` 形式
    *   ディレクトリ自動作成: 存在しないパスでもエラーにならない
2.  `features/tt/internal/agent/storage/filestore.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 10: Storage -- SQLite Index (TDD: Red -> Green)

1.  `features/tt/internal/agent/storage/index_test.go` を作成 -- テーブル駆動テスト:
    *   Store + 読み出し: event_id でレコードが取得できる
    *   冪等性: 同一 `client_request_id` で 2 回 Store -> 2 回目は既存 event_id を返す
    *   WAL mode: `PRAGMA journal_mode` が `wal` であることを確認
    *   FTS: task_summary のキーワード検索でヒットする
2.  `features/tt/internal/agent/storage/index.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 11: Storage -- Audit Log (TDD: Red -> Green)

1.  `features/tt/internal/agent/storage/audit_test.go` を作成
2.  `features/tt/internal/agent/storage/audit.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 12: Handler -- パイプラインオーケストレーション (TDD: Red -> Green)

1.  `features/tt/internal/agent/notify/handler_test.go` を作成 -- テスト:
    *   正常系: 有効な payload -> `accepted`, exit 0
    *   異常系: 不正 JSON -> `rejected`, exit 10
    *   異常系: スキーマ違反 -> `rejected`, exit 11
    *   冪等性: 同一 `client_request_id` -> 同じ event_id
    *   Git なし -> `accepted_with_warnings`, warnings に `NO_GIT_REPOSITORY`
2.  `features/tt/internal/agent/notify/handler.go` を実装
3.  テスト実行、Green を確認
4.  `git add && git commit`

### Step 13: コマンド層 (agent.go, agent_notify.go)

1.  `features/tt/cmd/agent.go` を作成 -- `tt agent` サブコマンドグループ
2.  `features/tt/cmd/agent_notify.go` を作成 -- `tt agent notify` コマンド
    *   CLI フラグの定義 (上記の全フラグ)
    *   JSON 経路 / CLI フラグ経路の分岐
    *   `--dry-run` / `--print-payload` の処理
    *   結果 JSON の stdout 出力
    *   exit code のマッピング
3.  `git add && git commit`

### Step 14: ビルドと検証

1.  Verification Plan を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "common" --specify "AgentNotify"
    ```
    *   **Log Verification**: `./scripts/utils/view-syslog.sh --tail 50` で `agent notify` 関連のログエントリを確認

### テスト項目設計のセルフレビュー

#### 網羅性の検証

「全テスト成功 = notify 機能が正しく動作している」と言えるか:

*   **スキーマバリデーション**: 正常・必須欠損・型不一致・境界値をカバー -> Yes
*   **正規化**: NFC・空白・空ノートをカバー -> Yes
*   **ID/Hash**: 一意性・同一性・branch 非依存性をカバー -> Yes
*   **Branch**: 正常 Git・Git なし・detached HEAD をカバー -> Yes
*   **FileStore**: atomic write・パス構造・ディレクトリ自動作成をカバー -> Yes
*   **Index**: CRUD・冪等性・WAL・FTS をカバー -> Yes
*   **Audit**: NDJSON フォーマット・成功/失敗分離をカバー -> Yes
*   **Handler**: パイプライン全体の正常/異常/冪等性をカバー -> Yes

#### ボトムアップ順序

テスト順序: schema -> normalize -> identity -> branch -> supplement -> filestore -> index -> audit -> handler -> cmd

これは依存関係 (handler -> storage -> identity/normalize/branch) のボトムアップ順序に合致する。

#### 観点チェックリスト

| # | 観点 | カバー状況 |
|---|------|-----------|
| 1 | 正常系の動作確認 | handler_test: 正常ペイロード -> accepted |
| 2 | 異常系・境界値 | schema_test: 境界値, normalize_test: 全空ノート |
| 3 | 外部連携の実動作 | index_test: SQLite CRUD, filestore_test: ファイル書き込み |
| 4 | データの一貫性 | filestore_test: 書き込み後読み出し一致 |
| 5 | 状態遷移の検証 | handler_test: 冪等性 (2 回目の呼び出し) |
| 6 | 設定・構成の反映 | index_test: WAL mode 確認 |
| 7 | 副作用の確認 | filestore_test: _tmp/ にファイルが残らないこと |

## Documentation

#### [MODIFY] [000-Agentic-Memory-Intake.md](file://prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/000-Agentic-Memory-Intake.md)
*   **更新内容**: 実装計画の作成に伴い、仕様書自体は変更不要 (既にレビュー済み)
