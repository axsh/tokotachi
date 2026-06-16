# 005-Far-Knowledge-Skillification-Part1

> **Source Specification**: [005-Far-Knowledge-Skillification.md](../ideas/005-Far-Knowledge-Skillification.md)

## Goal Description

Part1 では Go コード変更に集中する。`tt agent` サブコマンドの改名 (notify -> record)、フラグ追加、新規コマンド (intake processed, knowledge add/append/list/split/merge/rename/move)、および廃止コマンド (assist, task) の削除を実装する。

## User Review Required

> [!IMPORTANT]
> `tt agent notify` -> `tt agent record` の改名は**破壊的変更**。既存の統合テスト (`tt_agent_assist_test.go`) にも影響する。

> [!WARNING]
> Knowledge Atom 関連の型 (`KnowledgeAtom`, `KnowledgeAtomType` 等) を types.go から削除するが、これらが他のコードから参照されていないか確認が必要。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: notify フラグ追加 (design-pattern, convention, lesson-learned, preference) | Proposed Changes > agent_record.go, types.go |
| R1: notify -> record 改名 | Proposed Changes > agent_record.go, record/ パッケージ |
| R3: tt agent intake processed サブコマンド | Proposed Changes > agent_intake.go, intake/ |
| R8: tt agent knowledge サブコマンド群 (add/append/list/split/merge/rename/move) | Proposed Changes > agent_knowledge.go, knowledge/ |
| tt agent assist 廃止 | Proposed Changes > DELETE agent_assist.go, assist/ |
| tt agent task 廃止 | Proposed Changes > DELETE agent_task.go, task/ |
| スキーマ改名 (agent-notify-payload -> agent-record-payload 等) | Proposed Changes > schemas/ |
| スキーマ廃止 (agent-task, knowledge-atom, knowledge-atom-batch) | Proposed Changes > schemas/ |
| Knowledge Atom 型の削除 | Proposed Changes > types.go |

## Proposed Changes

### 1. 型定義とスキーマ

#### [MODIFY] [types.go](file://features/tt/internal/agent/types.go)
*   **Description**: Flags に新規フィールド追加。Knowledge Atom 関連型を削除。
*   **Technical Design**:
    ```go
    // Flags represents boolean flags for categorization.
    type Flags struct {
        ArchitectureImpact      bool `json:"architecture_impact,omitempty"`
        MemoryRelated           bool `json:"memory_related,omitempty"`
        PromptRelated           bool `json:"prompt_related,omitempty"`
        AgentBehaviorRelated    bool `json:"agent_behavior_related,omitempty"`
        RequiresImmediateAction bool `json:"requires_immediate_action,omitempty"`
        // 新規フィールド (R1)
        DesignPattern  bool `json:"design_pattern,omitempty"`
        Convention     bool `json:"convention,omitempty"`
        LessonLearned  bool `json:"lesson_learned,omitempty"`
        Preference     bool `json:"preference,omitempty"`
    }
    ```
*   **Logic**:
    *   L20-26: `Flags` 構造体に4つの新規フィールドを追加
    *   L132-L231: `KnowledgeAtomType`, `ValidKnowledgeTypes`, `KnowledgeAtom`, `ActivationHints`, `KnowledgeSource`, `KnowledgeTS` 等を削除

#### [MODIFY] [types_test.go](file://features/tt/internal/agent/types_test.go)
*   **Description**: Flags 新フィールドのテスト追加。Knowledge Atom テストを削除。

#### [MODIFY] [agent-notify-payload.schema.json -> agent-record-payload.schema.json](file://prompts/memory/schemas/agent-notify-payload.schema.json)
*   **Description**: ファイル名を `agent-record-payload.schema.json` に改名。`flags` オブジェクトに新規フィールド追加。
*   **Technical Design**:
    ```json
    {
      "flags": {
        "properties": {
          "architecture_impact": { "type": "boolean" },
          "memory_related": { "type": "boolean" },
          "prompt_related": { "type": "boolean" },
          "agent_behavior_related": { "type": "boolean" },
          "requires_immediate_action": { "type": "boolean" },
          "design_pattern": { "type": "boolean" },
          "convention": { "type": "boolean" },
          "lesson_learned": { "type": "boolean" },
          "preference": { "type": "boolean" }
        }
      }
    }
    ```

