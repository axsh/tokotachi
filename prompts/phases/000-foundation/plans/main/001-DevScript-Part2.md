# 001-DevScript-Part2

> **Source Specification**: [000-DevScript.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/000-DevScript.md)

## Goal Description

devctl Part 2 として、エディタ起動（`--open`）、SSH モード詳細、devcontainer.json 解釈、実行計画（planner）、`cmd/root.go` アクション dispatch 統合、build.sh 統合を実装する。

Part 1 で構築した基盤（detect, matrix, resolve, action, cmd 骨格）の上に、以下を積み上げる：
- エディタ起動ロジック（VSCode, Cursor, Antigravity, Claude Code）
- エディタコマンドの環境変数優先解決
- devcontainer.json 最小サブセット解釈
- 実行計画構築（detect → resolve → plan → execute → fallback）
- `cmd/root.go` に完全なアクション dispatch を統合
- `build.sh` に devctl ビルドステップ追加

## User Review Required

> [!IMPORTANT]
> 統合テストは Docker 環境が必須です。本計画ではユニットテストレベルの検証を優先し、E2E 統合テストの骨格は将来対応とします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: feature 単位の開発環境管理 | **Part 1 で完了** |
| R2: コンテナライフサイクル管理 | **Part 1 で完了** + `cmd/root.go` dispatch 統合 |
| R3: エディタ/エージェント起動 (`--open`, `--editor`) | `internal/editor/` パッケージ + `internal/action/open.go` |
| R4: エディタ別の接続方式 | `internal/editor/vscode.go`, `cursor.go`, `ag.go`, `claude.go` |
| R5: コンテナ内操作 (`--shell`, `--exec`) | **Part 1 で完了** + `cmd/root.go` dispatch 統合 |
| R6: SSH モード | `internal/action/up.go` の SSH オプション強化 |
| R7: マトリクス駆動の分岐制御 | `internal/plan/planner.go` による統合 |
| devcontainer.json 解釈 | `internal/resolve/devcontainer.go` |
| build/run 設定の解決優先順位 | `internal/resolve/devcontainer.go` |
| 実装アーキテクチャ（detect→resolve→plan→execute→fallback） | `internal/plan/planner.go` + `cmd/root.go` |
| 検証シナリオ 2-5, 9-11 | ユニットテスト（DryRun モード） |
| build.sh 統合 | `scripts/process/build.sh` に devctl ステップ追加 |

## Proposed Changes

### devcontainer 解釈 (`internal/resolve/`)

#### [NEW] [devcontainer.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/devcontainer.go)
*   **Description**: devcontainer.json の最小サブセット解釈
*   **Technical Design**:
    *   `DevcontainerConfig` 構造体: `Image string`, `Build DevcontainerBuild`, `WorkspaceFolder string`, `ContainerEnv map[string]string`
    *   `DevcontainerBuild` 構造体: `Dockerfile string`, `Context string`
    *   `IsEmpty() bool` — 設定不在判定
    *   `HasDockerfile() bool` — Dockerfile ベースビルドかどうか
    *   `LoadDevcontainerConfig(repoRoot, feature string) (DevcontainerConfig, error)`
        *   検索優先順位（仕様の「build/run 設定の解決優先順位」に従う）:
            1. `work/<feature>/.devcontainer/devcontainer.json` → JSON パース
            2. `work/<feature>/.devcontainer/Dockerfile` → `Build.Dockerfile` に設定
            3. `work/<feature>/Dockerfile` → `Build.Dockerfile` に設定
        *   いずれも存在しなければ空の `DevcontainerConfig` を返す

#### [NEW] [devcontainer_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/devcontainer_test.go)
*   **Description**: devcontainer.json 解釈のテスト
*   **テストケース**:
    *   JSON ファイルからの読み込み（`build`, `workspaceFolder`, `containerEnv` の各フィールド）
    *   `image` フィールドのみの場合
    *   ファイル不在時に `IsEmpty() == true`
    *   優先順位テスト（`devcontainer.json` が存在すれば `Dockerfile` は無視される）

---

### エディタパッケージ (`internal/editor/`)

#### コマンド解決の環境変数対応

各エディタのコマンド名は、環境変数から優先的に解決する。環境変数が未設定の場合はデフォルト定数にフォールバックする。

| Editor | 環境変数 | デフォルト |
|---|---|---|
| VSCode | `DEVCTL_CMD_CODE` | `code` |
| Cursor | `DEVCTL_CMD_CURSOR` | `cursor` |
| Antigravity | `DEVCTL_CMD_AG` | `antigravity` |
| Claude Code | `DEVCTL_CMD_CLAUDE` | `claude` |

