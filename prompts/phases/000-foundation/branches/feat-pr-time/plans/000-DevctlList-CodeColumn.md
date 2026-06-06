# 000-DevctlList-CodeColumn

> **Source Specification**: [000-DevctlList-CodeColumn.md](file://prompts/phases/000-foundation/ideas/feat-pr-time/000-DevctlList-CodeColumn.md)

## Goal Description

`devctl list` コマンドのテーブル出力に `CODE` 列を追加し、各ブランチの GitHub 上での状態（`(local)`, `hosted`, `PR(Nm ago)`, `deleted`）を表示する。また `STATE` 列を `CONTAINER` へリネームする。バックグラウンド更新プロセスと `--update` オプションを実装する。

## User Review Required

> [!IMPORTANT]
> **StateFile への `CodeStatus` フィールド追加**: 既存の YAML ステートファイル（`work/<branch>.state.yaml`）にコードホスティング状態を追加する。これは既存の StateFile を読み書きする全てのフローに影響する可能性がある。YAML の `omitempty` により、既存ファイルとの後方互換性は保たれる。

> [!WARNING]
> **バックグラウンドプロセス**: `devctl _update-code-status` という隠しサブコマンドを追加し、`devctl list` から `os/exec` で子プロセスとして起動する。親プロセスとは切り離す (`SysProcAttr` でデタッチ)。Windows 環境での動作に注意が必要。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: STATE→CONTAINER リネーム、CODE 列追加 | Proposed Changes > listing パッケージ |
| R2: CODE 列ステータス定義 (local/hosted/PR/deleted) | Proposed Changes > state パッケージ (CodeStatus 型) |
| R3: PR 経過時間フォーマット | Proposed Changes > listing パッケージ (FormatCodeColumn) |
| R4: 状態遷移 | Proposed Changes > codestatus パッケージ (Resolve 関数) |
| R5: バックグラウンド更新プロセス | Proposed Changes > codestatus パッケージ + cmd/_update_code_status.go |
| R6: ブランチ単位の最終更新日時 | Proposed Changes > state パッケージ (CodeStatus.LastCheckedAt) |
| R7: `--update` オプション | Proposed Changes > cmd/list.go |

## Proposed Changes

### state パッケージ

#### [MODIFY] [state.go](file://features/devctl/internal/state/state.go)
*   **Description**: `StateFile` に `CodeStatus` フィールドを追加し、コードホスティング状態を永続化可能にする。
*   **Technical Design**:
    ```go
    // CodeStatusType represents the code hosting status.
    type CodeStatusType string

    const (
        CodeStatusLocal   CodeStatusType = "local"
        CodeStatusHosted  CodeStatusType = "hosted"
        CodeStatusPR      CodeStatusType = "pr"
        CodeStatusDeleted CodeStatusType = "deleted"
    )

    // CodeStatus holds the code hosting service status for a branch.
    type CodeStatus struct {
        Status        CodeStatusType `yaml:"status"`
        PRCreatedAt   *time.Time     `yaml:"pr_created_at,omitempty"`
        LastCheckedAt *time.Time     `yaml:"last_checked_at,omitempty"`
    }
    ```
    `StateFile` 構造体に以下のフィールドを追加:
    ```go
    type StateFile struct {
        Branch     string                  `yaml:"branch"`
        CreatedAt  time.Time               `yaml:"created_at"`
        Features   map[string]FeatureState `yaml:"features,omitempty"`
        CodeStatus *CodeStatus             `yaml:"code_status,omitempty"` // 新規追加
    }
    ```
*   **Logic**:
    - `CodeStatus` はポインタ型で `omitempty` を付けることで、既存 YAML ファイルとの後方互換性を確保
    - `PRCreatedAt` は PR 作成日時を保持し、経過時間計算に使用
    - `LastCheckedAt` はブランチ毎の最終確認日時（R6 のブランチ単位更新用）

#### [MODIFY] [state_test.go](file://features/devctl/internal/state/state_test.go)
*   **Description**: `CodeStatus` フィールドの YAML シリアライズ/デシリアライズテストを追加。
*   **Technical Design**:
    - `TestStateFile_CodeStatus_RoundTrip`: CodeStatus を含む StateFile の Save/Load ラウンドトリップ
    - `TestStateFile_BackwardCompat`: CodeStatus なしの既存 YAML ファイルを Load できること
    - `TestStateFile_CodeStatus_OmitEmpty`: CodeStatus が nil の場合、YAML に出力されないこと

---

### codestatus パッケージ（新設）

#### [NEW] [codestatus.go](file://features/devctl/internal/codestatus/codestatus.go)
*   **Description**: GitHub 上のブランチ/PR 状態を取得し、`state.CodeStatus` を更新するロジック。
*   **Technical Design**:
    ```go
    package codestatus

    // Checker resolves the code hosting status for branches.
    type Checker struct {
        GitCmd   string        // git コマンドパス
        GhCmd    string        // gh コマンドパス
        RepoRoot string        // リポジトリルート
        Timeout  time.Duration // 子コマンドのタイムアウト
    }

    // BranchStatus holds the resolved status for a single branch.
    type BranchStatus struct {
        Status      state.CodeStatusType
        PRCreatedAt *time.Time
    }

    // Resolve checks the current code hosting status for a branch.
    func (c *Checker) Resolve(branch string) (BranchStatus, error)

    // UpdateAll updates code status for all given branches in state files.
    func (c *Checker) UpdateAll(branches []string) error
    ```
*   **Logic**:
    - `Resolve` のロジック:
        1. `git ls-remote --heads origin <branch>` を実行
        2. 出力が空 → 以前 `hosted` or `pr` だった場合は `deleted`、そうでなければ `local`
        3. 出力がある → `gh pr list --head <branch> --json number,createdAt --limit 1` を実行
        4. PR が存在すれば `pr`（`createdAt` を保持）、なければ `hosted`
    - `UpdateAll` は対象ブランチを順次 `Resolve` して `state.Save` で永続化
    - 各外部コマンドには `context.WithTimeout` を適用

#### [NEW] [codestatus_test.go](file://features/devctl/internal/codestatus/codestatus_test.go)
*   **Description**: Checker のロジックをテスト。外部コマンドの出力をモックして検証。
*   **Technical Design**:
    - テスト用の `FakeRunner` を用いて git/gh コマンドの出力をモック
    - テストケース:
        - `TestResolve_Local`: ls-remote が空 → `local`
        - `TestResolve_Hosted`: ls-remote にある＆PR なし → `hosted`
        - `TestResolve_PR`: ls-remote にある＆PR あり → `pr` (createdAt 検証)
        - `TestResolve_Deleted`: 以前 hosted → ls-remote 空 → `deleted`
        - `TestUpdateAll`: 複数ブランチの一括更新

---

### codestatus バックグラウンドプロセス

#### [NEW] [bgrunner.go](file://features/devctl/internal/codestatus/bgrunner.go)
*   **Description**: バックグラウンド更新プロセスの起動・制御ロジック。
*   **Technical Design**:
    ```go
    const (
        RefreshInterval = 5 * time.Minute   // 更新間隔
        ProcessTimeout  = 2 * time.Minute   // 子プロセスタイムアウト
        LockFileName    = ".codestatus.lock" // ロックファイル名
    )

    // NeedsUpdate checks if any branch needs a code status update.
    // Returns true if LastCheckedAt is nil or older than RefreshInterval.
    func NeedsUpdate(states map[string]state.StateFile) bool

    // BranchesNeedingUpdate returns branch names whose code status is stale.
    func BranchesNeedingUpdate(states map[string]state.StateFile) []string

    // IsRunning checks if the background updater is already running.
    // Uses lock file at <repoRoot>/work/<LockFileName>.
    func IsRunning(repoRoot string) bool

    // StartBackground starts the background updater process.
    // Spawns `devctl _update-code-status` as a detached child process.
    // Returns immediately; does not wait for completion.
    func StartBackground(repoRoot, devctlBinary string) error

    // AcquireLock creates the lock file with the current PID.
    // Returns error if lock already held.
    func AcquireLock(repoRoot string) error

    // ReleaseLock removes the lock file.
    func ReleaseLock(repoRoot string)
    ```
*   **Logic**:
    - `NeedsUpdate`: 全ブランチの `CodeStatus.LastCheckedAt` を確認。nil もしくは `RefreshInterval` 以上経過しているブランチがあれば true
    - `BranchesNeedingUpdate`: 条件に合致するブランチ名のスライスを返す
    - `IsRunning`:
        1. ロックファイルを確認
        2. ロックファイルが存在し、中に記載された PID のプロセスが存在すれば true
        3. ロックファイルが存在するが PID プロセスが存在しない場合は、ロックファイルを自動削除して false を返す（ゾンビロック対策）
    - `StartBackground`:
        1. `IsRunning` チェック、true なら何もしない
        2. 自身のバイナリを `_update-code-status` サブコマンド付きで起動
        3. `cmd.SysProcAttr` でプロセスをデタッチ
        4. `cmd.Start()` で起動後、即 return（Wait しない）

#### [NEW] [bgrunner_test.go](file://features/devctl/internal/codestatus/bgrunner_test.go)
*   **Description**: バックグラウンドプロセス制御のテスト。
*   **Technical Design**:
    - `TestNeedsUpdate_Fresh`: LastCheckedAt が1分前 → false
    - `TestNeedsUpdate_Stale`: LastCheckedAt が10分前 → true
    - `TestNeedsUpdate_Nil`: LastCheckedAt が nil → true
    - `TestIsRunning_NoLock`: ロックファイルなし → false
    - `TestIsRunning_StaleLock`: ロックファイルあり＆プロセスなし → false（自動クリーンアップ）
    - `TestAcquireLock_ReleaseLock`: ロック取得・解放サイクル

---

### listing パッケージ

#### [MODIFY] [listing_test.go](file://features/devctl/internal/listing/listing_test.go)
*   **Description**: テストを先に変更。`STATE` → `CONTAINER` リネーム、`CODE` 列追加に対応。
*   **Technical Design**:
    - `TestFormatTable` の既存テスト:
        - `assert.Contains(t, out, "STATE")` → `assert.Contains(t, out, "CONTAINER")` に変更
        - `assert.NotContains(t, out, "STATE")` を追加
    - `TestFormatTable` に `CODE` 列検証を追加:
        - `assert.Contains(t, out, "CODE")` でヘッダー確認
    - `BranchInfo` 構造体のテストデータに `CodeStatus` フィールドを追加
    - 新規テスト `TestFormatCodeColumn`:
        - 各経過時間フォーマットを検証するテーブル駆動テスト
        - ケース: 3分 → `PR(3m ago)`, 2時間 → `PR(2h ago)`, 5日 → `PR(5d ago)`, 31日 → `PR(01/15)` 等

#### [MODIFY] [listing.go](file://features/devctl/internal/listing/listing.go)
*   **Description**: `STATE` 列を `CONTAINER` にリネーム、`CODE` 列追加、`BranchInfo` にフィールド追加。
*   **Technical Design**:
    `BranchInfo` に以下のフィールドを追加:
    ```go
    type BranchInfo struct {
        Branch       string            `json:"branch"`
        Path         string            `json:"path"`
        Features     []FeatureInfo     `json:"features"`
        MainWorktree bool              `json:"main_worktree,omitempty"`
        CodeStatus   *state.CodeStatus `json:"code_status,omitempty"` // 新規追加
    }
    ```
    `stateColumn` 関数を `containerColumn` にリネーム（内部ロジックは同一）。

    新規関数 `FormatCodeColumn`:
    ```go
    // FormatCodeColumn builds a display string for the CODE column.
    // now パラメータはテスト容易性のために注入する。
    func FormatCodeColumn(bi BranchInfo, now time.Time) string
    ```
    ロジック:
    - `bi.MainWorktree` → `-`
    - `bi.CodeStatus == nil` → `(unknown)`
    - `bi.CodeStatus.Status == "local"` → `(local)`
    - `bi.CodeStatus.Status == "hosted"` → `hosted`
    - `bi.CodeStatus.Status == "deleted"` → `deleted`
    - `bi.CodeStatus.Status == "pr"`:
        - `bi.CodeStatus.PRCreatedAt` が nil → `PR`
        - 経過時間を計算:
            - `< 60m` → `PR({n}m ago)`
            - `< 24h` → `PR({n}h ago)`
            - `< 30d` → `PR({n}d ago)`
            - `>= 30d` → `PR(MM/DD)` (PRCreatedAt の月/日)

    `FormatTable` 関数変更:
    - ヘッダー: `STATE` → `CONTAINER`、`CODE` 列を `CONTAINER` と `PATH` の間に追加
    - 各行: `containerColumn(bi)` と `FormatCodeColumn(bi, time.Now())` を出力

    `CollectBranches` 関数変更:
    - `states` から `CodeStatus` を `BranchInfo.CodeStatus` にコピー

---

### cmd パッケージ

#### [NEW] [_update_code_status.go](file://features/devctl/cmd/_update_code_status.go)
*   **Description**: バックグラウンド更新用の隠しサブコマンド。
*   **Technical Design**:
    ```go
    var updateCodeStatusCmd = &cobra.Command{
        Use:    "_update-code-status",
        Short:  "Internal: update code hosting status (background)",
        Hidden: true,
        RunE:   runUpdateCodeStatus,
    }
    ```
    `runUpdateCodeStatus` ロジック:
    1. `codestatus.AcquireLock(repoRoot)` でロック取得。失敗なら exit
    2. `defer codestatus.ReleaseLock(repoRoot)`
    3. `context.WithTimeout(context.Background(), codestatus.ProcessTimeout)` で全体タイムアウト設定
    4. `state.ScanStateFiles(repoRoot)` でブランチ一覧取得
    5. `codestatus.BranchesNeedingUpdate(states)` で更新対象を絞り込み
    6. `checker.UpdateAll(branches)` で各ブランチの状態を取得・保存
    7. エラーが出ても可能な範囲で続行し、ロック解放して終了

#### [MODIFY] [list.go](file://features/devctl/cmd/list.go)
*   **Description**: `--update` フラグ追加、テーブル出力後のバックグラウンドメッセージ表示。
*   **Technical Design**:
    フラグ追加:
    ```go
    var flagListUpdate bool

    func init() {
        listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
        listCmd.Flags().BoolVar(&flagListPath, "path", false, "Show worktree path column")
        listCmd.Flags().BoolVar(&flagListUpdate, "update", false, "Force update code status immediately")
    }
    ```
    `runListBranches` ロジック変更:
    1. 既存: worktree 一覧取得、state スキャン、`CollectBranches`
    2. 追加: `flagListUpdate` が true の場合
        - `codestatus.Checker` を作成、フォアグラウンドで `UpdateAll` を実行
        - 完了後、state を再スキャン
    3. 追加: `flagListUpdate` が false の場合
        - `codestatus.NeedsUpdate(states)` をチェック
        - true なら `codestatus.StartBackground(repoRoot, os.Executable())` で子プロセス起動
    4. テーブル出力
    5. 追加: `codestatus.IsRunning(repoRoot)` が true の場合
        - `fmt.Fprintln(os.Stderr, "* update process is still running in the background.")` を出力

#### [MODIFY] [root.go](file://features/devctl/cmd/root.go)
*   **Description**: 隠しサブコマンド `_update-code-status` を登録。
*   **Technical Design**:
    `init()` に追加:
    ```go
    rootCmd.AddCommand(updateCodeStatusCmd)
    ```

---

### github パッケージ

#### [MODIFY] [github.go](file://features/devctl/internal/github/github.go)
*   **Description**: PR 一覧取得機能を追加。
*   **Technical Design**:
    ```go
    // PRInfo holds information about a pull request.
    type PRInfo struct {
        Number    int       `json:"number"`
        CreatedAt time.Time `json:"createdAt"`
    }

    // ListPRs returns open PRs matching the given head branch.
    // Uses `gh pr list --head <branch> --json number,createdAt --limit 1`.
    func (c *Client) ListPRs(workDir, branch string) ([]PRInfo, error)
    ```
    ロジック:
    - `gh pr list --head <branch> --json number,createdAt --limit 1` を実行
    - JSON 出力をパースして `[]PRInfo` を返す
    - 結果が空配列であれば PR なし
    - `cmdRunner` がセットされていれば `Runner.RunWithOpts` を使用、なければ直接 `exec.Command`

#### [MODIFY] [github_test.go](file://features/devctl/internal/github/github_test.go)
*   **Description**: `ListPRs` のテストを追加。
*   **Technical Design**:
    - `TestListPRs_WithPR`: gh コマンドが PR 情報を返すケースをモック
    - `TestListPRs_NoPR`: gh コマンドが空配列を返すケース

## Step-by-Step Implementation Guide

> [!IMPORTANT]
> TDD 原則に従い、各ステップでテストを先に書き、失敗を確認してから実装する。

### Phase 1: データモデル拡張

- [x] 1. **state パッケージのテスト追加**:
    - `state_test.go` に `TestStateFile_CodeStatus_RoundTrip`, `TestStateFile_BackwardCompat`, `TestStateFile_CodeStatus_OmitEmpty` を追加
    - `./scripts/process/build.sh` でテスト失敗を確認

- [x] 2. **state パッケージの実装**:
    - `state.go` に `CodeStatusType`, `CodeStatus` 型を追加
    - `StateFile` に `CodeStatus *CodeStatus` フィールドを追加
    - `./scripts/process/build.sh` でテスト成功を確認

### Phase 2: listing パッケージの列変更

- [x] 3. **listing テストの更新**:
    - `listing_test.go` の `TestFormatTable` を更新:
        - `STATE` → `CONTAINER` のアサーション変更
        - `CODE` 列のアサーション追加
    - `TestFormatCodeColumn` テーブル駆動テストを新規追加（経過時間フォーマット）
    - `BranchInfo` のテストデータに `CodeStatus` フィールド追加
    - `./scripts/process/build.sh` でテスト失敗を確認

- [x] 4. **listing パッケージの実装**:
    - `BranchInfo` に `CodeStatus` フィールド追加
    - `stateColumn` → `containerColumn` リネーム
    - `FormatCodeColumn` 関数を新規作成
    - `FormatTable` でヘッダーと行出力を変更
    - `CollectBranches` で `CodeStatus` をコピー
    - `./scripts/process/build.sh` でテスト成功を確認

### Phase 3: GitHub PR 情報取得

- [x] 5. **github パッケージのテスト追加**:
    - `github_test.go` に `TestListPRs_WithPR`, `TestListPRs_NoPR` を追加
    - `./scripts/process/build.sh` でテスト失敗を確認

- [x] 6. **github パッケージの実装**:
    - `github.go` に `PRInfo` 型と `ListPRs` メソッドを追加
    - `./scripts/process/build.sh` でテスト成功を確認

### Phase 4: codestatus パッケージ（新設）

- [x] 7. **codestatus テスト作成**:
    - `codestatus_test.go` に Resolve 関連テストを作成
    - `bgrunner_test.go` に NeedsUpdate/IsRunning/Lock 関連テストを作成
    - `./scripts/process/build.sh` でテスト失敗を確認

- [x] 8. **codestatus 実装**:
    - `codestatus.go`: `Checker`, `BranchStatus`, `Resolve`, `UpdateAll` を実装
    - `bgrunner.go`: `NeedsUpdate`, `BranchesNeedingUpdate`, `IsRunning`, `StartBackground`, `AcquireLock`, `ReleaseLock` を実装
    - `./scripts/process/build.sh` でテスト成功を確認

### Phase 5: cmd パッケージ統合

- [x] 9. **隠しサブコマンド追加**:
    - `cmd/_update_code_status.go` を作成
    - `cmd/root.go` に `AddCommand` 追加
    - `./scripts/process/build.sh` でビルド成功を確認

- [x] 10. **list.go の更新**:
    - `--update` フラグ追加
    - `runListBranches` にバックグラウンド起動ロジック追加
    - テーブル出力後のメッセージ表示追加
    - `./scripts/process/build.sh` で成功を確認

### Phase 6: 統合テスト

- [x] 11. **統合テスト作成**:
    - `tests/integration-test/devctl_list_code_test.go` を新規作成
    - テストケース:
        - `devctl list` 実行時に `CONTAINER` と `CODE` ヘッダーが表示される
        - `STATE` ヘッダーが表示されない
        - `--update` オプションが受け入れられる

- [x] 12. **全体検証**:
    - `./scripts/process/build.sh && ./scripts/process/integration_test.sh` で全テストパス確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "TestDevctlListCode"
    ```
    *   **Log Verification**: テスト出力に `CONTAINER` ヘッダーが表示され、`STATE` が表示されないことを確認。`CODE` 列が出力に含まれることを確認。

## Documentation

#### [MODIFY] [000-DevctlList-CodeColumn.md](file://prompts/phases/000-foundation/ideas/feat-pr-time/000-DevctlList-CodeColumn.md)
*   **更新内容**: 実装後に実装実態と乖離がある場合のみ更新。現時点では変更なし。