#### [MODIFY] [agent-notify-result.schema.json -> agent-record-result.schema.json](file://prompts/memory/schemas/agent-notify-result.schema.json)
*   **Description**: ファイル名を `agent-record-result.schema.json` に改名。内容変更なし。

#### [DELETE] [agent-task.schema.json](file://prompts/memory/schemas/agent-task.schema.json)
*   **Description**: `tt agent task` 廃止に伴い削除。

#### [DELETE] [knowledge-atom.schema.json](file://prompts/memory/schemas/knowledge-atom.schema.json)
*   **Description**: Knowledge Atom パイプライン廃止に伴い削除。

#### [DELETE] [knowledge-atom-batch.schema.json](file://prompts/memory/schemas/knowledge-atom-batch.schema.json)
*   **Description**: 同上。

---

### 2. record コマンド (notify -> record 改名)

#### [NEW] [agent_record.go](file://features/tt/cmd/agent_record.go)
*   **Description**: `agent_notify.go` を `agent_record.go` に改名し、コマンド名を `record` に変更。新規フラグを追加。
*   **Technical Design**:
    ```go
    var agentRecordCmd = &cobra.Command{
        Use:   "record",
        Short: "Record a far-knowledge intake event",
        Long: `Record a far-knowledge intake event from a coding agent.
    ...`,
        RunE: runAgentRecord,
    }

    // 新規フラグ変数
    var (
        // 既存フラグ (名前を recordXxx に変更)
        recordFile         string
        recordStdin        bool
        recordAgent        string
        recordSummary      string
        recordSummaryFile  string
        recordNotes        []string
        recordNotesFile    string
        recordChangedPaths []string
        recordFromGit      bool
        recordFlagArch     bool
        recordFlagMem      bool
        recordFlagPrompt   bool
        recordFlagAgent    bool
        recordFlagUrgent   bool
        // 新規フラグ (R1)
        recordFlagDesignPattern bool
        recordFlagConvention    bool
        recordFlagLessonLearned bool
        recordFlagPreference    bool
        recordClientReqID       string
        recordDryRun            bool
        recordPrintPayload      bool
    )
    ```
*   **Logic**:
    *   `init()` 内で `--design-pattern`, `--convention`, `--lesson-learned`, `--preference` フラグを追加登録
    *   `buildPayloadFromFlags()` 内で新規フラグを `Flags` 構造体に反映。既存の5フラグ + 新規4フラグのいずれかが true なら `payload.Flags` を生成
    *   `agentCmd.AddCommand(agentRecordCmd)` でコマンド登録
    *   スキーマパス `agent-notify-payload.schema.json` を `agent-record-payload.schema.json` に変更

#### [DELETE] [agent_notify.go](file://features/tt/cmd/agent_notify.go)
*   **Description**: `agent_record.go` に移行したため削除。

#### [MODIFY] [notify/ -> record/](file://features/tt/internal/agent/notify/)
*   **Description**: パッケージ名を `notify` -> `record` に改名。全 .go ファイルの `package notify` を `package record` に変更。import パスも更新。
*   **Logic**:
    *   `features/tt/internal/agent/notify/` -> `features/tt/internal/agent/record/` にディレクトリ移動
    *   全ファイル (handler.go, branch.go, identity.go, normalize.go, schema.go, supplement.go) の `package` 宣言を変更
    *   schema.go 内のスキーマファイル参照 `agent-notify-payload.schema.json` を `agent-record-payload.schema.json` に変更
    *   テストファイルも同様に更新

---

### 3. intake processed サブコマンド

#### [MODIFY] [agent_intake.go](file://features/tt/cmd/agent_intake.go)
*   **Description**: `processed` サブコマンドを追加。
*   **Technical Design**:
    ```go
    var agentIntakeProcessedCmd = &cobra.Command{
        Use:   "processed [event-id]",
        Short: "Move an intake event from pending to processed",
        Args:  cobra.ExactArgs(1),
        RunE:  runAgentIntakeProcessed,
    }

    func init() {
        // 既存の list, show に加えて processed を登録
        agentIntakeCmd.AddCommand(agentIntakeProcessedCmd)
    }

    func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
        varDir := filepath.Join("prompts", "memory", "var")
        eventID := args[0]
        err := intake.MoveToProcessed(varDir, eventID)
        if err != nil {
            return fmt.Errorf("failed to move event to processed: %w", err)
        }
        fmt.Printf("Event %s moved to processed\n", eventID)
        return nil
    }
    ```