#### [NEW] [editor.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/editor.go)
*   **Description**: エディタ起動のインターフェースと共通型
*   **Technical Design**:
    *   `LaunchOptions` 構造体: `WorktreePath`, `ContainerName`, `NewWindow bool`, `TryDevcontainer bool`, `Logger`, `DryRun bool`
    *   `LaunchResult` 構造体: `Method string` (`"devcontainer"`, `"local"`, `"cli"`), `Fallback bool`, `EditorCmd string`
    *   `Launcher` インターフェース:
        *   `Launch(opts LaunchOptions) (LaunchResult, error)`
        *   `Name() string`
    *   `ResolveCommand(envKey, defaultCmd string) string` — 環境変数から優先取得、なければデフォルト

#### [NEW] [vscode.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/vscode.go)
*   **Description**: VSCode 起動ロジック
*   **挙動**:
    1. `ResolveCommand("DEVCTL_CMD_CODE", "code")` でコマンド名を決定
    2. `TryDevcontainer && ContainerName != ""` の場合:
        *   `vscode-remote://attached-container+<container>/workspace` URI で `--folder-uri` 起動を試行
        *   失敗時はローカルフォールバック
    3. ローカル起動: `<cmd> [--new-window] <worktree>`
    4. DryRun 時はログ出力のみ

#### [NEW] [cursor.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/cursor.go)
*   **Description**: Cursor 起動ロジック
*   **挙動**: VSCode と同構造。コマンドを `ResolveCommand("DEVCTL_CMD_CURSOR", "cursor")` で解決する点のみ異なる

#### [NEW] [ag.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/ag.go)
*   **Description**: Antigravity 起動ロジック
*   **挙動**:
    1. `ResolveCommand("DEVCTL_CMD_AG", "antigravity")` でコマンド名を決定
    2. 常に `<cmd> <worktree>` でローカル起動
    3. Dev Container attach は **試行しない**（仕様: L4 非対応）

#### [NEW] [claude.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/claude.go)
*   **Description**: Claude Code 起動ロジック
*   **挙動**:
    1. `ResolveCommand("DEVCTL_CMD_CLAUDE", "claude")` でコマンド名を決定
    2. `cmd.Dir = worktreePath` で cwd を設定し、`cmd.Start()` で起動（バックグラウンド）
    3. Dev Container attach は非対応

#### [NEW] [factory.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/factory.go)
*   **Description**: `detect.Editor` から `Launcher` インスタンスを生成
*   **挙動**: `switch` で `EditorVSCode` → `&VSCode{}` 等を返す

#### [NEW] [editor_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/editor/editor_test.go)
*   **テストケース**:
    *   `TestNewLauncher`: 4 エディタすべてで正しい `Launcher` が返ること
    *   `TestNewLauncher_Invalid`: 未知エディタでエラー
    *   `TestLauncher_DryRun`: 4 エディタすべてで DryRun 時にエラーなく `LaunchResult` が返り、ログに `[DRY-RUN]` を含むこと
    *   `TestResolveCommand`: 環境変数設定時にそちらを優先、未設定時にデフォルト値を返すこと

---

### 実行計画パッケージ (`internal/plan/`)

#### [NEW] [planner.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/plan/planner.go)
*   **Description**: detect→resolve→plan フェーズの統合ロジック
*   **Technical Design**:
    *   `Input` 構造体: `Feature`, `OS`, `Editor`, `ContainerMode`, `Up`, `Open`, `Down`, `Status`, `Shell`, `Exec []string`, `SSH`, `Rebuild`, `NoBuild`
    *   `Plan` 構造体: `ShouldStartContainer`, `ShouldStopContainer`, `ShouldOpenEditor`, `ShouldShowStatus`, `ShouldOpenShell`, `ExecCommand []string`, `TryDevcontainerAttach`, `SSHMode`, `Rebuild`, `NoBuild`, `CompatLevel`
    *   `Build(input Input) Plan` 関数:
        1. `matrix.ResolveCapability(input.OS, input.Editor)` で Capability 取得
        2. 各フラグ（Up/Down/Status/Shell/Exec）を Plan にマッピング
        3. `input.Open` の場合:
            *   `cap.CanTryDevcontainerAttach && (mode == devcontainer || docker-local)` → `TryDevcontainerAttach = true`
            *   それ以外 → ローカル起動のみ
        4. `input.SSH` → `SSHMode = true`

