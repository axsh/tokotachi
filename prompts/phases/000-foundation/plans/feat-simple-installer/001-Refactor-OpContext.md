# 001-Refactor-OpContext

> **Source Specification**: [001-Refactor-OpContext.md](file://prompts/phases/000-foundation/ideas/feat-simple-installer/001-Refactor-OpContext.md)

## Goal Description

`tokotachi.go` のコンテキスト生成を `opContext` 構造体に統合し、各プリミティブメソッドを公開版 + 内部版（`doXxx`）に分割して、合成関数 `Open` が `doCreate → doUp → doEditor` を呼ぶだけのシンプルな構造にリファクタリングする。

## User Review Required

> [!IMPORTANT]
> `Scaffold` メソッドは `newContext()` すら使わず完全に独自のコンテキスト組み立てを行っているため、今回の `opContext` 統合対象外とします。

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `opContext` 構造体の導入 | Proposed Changes > tokotachi.go (opContext 追加) |
| 各メソッドの公開版 + 内部版分割 | Proposed Changes > tokotachi.go (doCreate, doUp, doDown, doEditor) |
| `Open` のリファクタリング | Proposed Changes > tokotachi.go (Open → doCreate + doUp + doEditor) |
| `Close` のコンテキスト統一 | Proposed Changes > tokotachi.go (Close で opContext 使用) |
| `SkipIfRunning` フラグ追加 | Proposed Changes > tokotachi.go (UpOptions) |
| 既存テスト + ビルドが通ること | Verification Plan |

## Proposed Changes

### tokotachi パッケージ

#### [MODIFY] [tokotachi.go](file://tokotachi.go)

*   **Description**: `opContext` 構造体と `newOpContext()` を導入し、7メソッドを公開版 + 内部版に分割。`Open` を合成関数にリファクタリング。

*   **Technical Design**:

    **1. `opContext` 構造体を追加（`newContext()` 直後に配置）**:
    ```go
    // opContext holds shared objects for a single operation.
    type opContext struct {
        logger       *log.Logger
        runner       *cmdexec.Runner
        actionRunner *action.Runner
        wm           *worktree.Manager
        projectName  string
    }

    // newOpContext builds a shared context for an operation.
    func (c *Client) newOpContext() *opContext {
        logger, runner, actionRunner := c.newContext()
        wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}
        return &opContext{
            logger:       logger,
            runner:       runner,
            actionRunner: actionRunner,
            wm:           wm,
            projectName:  "tt",
        }
    }
    ```

    **2. `UpOptions` に `SkipIfRunning` フィールド追加**:
    ```go
    type UpOptions struct {
        SSH           bool
        Rebuild       bool
        NoBuild       bool
        SkipIfRunning bool // If true, skip Up if container is already running.
    }
    ```

    **3. 各メソッドの分割パターン**:

    **`Create`** (L104-123):
    - 公開版: `opContext` 生成 → `doCreate()` 呼び出し
    - `doCreate(ctx *opContext, branch string, opts CreateOptions) error`:
      - 既存ロジックから `wm` 生成を除去し `ctx.wm` を使う
      - ロジック: `ctx.wm.Exists(branch)` でチェック → 存在すれば info ログで return nil → なければ `ctx.wm.Create(branch)`

    **`Up`** (L137-259):
    - 公開版: `opContext` 生成 → `doUp()` 呼び出し
    - `doUp(ctx *opContext, branch, feature string, opts UpOptions) error`:
      - 既存ロジックから `newContext()`, `runner`, `wm`, `projectName` 生成を除去し `ctx` のフィールドを使う
      - `SkipIfRunning` ロジック追加: `opts.SkipIfRunning` が true の場合、`ctx.actionRunner.Status(containerName, worktreePath)` が `action.StateContainerRunning` であれば early return nil
      - `containerName := resolve.ContainerName(ctx.projectName, feature)`
      - `imageName := resolve.ImageName(ctx.projectName, feature)`
      - worktree 解決: `resolve.Worktree(c.RepoRoot, branch)` → エラー時 DryRun なら `ctx.wm.Path(branch)` を使う
      - devcontainer.json 読み込み: `resolve.LoadDevcontainerConfig(c.RepoRoot, feature, branch)`
      - `action.UpOptions` の組み立て（現在の L188-216 と同一ロジック）
      - gitInfo 検出（現在の L218-226 と同一ロジック）
      - `ctx.actionRunner.Up(upOpts)`
      - state 保存（現在の L232-258 と同一ロジック）

    **`Down`** (L264-290):
    - 公開版: `opContext` 生成 → `doDown()` 呼び出し
    - `doDown(ctx *opContext, branch, feature string, opts DownOptions) error`:
      - `containerName := resolve.ContainerName(ctx.projectName, feature)`
      - `ctx.actionRunner.Down(containerName)`
      - state 更新（既存ロジック）

    **`Open`** (L298-451) → **合成関数にリファクタリング**:
    ```go
    func (c *Client) Open(branch, feature string, opts OpenOptions) error {
        if err := validateBranch(branch); err != nil {
            return err
        }
        ctx := c.newOpContext()

        // Step 1: Create worktree
        if err := c.doCreate(ctx, branch, CreateOptions{}); err != nil {
            return err
        }

        // Step 2: Up container (if feature specified)
        if feature != "" {
            if err := c.doUp(ctx, branch, feature, UpOptions{SkipIfRunning: true}); err != nil {
                return err
            }
        }

        // Step 3: Open editor
        return c.doEditor(ctx, branch, feature, opts.Editor)
    }
    ```

    **`doEditor`** — 新規内部メソッド（現在の `Open` L402-448 のロジックを抽出）:
    ```go
    func (c *Client) doEditor(ctx *opContext, branch, feature, editorFlag string) error {
        // worktree パス解決
        // editor launcher 作成
        // plan.Build で TryDevcontainerAttach 判定
        // containerName 解決
        // actionRunner.Open(launcher, launchOptions)
    }
    ```
    - ロジック:
      - `worktreePath := resolve.Worktree(c.RepoRoot, branch)` → エラー時 DryRun なら `ctx.wm.Path(branch)`
      - `editorName := detect.Editor(editorFlag)` → 空なら `detect.EditorCursor`
      - `containerMode := matrix.ContainerMode("docker-local")`
      - `p := plan.Build(plan.Input{...EditorOpen: true...})`
      - `launcher := editor.NewLauncher(editorName)`
      - `containerName := ""` → feature != "" なら `resolve.ContainerName(ctx.projectName, feature)`
      - `tryDevcontainer := p.TryDevcontainerAttach && feature != ""`
      - `ctx.actionRunner.Open(launcher, editor.LaunchOptions{...})`

    **`Close`** (L468-504):
    - 公開版: `opContext` 生成 → `doClose()` 呼び出し
    - `doClose(ctx *opContext, branch string, opts CloseOptions) error`:
      - `ctx.actionRunner.Close(action.CloseOptions{...}, ctx.wm)` — 既存ロジックそのまま
      - `stdin` 処理: `c.Stdin` が nil なら `os.Stdin`

    **`Status`** (L623-646):
    - 公開版: `opContext` 生成 → 内部ロジック
    - `status` は単純なためインラインのまま `ctx.wm` を使用（`doStatus` 分割は不要）

    **`List`** (L664-702):
    - 公開版: `opContext` 生成 → 内部ロジック
    - `list` も単純なため `ctx.runner` を使用してインラインのまま

*   **Logic**:
    - `newContext()` メソッドは `opContext` 内部で利用されるため残す（`Scaffold` が旧形式で使用）
    - `resolveProjectName()` は常に `"tt"` を返す（前回の devrc 削除で簡略化済み）ため、`opContext` では直接 `"tt"` をセット

## Step-by-Step Implementation Guide

1.  **`opContext` 構造体と `newOpContext()` を追加**:
    *   `tokotachi.go` の `newContext()` 直後（L82付近）に `opContext` 構造体と `newOpContext()` を追加

2.  **`UpOptions` に `SkipIfRunning` フィールド追加**:
    *   `tokotachi.go` L126-135 の `UpOptions` に `SkipIfRunning bool` を追加

3.  **`doCreate` を作成し `Create` をリファクタリング**:
    *   `doCreate(ctx *opContext, branch string, opts CreateOptions) error` を作成
    *   `Create` は `c.newOpContext()` → `c.doCreate()` に変更

4.  **`doUp` を作成し `Up` をリファクタリング**:
    *   `doUp(ctx *opContext, branch, feature string, opts UpOptions) error` を作成
    *   先頭に `SkipIfRunning` チェックロジックを追加
    *   `Up` は `c.newOpContext()` → `c.doUp()` に変更

5.  **`doDown` を作成し `Down` をリファクタリング**:
    *   `doDown(ctx *opContext, branch, feature string, opts DownOptions) error` を作成
    *   `Down` は `c.newOpContext()` → `c.doDown()` に変更

6.  **`doEditor` を新規作成**:
    *   現在の `Open` L402-448 のエディタ起動ロジックを `doEditor(ctx *opContext, branch, feature, editorFlag string) error` に抽出

7.  **`doClose` を作成し `Close` をリファクタリング**:
    *   `doClose(ctx *opContext, branch string, opts CloseOptions) error` を作成
    *   `Close` は `c.newOpContext()` → `c.doClose()` に変更

8.  **`Open` を合成関数にリファクタリング**:
    *   既存の L298-451 を削除し、`doCreate → doUp(SkipIfRunning: true) → doEditor` の呼び出しに置換

9.  **`Status` と `List` を `opContext` に対応**:
    *   `Status`: `c.newOpContext()` を使い `ctx.wm` に置換
    *   `List`: `c.newOpContext()` を使い `ctx.runner` に置換

10. **ビルド・テスト実行**:
    *   `./scripts/process/build.sh` でビルド + 単体テスト

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

## Documentation

影響を受ける既存のドキュメントはなし。本リファクタリングは内部構造の変更であり、公開APIのシグネチャは変更されない（`SkipIfRunning` フィールド追加は後方互換）。