#### [NEW] [intake/processed.go](file://features/tt/internal/agent/intake/processed.go)
*   **Description**: pending -> processed 移行ロジック。
*   **Technical Design**:
    ```go
    package intake

    // MoveToProcessed moves an intake event from pending/ to processed/.
    // 1. varDir/intake/pending/ を走査して event-id に一致する JSON を探す
    // 2. varDir/intake/processed/<date>/ に移動
    // 3. 元のディレクトリが空なら削除
    func MoveToProcessed(varDir, eventID string) error
    ```
*   **Logic**:
    *   `pending/` 配下のサブディレクトリを再帰走査し、`event_id` フィールドが一致する JSON を探す
    *   `processed/` 配下に同日付のディレクトリを作成し、ファイルを移動 (`os.Rename`)
    *   元の日付ディレクトリが空になったら `os.Remove` で削除
    *   該当ファイルが見つからない場合は `fmt.Errorf("event %s not found in pending", eventID)` を返す

#### [NEW] [intake/processed_test.go](file://features/tt/internal/agent/intake/processed_test.go)
*   **Description**: MoveToProcessed の単体テスト。
*   **Test Cases**:
    *   正常系: pending にあるイベントが processed に移動する
    *   正常系: 移動後に空ディレクトリが削除される
    *   異常系: 存在しない event-id を指定した場合にエラー
    *   異常系: 既に processed にある event-id を指定した場合にエラー

---

### 4. knowledge サブコマンド群

#### [NEW] [agent_knowledge.go](file://features/tt/cmd/agent_knowledge.go)
*   **Description**: `tt agent knowledge` サブコマンドグループの CLI 定義。
*   **Technical Design**:
    ```go
    var agentKnowledgeCmd = &cobra.Command{
        Use:   "knowledge",
        Short: "Manage far-knowledge categories",
    }

    // サブコマンド: add, append, list, split, merge, rename, move
    var agentKnowledgeAddCmd = &cobra.Command{
        Use:   "add",
        Short: "Create a new category and add a knowledge file",
        RunE:  runAgentKnowledgeAdd,
    }

    // add フラグ
    var (
        knowledgeAddCategoryPath string  // --category-path
        knowledgeAddTitle        string  // --title
        knowledgeAddDescription  string  // --description
        knowledgeAddContentFile  string  // --content-file
        knowledgeAddSourceEvents string  // --source-events (comma separated)
    )
    ```
    以下同様に append, list, split, merge, rename, move のそれぞれに対応する Command と フラグ を定義。

#### [NEW] [knowledge/](file://features/tt/internal/agent/knowledge/)
*   **Description**: カテゴリ操作ロジック。以下のファイルを含む:

##### [NEW] knowledge/types.go
*   **Technical Design**:
    ```go
    package knowledge

    import "time"

    // CategoryMeta represents _category.yaml content.
    type CategoryMeta struct {
        CategoryID  string    `yaml:"category_id"`
        Title       string    `yaml:"title"`
        Description string    `yaml:"description"`
        CreatedAt   time.Time `yaml:"created_at"`
        LastUpdated time.Time `yaml:"last_updated"`
    }

    // KnowledgeFileMeta represents frontmatter of a knowledge .md file.
    type KnowledgeFileMeta struct {
        KnowledgeID    string    `yaml:"knowledge_id"`
        Title          string    `yaml:"title"`
        CategoryPath   string    `yaml:"category_path"`
        CreatedAt      time.Time `yaml:"created_at"`
        LastUpdated    time.Time `yaml:"last_updated"`
        SourceEventIDs []string  `yaml:"source_event_ids"`
    }
    ```

