# 004-DevcontainerNativeInterpret

> **Source Specification**: [002-DevcontainerNativeInterpret.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/002-DevcontainerNativeInterpret.md)

## Goal Description

devctl が `devcontainer.json` を自前で解釈し、`docker build` / `docker run` の引数に自動変換する。Node.js / `devcontainer` CLI 不要で DevContainer 互換動作を実現する。

## User Review Required

> [!IMPORTANT]
> - `LoadDevcontainerConfig` のシグネチャを `(repoRoot, feature string)` → `(repoRoot, feature, branch string)` に変更する **破壊的変更**
> - `UpOptions` に `DevcontainerConfig` フィールドを追加。`buildImage` / `Up` の docker 引数生成ロジックが全面変更

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
|:---|:---|
| R1: devcontainer.json の検索と読み込み | `resolve/devcontainer.go` LoadDevcontainerConfig |
| R2: サポートフィールド | `resolve/devcontainer.go` DevcontainerConfig 構造体拡張 |
| R3: docker build への反映 | `action/up.go` buildImage |
| R4: docker run への反映 | `action/up.go` Up |
| R5: devcontainer.json なしのフォールバック | `resolve/devcontainer.go` + `action/up.go` |
| R6: 既存構造体の拡張 | `resolve/devcontainer.go` DevcontainerConfig |
| R7: LoadDevcontainerConfig 検索パス更新 | `resolve/devcontainer.go` |
| R8: customizations | **対象外（将来）** |

## Proposed Changes

### resolve パッケージ

#### [MODIFY] [devcontainer_test.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/devcontainer_test.go)
*   **Description**: テストを 3 引数対応に更新、新フィールドのテストケース追加
*   **テストケース追加**:
    -   `TestLoadDevcontainerConfig_WithMountsAndUser`: `mounts`, `remoteUser`, `name` のパースを検証
    -   `TestLoadDevcontainerConfig_FeatureDir`: `features/<feature>/` からの検索を検証
    -   `TestLoadDevcontainerConfig_FeatureDirPriority`: features ディレクトリが worktree より優先されることを検証
*   **既存テスト修正**: 全ての `LoadDevcontainerConfig(root, feature)` 呼び出しを `LoadDevcontainerConfig(root, feature, branch)` に更新

#### [MODIFY] [devcontainer.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/resolve/devcontainer.go)
*   **Description**: 構造体にフィールド追加、検索パス拡張
*   **Technical Design**:
    ```go
    type DevcontainerConfig struct {
        Name            string            `json:"name"`
        Image           string            `json:"image"`
        Build           DevcontainerBuild `json:"build"`
        WorkspaceFolder string            `json:"workspaceFolder"`
        ContainerEnv    map[string]string `json:"containerEnv"`
        Mounts          []string          `json:"mounts"`
        RemoteUser      string            `json:"remoteUser"`
    }

    // ConfigDir returns the directory containing devcontainer.json.
    // Used to resolve relative paths in build.dockerfile and build.context.
    func (c DevcontainerConfig) ConfigDir() string

    func LoadDevcontainerConfig(repoRoot, feature, branch string) (DevcontainerConfig, error)
    ```
*   **Logic**:
    1. 検索順序:
       - `features/<feature>/.devcontainer/devcontainer.json`
       - `work/<feature>/<branch>/.devcontainer/devcontainer.json`
       - `work/<feature>/.devcontainer/devcontainer.json`（後方互換）
    2. JSON パース時に `Mounts`, `RemoteUser`, `Name` も読み込む
    3. 見つかった `devcontainer.json` の親ディレクトリパスを `ConfigDir` として保持する

    `DevcontainerConfig` に `configDir string`（unexported）フィールドを追加し、`ConfigDir()` メソッドで返す。`build.dockerfile` の相対パス解決に使用する。

### action パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/up.go)
*   **Description**: `UpOptions` と `Up()`/`buildImage()` を `DevcontainerConfig` から動的に引数生成するよう修正
*   **Technical Design**:
    ```go
    type UpOptions struct {
        ContainerName string
        ImageName     string
        WorktreePath  string
        FeaturePath   string
        Rebuild       bool
        NoBuild       bool
        SSHMode       bool
        Env           map[string]string
        // DevcontainerConfig fields
        WorkspaceFolder string   // from devcontainer.json (default: "/workspace")
        Mounts          []string // from devcontainer.json
        ContainerEnv    map[string]string // from devcontainer.json
        RemoteUser      string   // from devcontainer.json
        DockerfilePath  string   // resolved absolute path to Dockerfile
        BuildContext    string   // resolved absolute path to build context
    }
    ```
