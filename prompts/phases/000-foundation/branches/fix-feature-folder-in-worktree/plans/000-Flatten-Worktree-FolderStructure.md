# 000-Flatten-Worktree-FolderStructure

> **Source Specification**: [000-Flatten-Worktree-FolderStructure.md](file://prompts/phases/000-foundation/ideas/fix-feature-folder-in-worktree/000-Flatten-Worktree-FolderStructure.md)

## Goal Description

worktreeフォルダ構造を簡略化し、`work/<branch>/features/<feature>/`と`work/<branch>/all/`の2系統を廃止して`work/<branch>/`に統一する。あわせてstateファイルをブランチ単位の`work/<branch>.state.yaml`に統合し、feature毎の接続情報（`connectivity`）を構造化して記録する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: worktreeフォルダ構造の統一 | Proposed Changes > worktree/worktree.go, resolve/worktree.go |
| R2: stateファイルのブランチ単位統合 | Proposed Changes > state/state.go |
| R3: git worktree作成・削除ロジック変更 | Proposed Changes > worktree/worktree.go, cmd/up.go |
| R4: closeコマンドの動作変更 | Proposed Changes > action/close.go, cmd/close.go |
| R5: devcontainer探索パス変更 | Proposed Changes > resolve/devcontainer.go |
| R6: 後方互換fallbackの削除 | Proposed Changes > resolve/worktree.go |
| R7: listコマンドの出力変更 | Proposed Changes > cmd/list.go |

## Proposed Changes

### state パッケージ（データ構造再設計）

#### [MODIFY] [state_test.go](file://features/devctl/internal/state/state_test.go)
*   **Description**: 新しい構造体とStatePath仕様に合わせてテスト全体を書き直し
*   **Technical Design**:
    ```go
    func TestStatePath(t *testing.T)
    // tests := []struct{ repoRoot, branch, expected string }
    // {"with branch", "/repo", "test-001", "/repo/work/test-001.state.yaml"}
    // featureパラメータは不要

    func TestSave_Load_Roundtrip(t *testing.T)
    // StateFile{ Branch, CreatedAt, Features: map[string]FeatureState{...} }
    // roundtrip保存・読込を確認
    // FeatureState内のConnectivity構造を含む

    func TestSetFeature_NewEntry(t *testing.T)
    // 空のFeaturesマップに新規featureを追加

    func TestSetFeature_UpdateExisting(t *testing.T)
    // 既存featureのステータスを更新

    func TestRemoveFeature(t *testing.T)
    // feature削除後のFeaturesマップを確認

    func TestRemoveFeature_LastOne(t *testing.T)
    // 最後のfeature削除でFeaturesが空になること

    func TestUpdateFeatureStatus(t *testing.T)
    // 既存featureのStatusのみ変更し、Connectivityが保持されることを確認

    func TestUpdateFeatureStatus_NotFound(t *testing.T)
    // 存在しないfeatureに対するUpdateFeatureStatusがエラーを返すこと

    func TestHasActiveFeatures_True(t *testing.T)
    func TestHasActiveFeatures_False(t *testing.T)
    // active statusのfeatureがあるか判定
    ```

#### [MODIFY] [state.go](file://features/devctl/internal/state/state.go)
*   **Description**: StateFile構造体をブランチ単位に再設計。feature毎の状態をマップで管理
*   **Technical Design**:
    ```go
    type Status string
    const (
        StatusActive  Status = "active"
        StatusStopped Status = "stopped"
        StatusClosed  Status = "closed"
    )

    type DockerConnectivity struct {
        Enabled       bool   `yaml:"enabled"`
        ContainerName string `yaml:"container_name"`
        Devcontainer  bool   `yaml:"devcontainer"`
    }

    type SSHConnectivity struct {
        Enabled  bool   `yaml:"enabled"`
        Endpoint string `yaml:"endpoint,omitempty"`
    }

    type Connectivity struct {
        Docker DockerConnectivity `yaml:"docker"`
        SSH    SSHConnectivity    `yaml:"ssh"`
    }

    type FeatureState struct {
        Status       Status       `yaml:"status"`
        StartedAt    time.Time    `yaml:"started_at"`
        Connectivity Connectivity `yaml:"connectivity"`
    }

    type StateFile struct {
        Branch    string                  `yaml:"branch"`
        CreatedAt time.Time               `yaml:"created_at"`
        Features  map[string]FeatureState `yaml:"features,omitempty"`
    }

    // StatePath: featureパラメータを削除
    // returns: filepath.Join(repoRoot, "work", branch+".state.yaml")
    func StatePath(repoRoot, branch string) string

    // SetFeature: stateファイルにfeatureエントリを追加・更新（全フィールド上書き）
    func (s *StateFile) SetFeature(feature string, fs FeatureState)

    // UpdateFeatureStatus: 既存featureのStatusのみ変更。Connectivityは保持
    // featureが存在しない場合はエラーを返す
    func (s *StateFile) UpdateFeatureStatus(feature string, status Status) error

    // RemoveFeature: stateファイルからfeatureエントリを削除
    func (s *StateFile) RemoveFeature(feature string)

    // HasActiveFeatures: active statusのfeatureが1つ以上あるか
    func (s *StateFile) HasActiveFeatures() bool

    // ActiveFeatureNames: active statusのfeature名一覧
    func (s *StateFile) ActiveFeatureNames() []string

    // Load, Save, Remove: 既存を維持（Loadの戻り値型のみ変更）
    ```
*   **Logic**:
    - `StatePath`: `filepath.Join(repoRoot, "work", branch+".state.yaml")` を返す
    - `SetFeature`: `s.Features` が nil なら初期化し、キーにfeature名、値にFeatureStateをセット
    - `UpdateFeatureStatus`: `s.Features[feature]` を取得しStatusのみ変更して書き戻す。キーが存在しなければエラー
    - `RemoveFeature`: `delete(s.Features, feature)`
    - `HasActiveFeatures`: rangeでFeatures走査し、StatusActiveが1つ以上あればtrue

---

### worktree パッケージ（パス・操作の簡略化）

#### [MODIFY] [worktree_test.go](file://features/devctl/internal/worktree/worktree_test.go)
*   **Description**: 全テストからfeatureパラメータを削除。Listテストを削除（stateベースに移行）
*   **Technical Design**:
    ```go
    func TestPath(t *testing.T)
    // m.Path("test-001") == filepath.Join(m.RepoRoot, "work", "test-001")

    func TestExists_True(t *testing.T)
    // m.Path("test-001") のディレクトリを作成し Exists("test-001") == true

    func TestExists_False(t *testing.T)
    // Exists("nonexistent") == false

    func TestCreateCmd(t *testing.T)
    // m.Create("test-001") — featureパラメータ削除

    func TestRemoveCmd(t *testing.T)
    // m.Remove("test-001", false) — featureパラメータ削除

    func TestRemoveCmd_Force(t *testing.T)
    // m.Remove("test-001", true) — featureパラメータ削除

    func TestDeleteBranchCmd(t *testing.T)
    func TestDeleteBranchCmd_Force(t *testing.T)
    // 変更なし
    ```
*   **Logic**:
    - `TestPath_WithFeature`と`TestPath_NoFeature`を統合して`TestPath`に
    - `TestListEntries`と`TestListEntries_NoFeatures`を削除

#### [MODIFY] [worktree.go](file://features/devctl/internal/worktree/worktree.go)
*   **Description**: Path/Create/Remove/Existsからfeatureパラメータを削除。List関数を削除
*   **Technical Design**:
    ```go
    // Path: feature引数削除。 "work/<branch>" を返す
    func (m *Manager) Path(branch string) string {
        return filepath.Join(m.RepoRoot, "work", branch)
    }

    func (m *Manager) Exists(branch string) bool
    func (m *Manager) Create(branch string) error
    func (m *Manager) Remove(branch string, force bool) error

    // List関数を削除（stateファイルベースに移行）

    // DeleteBranch: 変更なし
    ```
*   **Logic**:
    - `Path`: `filepath.Join(m.RepoRoot, "work", branch)` を返す
    - `Exists`: `os.Stat(m.Path(branch))` で判定
    - `Create`: `m.Path(branch)` を使って`git worktree add`
    - `Remove`: `m.Path(branch)` を使って`git worktree remove`

---

### resolve パッケージ（パス解決・探索の簡略化）

#### [MODIFY] [worktree_test.go](file://features/devctl/internal/resolve/worktree_test.go)
*   **Description**: feature引数を削除。旧パスfallbackテストを削除
*   **Technical Design**:
    ```go
    func TestResolveWorktree_Found(t *testing.T)
    // work/<branch>/ ディレクトリを作成して Worktree(root, "feat-x") を確認

    func TestResolveWorktree_NotFound(t *testing.T)
    // ディレクトリが存在しない場合のエラー確認

    // 以下のテストを削除:
    // TestResolveWorktree_NewPathStructure → TestResolveWorktree_Found に統合
    // TestResolveWorktree_NoFeature → 不要（feature分離なし）
    // TestResolveWorktree_NoFeature_NotFound → TestResolveWorktree_NotFound に統合
    // TestResolveWorktree_OldPathFallback → 削除（R6: fallback廃止）
    // TestResolveWorktree_OldFeatureOnlyFallback → 削除（R6）
    // TestResolveWorktree_NewPathTakesPriority → 削除（R6）
    ```

#### [MODIFY] [worktree.go](file://features/devctl/internal/resolve/worktree.go)
*   **Description**: feature引数を削除し、`work/<branch>/`のみを返す。旧パスfallbackを削除
*   **Technical Design**:
    ```go
    // Worktree: feature引数削除。work/<branch>/ の存在確認のみ
    func Worktree(repoRoot, branch string) (string, error) {
        path := filepath.Join(repoRoot, "work", branch)
        if info, err := os.Stat(path); err == nil && info.IsDir() {
            return path, nil
        }
        return "", fmt.Errorf("worktree for branch %q not found", branch)
    }
    ```

#### [MODIFY] [devcontainer_test.go](file://features/devctl/internal/resolve/devcontainer_test.go)
*   **Description**: `work/<branch>/features/<feature>/`系テストを`work/<branch>/`に更新。旧パスfallbackテストを削除
*   **Technical Design**:
    ```go
    func TestLoadDevcontainerConfig_FromJSON(t *testing.T)
    // work/<branch>/.devcontainer/devcontainer.json を作成して確認

    func TestLoadDevcontainerConfig_FeatureDir(t *testing.T)
    // features/<feature>/.devcontainer/ から読み込み（変更なし）

    func TestLoadDevcontainerConfig_FeatureDirPriority(t *testing.T)
    // features/<feature>/ と work/<branch>/ の両方があるとき features/ が優先

    // 以下のテストを削除:
    // TestLoadDevcontainerConfig_OldPathFallback → 削除（R6）
    ```

#### [MODIFY] [devcontainer.go](file://features/devctl/internal/resolve/devcontainer.go)
*   **Description**: 探索パスを簡略化。旧パスfallbackを削除
*   **Technical Design**:
    ```go
    // LoadDevcontainerConfig 探索優先順位:
    //  1. features/<feature>/.devcontainer/devcontainer.json
    //  2. work/<branch>/.devcontainer/devcontainer.json
    //  3. work/<branch>/.devcontainer/Dockerfile (fallback)
    //  4. work/<branch>/Dockerfile (fallback)
    func LoadDevcontainerConfig(repoRoot, feature, branch string) (DevcontainerConfig, error)
    ```
*   **Logic**:
    - `work/<branch>/features/<feature>/` 系の探索パス（Priority 2, 5, 6）を削除
    - `work/<feature>/<branch>/` 系（Priority 3, 4）を削除
    - Priority 2 を `work/<branch>/.devcontainer/devcontainer.json` に変更
    - Dockerfile fallback を `work/<branch>/.devcontainer/Dockerfile` と `work/<branch>/Dockerfile` に変更

---

### action パッケージ（close動作変更）

#### [MODIFY] [close.go](file://features/devctl/internal/action/close.go)
*   **Description**: CloseOptionsからfeatureパラメータを削除し、stateベースのclose動作に変更
*   **Technical Design**:
    ```go
    type CloseOptions struct {
        Feature       string   // 空の場合は全featureクリーンアップ
        Branch        string
        Force         bool
        RepoRoot      string
        ProjectName   string   // コンテナ名解決用に追加
    }

    // Close の新ロジック:
    // feature指定あり:
    //   1. 当該featureのコンテナを停止・削除
    //   2. stateファイルから当該featureエントリを削除
    //   3. worktreeは保持
    //
    // feature指定なし:
    //   1. stateファイルから全activeなfeatureを取得
    //   2. 各featureのコンテナを停止・削除
    //   3. 失敗したコンテナがあればworktree削除をスキップ
    //   4. 全コンテナ停止成功 → worktree削除 + ブランチ削除 + stateファイル削除
    func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error
    ```
*   **Logic**:
    - ContainerNameフィールドを削除し、ProjectNameからコンテナ名を動的解決: `resolve.ContainerName(opts.ProjectName, feature)`
    - `wm.Path(feature, branch)` → `wm.Path(branch)` に変更
    - `state.StatePath(repoRoot, feature, branch)` → `state.StatePath(repoRoot, branch)` に変更
    - featureなしcloseの場合: stateファイルを読み込み、`ActiveFeatureNames()`で一覧取得→各featureのコンテナを順次停止→失敗カウント0ならworktree削除

---

### cmd パッケージ（コマンド層）

#### [MODIFY] [up.go](file://features/devctl/cmd/up.go)
*   **Description**: worktree操作からfeatureパラメータを削除。state保存をSetFeature方式に変更
*   **Technical Design**:
    - `wm.Exists(ctx.Feature, ctx.Branch)` → `wm.Exists(ctx.Branch)` に変更
    - `wm.Path(ctx.Feature, ctx.Branch)` → `wm.Path(ctx.Branch)` に変更
    - `wm.Create(ctx.Feature, ctx.Branch)` → `wm.Create(ctx.Branch)` に変更
    - `resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)` → `resolve.Worktree(ctx.RepoRoot, ctx.Branch)` に変更
    - state保存: `state.StatePath` → featureなしの新シグネチャ使用。feature指定時は`SetFeature`で追加
*   **Logic**:
    - feature有無に関わらずworktreeは同じパス (`work/<branch>/`)
    - state保存:
      ```go
      statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
      sf, _ := state.Load(statePath) // 既存読込（なければゼロ値）
      if sf.Branch == "" {
          sf.Branch = ctx.Branch
          sf.CreatedAt = time.Now()
      }
      if ctx.HasFeature() {
          sf.SetFeature(ctx.Feature, state.FeatureState{
              Status:    state.StatusActive,
              StartedAt: time.Now(),
              Connectivity: state.Connectivity{
                  Docker: state.DockerConnectivity{
                      Enabled:       true,
                      ContainerName: containerName,
                      Devcontainer:  !dcCfg.IsEmpty(),
                  },
                  SSH: state.SSHConnectivity{Enabled: upFlagSSH},
              },
          })
      }
      state.Save(statePath, sf)
      ```

#### [MODIFY] [close.go](file://features/devctl/cmd/close.go)
*   **Description**: close呼び出しをProjectNameベースに変更
*   **Technical Design**:
    - `containerName := resolve.ContainerName(...)` の行を削除
    - `wm` のworktree操作からfeatureパラメータを削除
    - `CloseOptions`にProjectNameを渡す方式に変更
    ```go
    func runClose(cmd *cobra.Command, args []string) error {
        // ...
        ctx.ActionRunner.Close(action.CloseOptions{
            Feature:     ctx.Feature,
            Branch:      ctx.Branch,
            Force:       closeFlagForce,
            RepoRoot:    ctx.RepoRoot,
            ProjectName: projectName,
        }, wm)
    }
    ```

#### [MODIFY] [down.go](file://features/devctl/cmd/down.go)
*   **Description**: state更新を新しいSetFeature/StatePathシグネチャに変更
*   **Technical Design**:
    - `state.StatePath(ctx.RepoRoot, ctx.Feature, ctx.Branch)` → `state.StatePath(ctx.RepoRoot, ctx.Branch)` に変更
    - stateの更新: Load → `UpdateFeatureStatus`でStatusのみ変更（Connectivity保持） → Save
    ```go
    statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
    if sf, err := state.Load(statePath); err == nil {
        if err := sf.UpdateFeatureStatus(ctx.Feature, state.StatusStopped); err != nil {
            ctx.Logger.Warn("Failed to update feature status: %v", err)
        }
        state.Save(statePath, sf)
    }
    ```

#### [MODIFY] [status.go](file://features/devctl/cmd/status.go)
*   **Description**: worktree操作からfeatureパラメータを削除
*   **Technical Design**:
    - `wm.Path(ctx.Feature, ctx.Branch)` → `wm.Path(ctx.Branch)` に変更
    - `resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)` → `resolve.Worktree(ctx.RepoRoot, ctx.Branch)` に変更
    - `wm.Exists(ctx.Feature, ctx.Branch)` → `wm.Exists(ctx.Branch)` に変更

#### [MODIFY] [open.go](file://features/devctl/cmd/open.go)
*   **Description**: worktree操作からfeatureパラメータを削除、state保存を新方式に変更
*   **Technical Design**:
    - `wm.Exists(ctx.Feature, ctx.Branch)` → `wm.Exists(ctx.Branch)` に変更
    - `wm.Path(ctx.Feature, ctx.Branch)` → `wm.Path(ctx.Branch)` に変更
    - `wm.Create(ctx.Feature, ctx.Branch)` → `wm.Create(ctx.Branch)` に変更
    - `resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)` → `resolve.Worktree(ctx.RepoRoot, ctx.Branch)` に変更
    - state保存: Load→SetFeature→Save方式に変更（up.goと同じパターン）

#### [MODIFY] [list.go](file://features/devctl/cmd/list.go)
*   **Description**: ファイルシステムスキャンをstateファイル読み取り方式に変更
*   **Technical Design**:
    ```go
    func runList(cmd *cobra.Command, args []string) error {
        // ...
        statePath := state.StatePath(ctx.RepoRoot, branch)
        sf, err := state.Load(statePath)
        if err != nil {
            fmt.Fprintf(os.Stdout, "No state found for branch %q\n", branch)
            return nil
        }
        if len(sf.Features) == 0 {
            fmt.Fprintf(os.Stdout, "No features for branch %q\n", branch)
            return nil
        }
        // Print table from sf.Features map
        for name, fs := range sf.Features {
            fmt.Fprintf(os.Stdout, "%-20s %-10s %-20s %s\n",
                name, string(fs.Status),
                fs.Connectivity.Docker.ContainerName,
                fs.StartedAt.Format("2006-01-02 15:04"))
        }
    }
    ```
*   **Logic**: `wm.List()` 呼び出しを削除し、`state.Load()` + `sf.Features` イテレーションに置換

---

### resolve パッケージ（container.go — 変更なし）

#### container.go
*   **変更なし** — `ContainerName` / `ImageName` は既にfeature空チェックを行っている

---

### resolve パッケージ（gitworktree.go — 変更なし）

#### gitworktree.go
*   **変更なし** — パスに依存しない汎用的な実装

---

### cmd パッケージ (common.go — 変更なし)

#### common.go
*   **変更なし** — `ParseBranchFeature`, `HasFeature`, `InitContext` はそのまま

---

## Step-by-Step Implementation Guide

### Phase 1: state パッケージ再設計（TDD）

1.  **state_test.go を書き直し**:
    - 既存テストをすべて新しい構造体・シグネチャに合わせて更新
    - `TestStatePath`, `TestSave_Load_Roundtrip`, `TestSetFeature_NewEntry`, `TestSetFeature_UpdateExisting`, `TestRemoveFeature`, `TestRemoveFeature_LastOne`, `TestHasActiveFeatures_True/False`
    - **この時点でテストはコンパイルエラーになる（期待通り）**

2.  **state.go を実装**:
    - 構造体定義を変更（DockerConnectivity, SSHConnectivity, Connectivity, FeatureState, StateFile）
    - `StatePath` のシグネチャ変更（feature引数削除）
    - `SetFeature`, `RemoveFeature`, `HasActiveFeatures`, `ActiveFeatureNames` メソッド追加
    - `Load`, `Save`, `Remove` は既存ロジック維持

3.  **ビルド確認**: `./scripts/process/build.sh` でstate パッケージのテストがパスすることを確認

### Phase 2: worktree パッケージ簡略化（TDD）

4.  **worktree_test.go を更新**:
    - feature引数を削除した新シグネチャでテスト更新
    - `TestListEntries`, `TestListEntries_NoFeatures` を削除

5.  **worktree.go を実装**:
    - `Path`, `Exists`, `Create`, `Remove` からfeature引数を削除
    - `List` 関数を削除

6.  **ビルド確認**: `./scripts/process/build.sh`

### Phase 3: resolve パッケージ簡略化（TDD）

7.  **resolve/worktree_test.go を更新**:
    - feature引数を削除。旧パスfallbackテスト削除

8.  **resolve/worktree.go を実装**:
    - feature引数を削除。fallbackロジック削除

9.  **resolve/devcontainer_test.go を更新**:
    - `work/<branch>/features/<feature>/` 系パスを `work/<branch>/` に変更。旧パスfallbackテスト削除

10. **resolve/devcontainer.go を実装**:
    - 探索パスを簡略化

11. **ビルド確認**: `./scripts/process/build.sh`

### Phase 4: action パッケージ変更

12. **action/close.go を更新**:
    - CloseOptions変更、featureなし/あり分岐実装

13. **ビルド確認**: `./scripts/process/build.sh`

### Phase 5: cmd パッケージ更新

14. **cmd/up.go を更新**:
    - worktree操作のfeature引数削除、state保存をSetFeature方式に

15. **cmd/close.go を更新**:
    - ProjectNameベースに変更

16. **cmd/down.go を更新**:
    - state更新を新方式に

17. **cmd/status.go を更新**:
    - worktree操作のfeature引数削除

18. **cmd/open.go を更新**:
    - worktree操作のfeature引数削除、state保存を新方式に

19. **cmd/list.go を更新**:
    - stateファイルベースに変更

20. **全体ビルド確認**: `./scripts/process/build.sh`

### Phase 6: 最終検証

21. **全体テスト実行**: `./scripts/process/build.sh && ./scripts/process/integration_test.sh`

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    各Phase終了時に実行:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    Phase 6で全体検証:
    ```bash
    ./scripts/process/integration_test.sh
    ```

## Documentation

#### [MODIFY] [README.md](file://README.md)
*   **更新内容**: worktreeパス構造の説明を`work/<branch>/`に更新。stateファイルパスを`work/<branch>.state.yaml`に更新。