##### [NEW] knowledge/store.go
*   **Description**: `prompts/memory/knowledge/` 配下のファイル操作を抽象化する Store。
*   **Technical Design**:
    ```go
    // Store manages the knowledge directory tree.
    type Store struct {
        RootDir string // prompts/memory/knowledge/
    }

    // NewStore creates a new Store.
    func NewStore(rootDir string) *Store

    // Add creates a new category directory (if needed) with _category.yaml
    // and writes a knowledge file.
    func (s *Store) Add(categoryPath, title, description, contentFile string, sourceEvents []string) error

    // Append adds a new knowledge file to an existing category.
    func (s *Store) Append(categoryPath, title, contentFile string, sourceEvents []string) error

    // List returns the category tree with statistics.
    func (s *Store) List() ([]CategoryInfo, error)

    // Split splits a category into subcategories based on a plan file.
    func (s *Store) Split(categoryPath string, intoNames []string, planFile string) error

    // Merge merges multiple categories into one.
    func (s *Store) Merge(categoryPaths []string, into string, planFile string) error

    // Rename renames a category directory and updates metadata.
    func (s *Store) Rename(oldPath, newPath, newTitle string) error

    // Move moves a knowledge file to a different category.
    func (s *Store) Move(fromFile, toCategoryPath string) error
    ```

##### [NEW] knowledge/store_test.go
*   **Description**: Store の全メソッドの単体テスト。
*   **Test Cases** (テーブル駆動):
    *   **Add**: 新規カテゴリ + 知識ファイル作成。_category.yaml の生成確認。
    *   **Add**: 既存カテゴリへの2回目の add。_category.yaml が上書きされないこと。
    *   **Append**: 既存カテゴリに知識ファイル追加。
    *   **Append**: 存在しないカテゴリにはエラー。
    *   **List**: 空ディレクトリ -> 0件。
    *   **List**: 3カテゴリ + サブカテゴリのツリー表示。
    *   **Split**: 1カテゴリ -> 2サブカテゴリ。元ファイルの振り分け。_category.yaml の整合性。
    *   **Merge**: 2カテゴリ -> 1カテゴリ。source_event_ids の結合。
    *   **Rename**: ディレクトリ名変更 + _category.yaml 更新。
    *   **Move**: 知識ファイルの移動 + frontmatter の category_path 更新。

##### [NEW] knowledge/plan.go
*   **Description**: split/merge の計画ファイル (JSON) のパーサー。
*   **Technical Design**:
    ```go
    // SplitPlan represents a split operation plan.
    type SplitPlan struct {
        Assignments map[string]string `json:"assignments"` // knowledge_file -> target_subcategory
    }

    // MergePlan represents a merge operation plan.
    type MergePlan struct {
        Title       string `json:"title"`
        Description string `json:"description"`
    }

    func ParseSplitPlan(planFile string) (*SplitPlan, error)
    func ParseMergePlan(planFile string) (*MergePlan, error)
    ```

##### [NEW] knowledge/frontmatter.go
*   **Description**: マークダウンファイルの YAML frontmatter の読み書き。
*   **Technical Design**:
    ```go
    // ReadFrontmatter reads YAML frontmatter from a markdown file.
    func ReadFrontmatter(path string) (*KnowledgeFileMeta, string, error)
    // returns: meta, body, error

    // WriteFrontmatter writes YAML frontmatter + body to a markdown file.
    func WriteFrontmatter(path string, meta *KnowledgeFileMeta, body string) error

    // ReadCategoryMeta reads _category.yaml.
    func ReadCategoryMeta(path string) (*CategoryMeta, error)

    // WriteCategoryMeta writes _category.yaml.
    func WriteCategoryMeta(path string, meta *CategoryMeta) error
    ```

---

### 5. 廃止コマンドの削除

#### [DELETE] [agent_assist.go](file://features/tt/cmd/agent_assist.go)

#### [DELETE] [assist/](file://features/tt/internal/agent/assist/)

#### [DELETE] [agent_task.go](file://features/tt/cmd/agent_task.go)

#### [DELETE] [task/](file://features/tt/internal/agent/task/)

#### [DELETE] [tt_agent_assist_test.go](file://tests/tt/tt_agent_assist_test.go)

---

## Step-by-Step Implementation Guide