*   **Logic for `buildImage()`**:
    1. `opts.DockerfilePath` が指定されていれば `docker build -f <DockerfilePath> <BuildContext>` を実行
    2. `opts.BuildContext` が指定されていれば、それをビルドコンテキストとして使用
    3. どちらもなければ、従来の `.devcontainer/Dockerfile` 自動検出フォールバック
*   **Logic for `Up()` (docker run 引数生成)**:
    1. `-w` に `opts.WorkspaceFolder` を使用（デフォルト `/workspace`）
    2. `-v` に `opts.WorktreePath + ":" + opts.WorkspaceFolder` を使用
    3. `opts.Mounts` の各エントリを `--mount` フラグとして追加
    4. `opts.ContainerEnv` の各エントリを `-e KEY=VALUE` として追加
    5. `opts.RemoteUser` が指定されていれば `--user` フラグを追加
    6. `opts.Env`（CLI フラグからの環境変数）も引き続き追加
    7. SSH モードの `ENABLE_SSH=1` も引き続き追加

### cmd パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/cmd/up.go)
*   **Description**: `LoadDevcontainerConfig` を呼び出し、結果を `UpOptions` に渡す
*   **Logic**:
    1. `resolve.LoadDevcontainerConfig(ctx.RepoRoot, ctx.Feature, ctx.Branch)` を呼び出す
    2. `DevcontainerConfig` から以下を解決:
       - `DockerfilePath`: `configDir + build.dockerfile` の相対パス解決
       - `BuildContext`: `configDir + build.context` の相対パス解決（未指定時は configDir）
       - `WorkspaceFolder`: `cfg.WorkspaceFolder`（デフォルト `/workspace`）
       - `Mounts`: `cfg.Mounts`
       - `ContainerEnv`: `cfg.ContainerEnv`
       - `RemoteUser`: `cfg.RemoteUser`
    3. `cfg.Image` が指定されていて `cfg.Build` が空の場合、`ImageName` を `cfg.Image` に設定し `NoBuild = true`
    4. これらを `UpOptions` に設定して `action.Runner.Up()` を呼び出す

## Step-by-Step Implementation Guide

### Phase 1: resolve.DevcontainerConfig 拡張 (TDD)

- [x] 1. `internal/resolve/devcontainer_test.go` を更新
    - 既存 5 テスト: `LoadDevcontainerConfig` を 3 引数に変更（branch 引数追加）
    - 新規テスト 3 件追加: `TestLoadDevcontainerConfig_WithMountsAndUser`, `TestLoadDevcontainerConfig_FeatureDir`, `TestLoadDevcontainerConfig_FeatureDirPriority`
- [x] 2. `internal/resolve/devcontainer.go` を更新
    - `Mounts []string`, `RemoteUser string`, `Name string`, `configDir string` フィールド追加
    - `ConfigDir()` メソッド追加
    - `LoadDevcontainerConfig` を 3 引数に変更、検索パスに `features/<feature>/` 追加
- [x] 3. テスト実行 → 全 PASS 確認

### Phase 2: action.Up の DevcontainerConfig 対応

- [x] 4. `internal/action/up.go` の `UpOptions` を拡張（WorkspaceFolder, Mounts, ContainerEnv, RemoteUser, DockerfilePath, BuildContext）
- [x] 5. `buildImage()` を `DockerfilePath`/`BuildContext` ベースに修正
- [x] 6. `Up()` の docker run 引数生成を `UpOptions` のフィールドから動的に構築するよう修正

### Phase 3: cmd/up.go の統合

- [x] 7. `cmd/up.go` で `LoadDevcontainerConfig` を呼び出し、パス解決して `UpOptions` を構築
- [x] 8. ビルド → 全テスト PASS 確認
- [x] 9. `devctl up devctl test-001 --dry-run --verbose` で動作確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **手動動作確認（dry-run）**:
    ```bash
    # devcontainer.json が解釈されていることを確認
    ./bin/devctl up devctl test-001 --dry-run --verbose
    # 期待される出力:
    #   [DRY-RUN] docker build -f <../.devcontainer/Dockerfile> <context>
    #   [DRY-RUN] docker run ... --mount source=/var/run/docker.sock,... -w /workspace --user root ...
    ```

## Documentation

#### [MODIFY] [README.md](file:///c:/Users/yamya/myprog/escape/features/devctl/README.md)
*   **更新内容**: devcontainer.json 解釈機能の説明追加