#### [NEW] [planner_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/plan/planner_test.go)
*   **テストケース**:
    *   `TestBuildPlan_UpOnly`: Up のみの場合、`ShouldStartContainer=true`, `ShouldOpenEditor=false`
    *   `TestBuildPlan_OpenWithDevcontainer`: Cursor+devcontainer → `TryDevcontainerAttach=true`
    *   `TestBuildPlan_OpenAG_NoDevcontainer`: AG → `TryDevcontainerAttach=false`
    *   `TestBuildPlan_UpAndOpen`: 両方 true の場合
    *   `TestBuildPlan_Down`: `ShouldStopContainer=true`
    *   `TestBuildPlan_SSH`: `SSHMode=true`

---

### アクション統合 (`internal/action/`)

#### [NEW] [open.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/open.go)
*   **Description**: `--open` アクションの editor パッケージへの委譲
*   **挙動**:
    1. `editor.Launcher` と `editor.LaunchOptions` を受け取る
    2. `launcher.Launch(opts)` を実行
    3. `result.Fallback == true` の場合は `[WARN]` でログ出力
    4. 結果を返す

---

### CLI dispatch 統合 (`cmd/`)

#### [MODIFY] [root.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/root.go)
*   **Description**: Part 1 のプレースホルダー（行 87-98）を完全なアクション dispatch に置き換え
*   **挙動**:
    1. `resolve.Worktree(repoRoot, feature)` で worktree パスを解決
    2. `resolve.ContainerName(projectName, feature)` / `resolve.ImageName(...)` でコンテナ識別子を解決
    3. `matrix.ContainerMode(globalCfg.DefaultContainerMode)` でコンテナモードを型変換
    4. `plan.Build(input)` で実行計画を構築
    5. `[DEBUG]` レベルで decision log 出力: `OS`, `Editor`, `ContainerMode`, `CompatLevel`
    6. 計画に基づいて順序実行:
        *   `plan.ShouldStartContainer` → `runner.Up(opts)`
        *   `plan.ShouldOpenEditor` → `editor.NewLauncher(ed)` → `runner.Open(launcher, launchOpts)`
        *   `plan.ShouldStopContainer` → `runner.Down(containerName)`
        *   `plan.ShouldShowStatus` → `runner.PrintStatus(...)`
        *   `plan.ShouldOpenShell` → `runner.Shell(containerName)`
        *   `plan.ExecCommand` → `runner.Exec(containerName, cmd)`
    7. 実行順序: up → open → (stop/shell/exec/status は排他的)

---

### build.sh 統合

#### [MODIFY] [build.sh](file:///c:/Users/yamya/myprog/escape/scripts/process/build.sh)
*   **Description**: devctl のビルドステップを追加
*   **挙動**:
    *   `build_devctl()` 関数を追加
    *   `features/devctl/go.mod` が存在する場合のみ実行
    *   `cd features/devctl && go build -o ../../bin/devctl .` でバイナリ生成
    *   `go test -v -count=1 ./...` でユニットテスト実行
    *   `main()` から `build_backend` の後に `build_devctl` を呼び出し

---

## Step-by-Step Implementation Guide

1.  **devcontainer 解釈の実装** (TDD):
    *   `internal/resolve/devcontainer_test.go` を作成（テスト先行）
    *   `internal/resolve/devcontainer.go` を実装

2.  **エディタパッケージの実装** (TDD):
    *   `internal/editor/editor.go` を作成（インターフェース、`ResolveCommand` 関数）
    *   `internal/editor/editor_test.go` を作成（ファクトリ + DryRun + 環境変数テスト）
    *   `internal/editor/vscode.go`, `cursor.go`, `ag.go`, `claude.go`, `factory.go` を実装

3.  **実行計画パッケージの実装** (TDD):
    *   `internal/plan/planner_test.go` を作成
    *   `internal/plan/planner.go` を実装

4.  **アクション open の実装**:
    *   `internal/action/open.go` を作成

5.  **CLI dispatch 統合**:
    *   `cmd/root.go` のプレースホルダーを完全な dispatch に置き換え

6.  **build.sh 統合**:
    *   `scripts/process/build.sh` に `build_devctl()` 関数を追加

7.  **依存関係解決・ビルド確認**:
    *   `go mod tidy` → `./scripts/process/build.sh`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **検証内容**:
        *   全パッケージのビルド成功
        *   `internal/resolve/` テスト: devcontainer.json 解釈と優先順位
        *   `internal/editor/` テスト: ファクトリ、DryRun、環境変数解決
        *   `internal/plan/` テスト: 各アクション組み合わせの計画構築
        *   `bin/devctl` バイナリが生成されること

## Documentation

なし（新規プロジェクトのため既存ドキュメント更新不要）