1.  **Flags 追加 + Knowledge Atom 削除 (types.go)**:
    *   Edit `features/tt/internal/agent/types.go`:
        *   `Flags` に `DesignPattern`, `Convention`, `LessonLearned`, `Preference` を追加
        *   L132-L271 の Knowledge Atom 関連型を全て削除
    *   Edit `features/tt/internal/agent/types_test.go` を対応更新

2.  **スキーマファイルの改名・削除**:
    *   `prompts/memory/schemas/agent-notify-payload.schema.json` -> `agent-record-payload.schema.json` に改名。flags に4フィールド追加。
    *   `prompts/memory/schemas/agent-notify-result.schema.json` -> `agent-record-result.schema.json` に改名
    *   `agent-task.schema.json`, `knowledge-atom.schema.json`, `knowledge-atom-batch.schema.json` を削除

3.  **notify -> record パッケージ改名**:
    *   `features/tt/internal/agent/notify/` -> `features/tt/internal/agent/record/` にディレクトリ移動
    *   全ファイルの `package notify` -> `package record` に変更
    *   schema.go 内のスキーマファイル参照を更新

4.  **agent_record.go の作成**:
    *   `features/tt/cmd/agent_notify.go` を `agent_record.go` にコピー
    *   コマンド名、変数名、import パスを全て更新
    *   新規フラグを `init()` と `buildPayloadFromFlags()` に追加
    *   元の `agent_notify.go` を削除

5.  **assist / task の削除**:
    *   `features/tt/cmd/agent_assist.go` を削除
    *   `features/tt/internal/agent/assist/` を削除
    *   `features/tt/cmd/agent_task.go` を削除
    *   `features/tt/internal/agent/task/` を削除
    *   `tests/tt/tt_agent_assist_test.go` を削除

6.  **intake processed の実装**:
    *   `features/tt/internal/agent/intake/processed_test.go` を作成 (TDD: まず失敗するテスト)
    *   `features/tt/internal/agent/intake/processed.go` を実装
    *   `features/tt/cmd/agent_intake.go` に `processed` サブコマンドを追加

7.  **knowledge パッケージの実装**:
    *   `features/tt/internal/agent/knowledge/types.go` を作成
    *   `features/tt/internal/agent/knowledge/frontmatter.go` + テストを作成 (TDD)
    *   `features/tt/internal/agent/knowledge/plan.go` + テストを作成 (TDD)
    *   `features/tt/internal/agent/knowledge/store_test.go` を作成 (TDD: 全テストケース)
    *   `features/tt/internal/agent/knowledge/store.go` を実装 (テストを通す)
    *   `features/tt/cmd/agent_knowledge.go` を作成

8.  **ビルド確認**:
    *   `scripts/process/build.sh --skip-frontend --skip-etc` でコンパイルエラーがないこと確認

9.  **Verification Plan の実行** (後述)

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh --skip-frontend --skip-etc
    ```

2.  **Integration Tests** (CLI サブコマンドの変更を含むため必須):
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "common" --specify "AgentRecord"
    ```
    *   **Log Verification**: `tt agent record` コマンドが正常に動作し、pending に intake event が保存されること

    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "common" --specify "AgentIntakeProcessed"
    ```
    *   **Log Verification**: `tt agent intake processed` でイベントが pending -> processed に移動すること

    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories "common" --specify "AgentKnowledge"
    ```
    *   **Log Verification**: `tt agent knowledge add/append/list/split/merge/rename/move` が正常動作すること

3.  **最終全体検証**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```

### GUI E2E Tests

GUI関連の変更なし。E2E テスト不要。

## Documentation

#### [MODIFY] [005-Far-Knowledge-Skillification.md](file://prompts/phases/000-foundation/branches/fix-memory-compiling/ideas/005-Far-Knowledge-Skillification.md)
*   **更新内容**: 実装後に検証結果を追記

---

## 継続計画について

本実装は Part1 (Go コード変更), Part2 (Wrapper スクリプト + プロンプト/ワークフロー変更), Part3 (Emitter 拡張 + E2E テスト) の3 Part に分割する。

*   **Part2**: Wrapper スクリプトの整理、capability/policy の改名・内容改修、systematize-far-knowledge ワークフロー作成
*   **Part3**: emitter 拡張 (branches/*/skills/ からの集約 compile)、E2E シナリオテスト
