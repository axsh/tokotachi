# 000-CLI-Argument-Reorder-Part2

> **Source Specification**: `prompts/phases/000-foundation/ideas/feat-remove-feature-name/000-CLI-Argument-Reorder.md`

## Goal Description

Part2では**CLIコマンド層**（`cmd/*.go`）のUse文字列変更、feature有無による分岐追加、およびR9（`open --up`オプション）を実装する。

## User Review Required

> [!IMPORTANT]
> - `down`/`shell`/`exec`はfeature未指定時にエラーを返す方針で実装します。
> - `list`はブランチベースに変更し、`work/<branch>/features/`配下のfeature一覧を表示します。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: CLI引数順序の変更 | Proposed Changes > cmd/*.go (全サブコマンドのUse変更) |
| R2: feature省略時の動作 | Proposed Changes > cmd/up.go, cmd/open.go, cmd/close.go 等 |
| R3: feature指定時の既存動作維持 | Proposed Changes > cmd/*.go (HasFeature分岐) |
| R4: 各コマンドのfeature省略時の具体的動作 | Proposed Changes > cmd/*.go |
| R9: open --up オプション | Proposed Changes > cmd/open.go |

## Proposed Changes

### cmd パッケージ (サブコマンド)

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/up.go)

*   **Description**: Use文字列変更、feature省略時にcontainer起動をスキップ
*   **Technical Design**:
    *   `Use`: `"up <feature> [branch]"` → `"up <branch> [feature]"`
    *   `Args`: `cobra.RangeArgs(1, 2)` 変更なし
    *   feature省略時の処理フロー:
        1. worktree作成は実行
        2. `ctx.HasFeature()` チェック
        3. feature無し → container関連処理（`containerName`解決、devcontainer読込、`ActionRunner.Up()`、state保存）をスキップ
        4. エディタ起動は`--editor`指定時にworktreeパスに対して実行（container attachなし）
*   **Logic**:
    ```go
    func runUp(cmd *cobra.Command, args []string) error {
        ctx, err := InitContext(args)
        // ...

        // Resolve worktree path
        wm := &worktree.Manager{...}
        if !wm.Exists(ctx.Feature, ctx.Branch) {
            // worktree作成 (feature有無に関わらず実行)
            wm.Create(ctx.Feature, ctx.Branch)
        }

        if ctx.HasFeature() {
            // === feature指定あり: 既存のcontainer起動フロー ===
            containerName := resolve.ContainerName(projectName, ctx.Feature)
            imageName := resolve.ImageName(projectName, ctx.Feature)
            // devcontainer読込、UpOptions構築、ActionRunner.Up()、state保存
            // ...
        }

        // エディタ起動 (feature有無に関わらず--editor指定時に実行)
        if p.ShouldOpenEditor {
            // feature無しの場合: TryDevcontainer=false
            // ...
        }
    }
    ```

#### [MODIFY] [down.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/down.go)

*   **Description**: Use変更、feature未指定時にエラー
*   **Technical Design**:
    *   `Use`: `"down <feature> [branch]"` → `"down <branch> [feature]"`
    *   feature無し → `return fmt.Errorf("feature is required for 'down' command (container operation)")`

#### [MODIFY] [close.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/close.go)

*   **Description**: Use変更、feature省略時はcontainer停止をスキップしworktree/ブランチ削除のみ実行
*   **Technical Design**:
    *   `Use`: `"close <feature> [branch]"` → `"close <branch> [feature]"`
    *   feature無し → `CloseOptions.ContainerName` を空文字列で渡す
    *   `action.Close` 内で `ContainerName` が空の場合はcontainer停止をスキップ

#### [MODIFY] [close.go (action)](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/action/close.go)

*   **Description**: `ContainerName`空文字時のcontainerステップスキップ
*   **Technical Design**:
    ```go
    func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error {
        // Step 1: Down container if running (skip if no container name)
        if opts.ContainerName != "" {
            containerState := r.Status(opts.ContainerName, wm.Path(opts.Feature, opts.Branch))
            if containerState == StateContainerRunning || containerState == StateContainerStopped {
                r.Logger.Info("Stopping container before close...")
                if err := r.Down(opts.ContainerName); err != nil {
                    r.Logger.Warn("Container down failed: %v", err)
                }
            }
        }
        // Step 2-4: worktree/branch/state は変更なし
    }
    ```

#### [MODIFY] [open.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/open.go)

*   **Description**: Use変更、feature省略時のcontainer attachスキップ、`--up`フラグ追加
*   **Technical Design**:
    *   `Use`: `"open <feature> [branch]"` → `"open <branch> [feature]"`
    *   新フラグ: `openFlagUp bool` (`--up`)
    *   `--up`指定時の処理フロー:
        1. container状態を確認
        2. containerが起動していない場合、`runUp`相当の処理を実行
        3. エディタを開く
*   **Logic**:
    ```go
    var openFlagUp bool

    func init() {
        openCmd.Flags().BoolVar(&openFlagUp, "up", false,
            "Start the container if not running before opening editor")
        // ...
    }

    func runOpen(cmd *cobra.Command, args []string) error {
        ctx, err := InitContext(args)
        // ...

        worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)

        // --up フラグ処理
        if openFlagUp {
            wm := &worktree.Manager{...}
            // worktreeが無ければ作成
            if !wm.Exists(ctx.Feature, ctx.Branch) {
                wm.Create(ctx.Feature, ctx.Branch)
                // worktreePath を再解決
                worktreePath, err = resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
            }

            if ctx.HasFeature() {
                // containerが起動していなければup相当の処理
                containerName := resolve.ContainerName(projectName, ctx.Feature)
                state := ctx.ActionRunner.Status(containerName, worktreePath)
                if state != action.StateContainerRunning {
                    // container起動処理 (up.goと同等のUpOptions構築 → ActionRunner.Up())
                    // ...
                }
            }
        }

        // エディタ起動 (既存ロジック)
        // feature無しの場合: TryDevcontainer=false
        // ...
    }
    ```

#### [MODIFY] [shell.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/shell.go)

*   **Description**: Use変更、feature未指定時にエラー
*   **Technical Design**:
    *   `Use`: `"shell <feature> [branch]"` → `"shell <branch> [feature]"`
    *   feature無し → `return fmt.Errorf("feature is required for 'shell' command (container operation)")`

#### [MODIFY] [exec_cmd.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/exec_cmd.go)

*   **Description**: Use変更、引数解析変更、feature未指定時にエラー
*   **Technical Design**:
    *   `Use`: `"exec <feature> [branch] -- <command...>"` → `"exec <branch> [feature] -- <command...>"`
    *   引数解析: `beforeDash` の解釈を `args[0]=branch`, `args[1]=feature` に変更
    *   feature無し → `return fmt.Errorf("feature is required for 'exec' command (container operation)")`

#### [MODIFY] [pr.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/pr.go)

*   **Description**: Use変更。PRはcontainer不要なので特別な分岐不要
*   **Technical Design**:
    *   `Use`: `"pr <feature> [branch]"` → `"pr <branch> [feature]"`
    *   `InitContext` の変更で自動的に対応される

#### [MODIFY] [status.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/status.go)

*   **Description**: Use変更、feature省略時はcontainer状態スキップ
*   **Technical Design**:
    *   `Use`: `"status <feature> [branch]"` → `"status <branch> [feature]"`
    *   feature無し → `containerName` を空文字列にし、worktree状態のみ表示

#### [MODIFY] [list.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/list.go)

*   **Description**: Use変更、引数をbranchに変更、`wm.List(branch)`で`work/<branch>/features/`配下のfeature一覧表示
*   **Technical Design**:
    *   `Use`: `"list <feature>"` → `"list <branch>"`
    *   引数からbranchを取得: `branch := args[0]`
    *   `wm.List(branch)` → `work/<branch>/features/`配下をスキャン
    *   テーブルヘッダーを "Branch" → "Feature" 等に変更

---

### action パッケージ

#### [MODIFY] [status.go (action)](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/action/status.go)

*   **Description**: `PrintStatus`のfeature空対応
*   **Technical Design**:
    *   `feature`が空の場合、containerステータス表示をスキップし、worktree情報のみ表示

## Step-by-Step Implementation Guide

> **前提**: Part1が完了していること。

1.  **cmd/up.go: Use変更とfeature分岐追加**
    *   `Use`文字列を`"up <branch> [feature]"`に変更
    *   `ctx.HasFeature()`チェックを追加し、feature無し時にcontainer関連をスキップ

2.  **cmd/down.go: Use変更とfeatureチェック**
    *   `Use`文字列変更
    *   `ctx.HasFeature()`チェックを追加し、feature無し時にエラー

3.  **cmd/close.go + action/close.go: Use変更とcontainerスキップ**
    *   `Use`文字列変更
    *   `ContainerName`空文字時にcontainerステップスキップ

4.  **cmd/open.go: Use変更、feature分岐、--upフラグ追加**
    *   `Use`文字列変更
    *   `--up`フラグ追加
    *   `--up`指定時のcontainer自動起動ロジック実装
    *   feature無し時のcontainer attachスキップ

5.  **cmd/shell.go: Use変更とfeatureチェック**
    *   `Use`文字列変更
    *   feature無し時エラー

6.  **cmd/exec_cmd.go: Use変更と引数解析変更**
    *   `Use`文字列変更
    *   引数解析を `branch, feature` 順に変更
    *   feature無し時エラー

7.  **cmd/pr.go, cmd/status.go: Use変更**
    *   `Use`文字列変更
    *   status: feature無し時のcontainer表示スキップ

8.  **cmd/list.go: ブランチベースに変更**
    *   `Use`文字列変更
    *   引数解釈をbranchに変更
    *   `wm.List(branch)` 呼び出しに変更

9.  **action/status.go: feature空対応**
    *   `PrintStatus`のfeature空対応

10. **ビルド検証**
    *   `./scripts/process/build.sh` を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: 全テストがパスすること、ビルドエラーがないこと

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh
    ```
    *   **Log Verification**: 全統合テストがパスすること

## Documentation

#### [MODIFY] [000-CLI-Argument-Reorder.md](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/prompts/phases/000-foundation/ideas/feat-remove-feature-name/000-CLI-Argument-Reorder.md)
*   **更新内容**: R4の未定事項（`down`/`shell`/`exec`のfeature省略時動作=エラー）を確定値に更新
