# 000-LocalMerge

> **Source Specification**: [001-LocalMerge.md](file://prompts/phases/000-foundation/ideas/feat-local-merge-request/001-LocalMerge.md)

## Goal Description

`tt merge <branch>` コマンドを新設し、GitHub PR を経由せずローカルの `git merge` だけでブランチをマージできるようにする。あわせて、`tt open` 時に親ブランチ（BaseBranch）を `StateFile` に記録し、マージ先ブランチの自動解決と誤マージ防止を実現する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: `StateFile` に `BaseBranch` フィールド追加 | Proposed Changes > state > `state.go`, `state_test.go` |
| R1: `tt open` 時に BaseBranch を記録 | Proposed Changes > cmd > `open.go` |
| R2: `tt merge <branch>` コマンド新設 | Proposed Changes > cmd > `merge.go`, action > `merge.go` |
| R3: マージ戦略オプション (`--ff-only`, `--no-ff`, `--ff`) | Proposed Changes > cmd > `merge.go`, action > `merge.go` |
| R4: 事前チェック（uncommitted changes, dirty root） | Proposed Changes > action > `merge.go` |
| R5: コンフリクト処理（中断＋状況表示） | Proposed Changes > action > `merge.go` |
| R6: 誤マージ防止（BaseBranch 不一致時の拒否） | Proposed Changes > action > `merge.go` |

## Proposed Changes

### state パッケージ

#### [MODIFY] [state_test.go](file://pkg/state/state_test.go)

*   **Description**: `BaseBranch` フィールドのシリアライズ/デシリアライズテスト、後方互換性テストを追加
*   **Technical Design**:
    ```go
    func TestSave_Load_WithBaseBranch(t *testing.T)
    func TestStateFile_BaseBranch_OmitEmpty(t *testing.T)
    func TestStateFile_BackwardCompat_NoBaseBranch(t *testing.T)
    ```
*   **Logic**:
    *   `TestSave_Load_WithBaseBranch`: `BaseBranch: "main"` を含む `StateFile` を Save → Load し、値が保持されることを確認
    *   `TestStateFile_BaseBranch_OmitEmpty`: `BaseBranch` が空の場合、YAML 出力に `base_branch` キーが含まれないことを確認
    *   `TestStateFile_BackwardCompat_NoBaseBranch`: `BaseBranch` なしの既存ファイルを Load しても `BaseBranch` が空文字列で正常に読み込めることを確認（既存の `TestStateFile_BackwardCompat` の拡張）

#### [MODIFY] [state.go](file://pkg/state/state.go)

*   **Description**: `StateFile` 構造体に `BaseBranch` フィールドを追加
*   **Technical Design**:
    ```go
    type StateFile struct {
        Branch     string                  `yaml:"branch"`
        BaseBranch string                  `yaml:"base_branch,omitempty"` // 新規追加
        CreatedAt  time.Time               `yaml:"created_at"`
        Features   map[string]FeatureState `yaml:"features,omitempty"`
        CodeStatus *CodeStatus             `yaml:"code_status,omitempty"`
    }
    ```
*   **Logic**:
    *   `omitempty` タグにより、空文字列の場合は YAML 出力に含まれない（後方互換性を確保）

---

### action パッケージ

#### [NEW] [merge_test.go](file://pkg/action/merge_test.go)

*   **Description**: `Merge` アクションの単体テスト
*   **Technical Design**:
    既存の `close_test.go` と同じテストパターン（`testEnv` ヘルパー、`cmdexec.Recorder`、dry-run モード）を使用する。`testEnv` と `setupStateFile`、`hasRecordContaining` は `close_test.go` で定義済みなので、同一パッケージ `action_test` 内から直接参照可能。
    ```go
    func TestMerge_Success_FFOnly(t *testing.T)
    func TestMerge_Success_NoFF(t *testing.T)
    func TestMerge_Success_FF(t *testing.T)
    func TestMerge_BaseBranch_FromState(t *testing.T)
    func TestMerge_BaseBranch_Fallback_Main(t *testing.T)
    func TestMerge_UncommittedChanges_Error(t *testing.T)
    func TestMerge_DirtyRoot_Error(t *testing.T)
    func TestMerge_DryRun(t *testing.T)
    ```
*   **Logic**:
    *   `TestMerge_Success_FFOnly`: `MergeOptions{Strategy: "ff-only"}` で Merge を呼び、Recorder に `git merge --ff-only <branch>` が記録されることを確認
    *   `TestMerge_Success_NoFF`: `MergeOptions{Strategy: "no-ff"}` で Merge を呼び、`git merge --no-ff <branch>` が記録されることを確認
    *   `TestMerge_Success_FF`: `MergeOptions{Strategy: "ff"}` で Merge を呼び、`git merge <branch>` が記録されることを確認
    *   `TestMerge_BaseBranch_FromState`: StateFile に `BaseBranch: "develop"` を設定し、Merge が `develop` ブランチへのマージとして動作することを確認
    *   `TestMerge_BaseBranch_Fallback_Main`: StateFile に `BaseBranch` が空の場合、`main` にフォールバックされることを確認
    *   `TestMerge_UncommittedChanges_Error`: worktree に uncommitted changes がある場合、`ErrUncommittedChanges` エラーが返ることを確認（dry-run では git status をスキップするため、`CheckWorktreeClean` メソッドを直接テスト）
    *   `TestMerge_DirtyRoot_Error`: ルートリポジトリが dirty な場合、`ErrDirtyRoot` エラーが返ることを確認
    *   `TestMerge_DryRun`: dry-run モードで、git merge コマンドが Recorder に記録されるが実行されないことを確認

#### [NEW] [merge.go](file://pkg/action/merge.go)

*   **Description**: `Merge` アクションのビジネスロジック
*   **Technical Design**:
    ```go
    // MergeStrategy represents the git merge strategy option.
    type MergeStrategy string

    const (
        MergeStrategyFFOnly MergeStrategy = "ff-only" // default
        MergeStrategyNoFF   MergeStrategy = "no-ff"
        MergeStrategyFF     MergeStrategy = "ff"
    )

    // MergeOptions holds parameters for the merge action.
    type MergeOptions struct {
        Branch   string        // branch to merge
        RepoRoot string        // repo root (where BaseBranch is checked out)
        Strategy MergeStrategy // merge strategy
    }

    // MergeResult holds the result of a merge operation.
    type MergeResult struct {
        BaseBranch string // the branch merged into
        Strategy   MergeStrategy
        Success    bool
    }

    // Merge executes a local git merge.
    func (r *Runner) Merge(opts MergeOptions) (MergeResult, error)
    ```
*   **Logic**:
    1. `StateFile` を Load して `BaseBranch` を取得。空なら `"main"` にフォールバック
    2. worktree パス内で `git status --porcelain` を実行し、出力があれば `ErrUncommittedChanges` を返す
    3. ルートリポジトリ（`RepoRoot`）で `git status --porcelain` を実行し、出力があれば `ErrDirtyRoot` を返す
    4. `Strategy` に応じて git merge コマンドを構築:
        - `ff-only` → `git merge --ff-only <branch>`
        - `no-ff` → `git merge --no-ff <branch>`
        - `ff` → `git merge <branch>`
    5. ルートリポジトリの Dir オプション付きで `git merge` を実行
    6. エラー時はコンフリクト状況をログ出力して `error` を返す
    7. 成功時は `MergeResult` を返す

    ```go
    // エラー定義
    var (
        ErrUncommittedChanges = fmt.Errorf("uncommitted changes exist in worktree; please commit or stash first")
        ErrDirtyRoot          = fmt.Errorf("root repository has uncommitted changes; please clean up first")
    )
    ```

---

### cmd パッケージ

#### [NEW] [merge.go](file://features/tt/cmd/merge.go)

*   **Description**: cobra コマンド定義。フラグ解析と `action.Merge` への委譲
*   **Technical Design**:
    ```go
    var (
        mergeFlagNoFF bool
        mergeFlagFF   bool
    )

    var mergeCmd = &cobra.Command{
        Use:   "merge <branch>",
        Short: "Merge branch into its base branch locally",
        Long:  "Performs a local git merge of the specified branch into its base branch (recorded at tt open time).",
        Args:  cobra.ExactArgs(1),
        RunE:  runMerge,
    }

    func init() {
        mergeCmd.Flags().BoolVar(&mergeFlagNoFF, "no-ff", false, "Always create a merge commit")
        mergeCmd.Flags().BoolVar(&mergeFlagFF, "ff", false, "Use git default merge strategy (ff if possible)")
    }

    func runMerge(cmd *cobra.Command, args []string) error
    ```
*   **Logic**:
    1. `InitContext(args)` でコンテキスト初期化（branch を取得）
    2. マージ戦略の決定: `--no-ff` → `MergeStrategyNoFF`, `--ff` → `MergeStrategyFF`, それ以外 → `MergeStrategyFFOnly`
    3. `ctx.ActionRunner.Merge(action.MergeOptions{...})` を呼び出す
    4. 成功/失敗を `Report.Steps` に記録
    5. `--ff-only` 失敗時は「`--no-ff` または `--ff` を試してください」とヒントメッセージを表示

#### [MODIFY] [root.go](file://features/tt/cmd/root.go)

*   **Description**: `mergeCmd` をルートコマンドに登録
*   **Technical Design**:
    ```go
    func init() {
        // 既存の AddCommand の後に追加
        rootCmd.AddCommand(mergeCmd)
    }
    ```

#### [MODIFY] [open.go](file://features/tt/cmd/open.go)

*   **Description**: worktree 作成時に BaseBranch を StateFile に記録する処理を追加
*   **Technical Design**:
    worktree 作成前（Step 1 の直前）に、現在のHEADブランチを取得:
    ```go
    gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
    baseBranch, _ := ctx.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), gitCmd, "rev-parse", "--abbrev-ref", "HEAD")
    baseBranch = strings.TrimSpace(baseBranch)
    ```
    StateFile 保存時（Step 2 の state.Save 付近）に `sf.BaseBranch = baseBranch` を設定。
    コンテナなしで open した場合は CodeStatus 初期化と同時に BaseBranch も設定する必要があるので、Step 1 の直後にステート初期化ロジックを追加:
    ```go
    // Step 1 の worktree 作成直後に追加
    statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
    sf, _ := state.Load(statePath)
    if sf.Branch == "" {
        sf.Branch = ctx.Branch
        sf.CreatedAt = time.Now()
    }
    if sf.BaseBranch == "" {
        sf.BaseBranch = baseBranch
    }
    if sf.CodeStatus == nil {
        sf.CodeStatus = &state.CodeStatus{
            Status: state.CodeStatusLocal,
        }
    }
    _ = state.Save(statePath, sf)
    ```

---

## Step-by-Step Implementation Guide

### Phase 1: StateFile の BaseBranch フィールド追加（TDD）

- [x] 1. **state テスト作成**: `pkg/state/state_test.go` に `TestSave_Load_WithBaseBranch`, `TestStateFile_BaseBranch_OmitEmpty`, `TestStateFile_BackwardCompat_NoBaseBranch` の3テストを追加
- [x] 2. **state 実装**: `pkg/state/state.go` の `StateFile` 構造体に `BaseBranch` フィールドを追加
- [x] 3. **ビルド検証**: `./scripts/process/build.sh` を実行し、テストが全て通ることを確認

### Phase 2: Merge アクション（TDD）

- [x] 4. **merge テスト作成**: `pkg/action/merge_test.go` を新規作成。8テストケースを追加
- [x] 5. **merge 実装**: `pkg/action/merge.go` を新規作成。`MergeOptions`, `MergeResult`, `Merge()` メソッドを実装
- [x] 6. **ビルド検証**: `./scripts/process/build.sh` を実行

### Phase 3: コマンド統合

- [x] 7. **merge コマンド作成**: `features/tt/cmd/merge.go` を新規作成
- [x] 8. **root.go 修正**: `rootCmd.AddCommand(mergeCmd)` を追加
- [x] 9. **open.go 修正**: worktree 作成時に BaseBranch をステートに記録する処理を追加
- [x] 10. **最終ビルド検証**: `./scripts/process/build.sh` を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**:
        *   `pkg/state` のテスト: `BaseBranch` の roundtrip、omitempty、後方互換性
        *   `pkg/action` のテスト: Merge の各戦略、BaseBranch 自動解決、事前チェック、dry-run
        *   全体ビルドエラーなし

## Documentation

該当なし（`prompts/specifications` 配下に本機能に関連する既存ドキュメントはなし）。
