# 000-CLI-Argument-Reorder-Part1

> **Source Specification**: `prompts/phases/000-foundation/ideas/feat-remove-feature-name/000-CLI-Argument-Reorder.md`

## Goal Description

devctlのCLIコマンド体系を `<feature> [branch]` から `<branch> [feature]` に変更し、feature省略時はdev container起動をスキップする。
Part1では**内部パッケージの変更**（引数解析、パス解決、worktree管理、state管理）を行う。

## User Review Required

> [!IMPORTANT]
> - R4で回答未定の`down`/`shell`/`exec`のfeature省略時動作は、「featureが必要なコマンドにfeatureなしで実行→エラー」として実装する前提で計画しています。
> - `list`コマンドは引数をbranchに変更し、`work/<branch>/features/`配下のfeature一覧を表示する形にします。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R5: `ParseFeatureBranch`改名と引数解析変更 | Proposed Changes > cmd/common.go |
| R6: `HasFeature()`ヘルパー | Proposed Changes > cmd/common.go |
| R7: worktreeパス構造変更 | Proposed Changes > resolve/worktree.go, worktree/worktree.go |
| R8: ContainerName/ImageName featureチェック | Proposed Changes > resolve/container.go |
| O1: stateファイルパス変更 | Proposed Changes > state/state.go |

## Proposed Changes

### cmd パッケージ (引数解析)

#### [MODIFY] [common_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/common_test.go)

> **注意**: このファイルが存在しない場合は新規作成する。

*   **Description**: `ParseBranchFeature`と`HasFeature`のテストを追加
*   **Technical Design**:
    ```go
    func TestParseBranchFeature(t *testing.T) {
        tests := []struct {
            name        string
            args        []string
            wantBranch  string
            wantFeature string
        }{
            {"branch only", []string{"feat-x"}, "feat-x", ""},
            {"branch and feature", []string{"feat-x", "devctl"}, "feat-x", "devctl"},
        }
        // テーブル駆動テスト
    }

    func TestHasFeature(t *testing.T) {
        tests := []struct {
            name    string
            feature string
            want    bool
        }{
            {"empty feature", "", false},
            {"with feature", "devctl", true},
        }
        // ctx.HasFeature()をテスト
    }

    func TestInitContext_BranchOnly(t *testing.T) {
        // args=["feat-x"] → Branch="feat-x", Feature=""
    }

    func TestInitContext_BranchAndFeature(t *testing.T) {
        // args=["feat-x", "devctl"] → Branch="feat-x", Feature="devctl"
    }
    ```

#### [MODIFY] [common.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/cmd/common.go)

*   **Description**: 引数解析ロジックの変更とヘルパーメソッド追加
*   **Technical Design**:
    *   `ParseFeatureBranch` → `ParseBranchFeature` にリネーム
    *   返り値の順序変更: `(feature, branch string)` → `(branch, feature string)`
    *   feature省略時: feature="" (空文字列)
    *   `InitContext`: エラーメッセージを "branch name is required" に変更
    *   `HasFeature()` メソッドを `AppContext` に追加
*   **Logic**:
    ```go
    func ParseBranchFeature(args []string) (branch, feature string) {
        branch = args[0]
        if len(args) >= 2 {
            feature = args[1]
        }
        return
    }

    func (ctx *AppContext) HasFeature() bool {
        return ctx.Feature != ""
    }
    ```
    *   `InitContext`内の呼び出し:
        ```go
        func InitContext(args []string) (*AppContext, error) {
            if len(args) == 0 {
                return nil, fmt.Errorf("branch name is required")
            }
            branch, feature := ParseBranchFeature(args)
            // ... ctx.Feature = feature, ctx.Branch = branch
        }
        ```

---

### resolve パッケージ (パス解決)

#### [MODIFY] [worktree_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/worktree_test.go)

*   **Description**: 新パス構造 `work/<branch>/features/<feature>` および `work/<branch>/all/` のテスト追加
*   **Technical Design**:
    ```go
    func TestResolveWorktree_NewPathStructure(t *testing.T) {
        root := t.TempDir()
        // work/<branch>/features/<feature> パス
        featureDir := filepath.Join(root, "work", "feat-x", "features", "devctl")
        require.NoError(t, os.MkdirAll(featureDir, 0755))

        path, err := resolve.Worktree(root, "devctl", "feat-x")
        require.NoError(t, err)
        assert.Equal(t, featureDir, path)
    }

    func TestResolveWorktree_NoFeature(t *testing.T) {
        root := t.TempDir()
        // work/<branch>/all/ パス
        allDir := filepath.Join(root, "work", "feat-x", "all")
        require.NoError(t, os.MkdirAll(allDir, 0755))

        path, err := resolve.Worktree(root, "", "feat-x")
        require.NoError(t, err)
        assert.Equal(t, allDir, path)
    }
    ```
*   **既存テストの修正**: `TestResolveWorktree_FeatureBranch` のディレクトリパスを新構造に変更

#### [MODIFY] [worktree.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/worktree.go)

*   **Description**: `Worktree`関数のパス解決ロジックを新構造に変更
*   **Technical Design**:
    *   引数: `(repoRoot, feature, branch string)` — 変更なし
    *   feature="" の場合: `work/<branch>/all/` を解決
    *   feature!="" の場合: `work/<branch>/features/<feature>` を解決
    *   後方互換: 旧パス `work/<feature>/<branch>` もフォールバックとして検索
*   **Logic**:
    ```go
    func Worktree(repoRoot, feature, branch string) (string, error) {
        if feature == "" {
            // Feature省略: work/<branch>/all/
            allPath := filepath.Join(repoRoot, "work", branch, "all")
            if info, err := os.Stat(allPath); err == nil && info.IsDir() {
                return allPath, nil
            }
            return "", fmt.Errorf("worktree for branch %q (no feature) not found", branch)
        }

        // Feature指定あり
        // Priority 1: work/<branch>/features/<feature> (新構造)
        newPath := filepath.Join(repoRoot, "work", branch, "features", feature)
        if info, err := os.Stat(newPath); err == nil && info.IsDir() {
            return newPath, nil
        }

        // Priority 2: work/<feature>/<branch> (旧構造 - 後方互換)
        oldPath := filepath.Join(repoRoot, "work", feature, branch)
        if info, err := os.Stat(oldPath); err == nil && info.IsDir() {
            return oldPath, nil
        }

        // Priority 3: work/<feature> (旧構造 - 後方互換)
        oldFallback := filepath.Join(repoRoot, "work", feature)
        if info, err := os.Stat(oldFallback); err == nil && info.IsDir() {
            return oldFallback, nil
        }

        return "", fmt.Errorf("worktree for feature %q branch %q not found", feature, branch)
    }
    ```

#### [MODIFY] [container_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/container_test.go)

*   **Description**: feature空文字時のテストケース追加
*   **Technical Design**:
    ```go
    // 既存TestContainerNameのテーブルに追加:
    {"myproj", "", ""},  // feature空 → 空文字列を返す
    ```

#### [MODIFY] [container.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/container.go)

*   **Description**: feature空文字時に空文字列を返すように変更
*   **Technical Design**:
    ```go
    func ContainerName(project, feature string) string {
        if feature == "" {
            return ""
        }
        return sanitize(project) + "-" + sanitize(feature)
    }

    func ImageName(project, feature string) string {
        if feature == "" {
            return ""
        }
        return sanitize(project) + "-dev-" + sanitize(feature)
    }
    ```

#### [MODIFY] [devcontainer_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/devcontainer_test.go)

*   **Description**: 新パス構造のテスト追加と既存テストのパス修正
*   **Technical Design**:
    *   既存テストのパスを `work/<feature>/<branch>` → `work/<branch>/features/<feature>` に変更
    *   `features/<feature>/.devcontainer/` からの読み込みテスト（変更なし）

#### [MODIFY] [devcontainer.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/resolve/devcontainer.go)

*   **Description**: `LoadDevcontainerConfig`のパス検索を新構造に変更、feature空の場合は早期リターン
*   **Technical Design**:
    *   feature="" の場合: `DevcontainerConfig{}` を即座に返す（container不要なので）
    *   feature!="" の場合:
        1. `features/<feature>/.devcontainer/devcontainer.json` （変更なし）
        2. `work/<branch>/features/<feature>/.devcontainer/devcontainer.json` （新構造）
        3. `work/<feature>/<branch>/.devcontainer/devcontainer.json` （旧構造フォールバック）
        4. `work/<feature>/.devcontainer/devcontainer.json` （旧構造フォールバック）

---

### worktree パッケージ

#### [MODIFY] [worktree_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/worktree/worktree_test.go)

*   **Description**: 新パス構造のテスト修正・追加
*   **Technical Design**:
    ```go
    func TestPath_WithFeature(t *testing.T) {
        m := newTestManager(t, true)
        got := m.Path("devctl", "test-001")
        // 新構造: work/<branch>/features/<feature>
        assert.Equal(t, filepath.Join(m.RepoRoot, "work", "test-001", "features", "devctl"), got)
    }

    func TestPath_NoFeature(t *testing.T) {
        m := newTestManager(t, true)
        got := m.Path("", "test-001")
        // feature省略: work/<branch>/all/
        assert.Equal(t, filepath.Join(m.RepoRoot, "work", "test-001", "all"), got)
    }
    ```
    *   既存の `TestPath`, `TestExists_True/False`, `TestCreateCmd`, `TestRemoveCmd`, `TestListEntries` も新パスに修正

#### [MODIFY] [worktree.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/worktree/worktree.go)

*   **Description**: パス構成を新構造に変更、`List`はブランチベースに変更
*   **Technical Design**:
    ```go
    // Path: feature指定あり → work/<branch>/features/<feature>
    //       feature="" → work/<branch>/all
    func (m *Manager) Path(feature, branch string) string {
        if feature == "" {
            return filepath.Join(m.RepoRoot, "work", branch, "all")
        }
        return filepath.Join(m.RepoRoot, "work", branch, "features", feature)
    }
    ```
    *   `Create`, `Remove`, `Exists` は内部で `Path()` を使うので自動的に新パスになる
    *   `List` メソッドはブランチベースに変更:
    ```go
    // List returns all feature worktree entries for a branch by scanning work/<branch>/features/.
    func (m *Manager) List(branch string) ([]WorktreeInfo, error) {
        featuresDir := filepath.Join(m.RepoRoot, "work", branch, "features")
        entries, err := os.ReadDir(featuresDir)
        // ...各エントリを WorktreeInfo{Feature: e.Name(), Branch: branch, ...} として返す
    }
    ```
    *   `WorktreeInfo` の `Feature` と `Branch` フィールドは残す

---

### state パッケージ

#### [MODIFY] [state_test.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/state/state_test.go)

*   **Description**: 新パス構造のテスト修正・追加
*   **Technical Design**:
    ```go
    func TestStatePath_WithFeature(t *testing.T) {
        got := state.StatePath("/repo", "devctl", "test-001")
        expected := filepath.Join("/repo", "work", "test-001", "features", "devctl.state.yaml")
        assert.Equal(t, expected, got)
    }

    func TestStatePath_NoFeature(t *testing.T) {
        got := state.StatePath("/repo", "", "test-001")
        expected := filepath.Join("/repo", "work", "test-001", "all.state.yaml")
        assert.Equal(t, expected, got)
    }
    ```

#### [MODIFY] [state.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/feat-remove-feature-name/features/devctl/internal/state/state.go)

*   **Description**: `StatePath`のパス構造を新形式に変更
*   **Technical Design**:
    ```go
    func StatePath(repoRoot, feature, branch string) string {
        if feature == "" {
            // work/<branch>/all.state.yaml
            return filepath.Join(repoRoot, "work", branch, "all.state.yaml")
        }
        // work/<branch>/features/<feature>.state.yaml
        return filepath.Join(repoRoot, "work", branch, "features", feature+".state.yaml")
    }
    ```

## Step-by-Step Implementation Guide

1.  **テスト作成: cmd/common_test.go**
    *   `cmd/common_test.go`に`ParseBranchFeature`, `HasFeature`, `InitContext`のテストを作成
    *   テスト実行 → 失敗を確認（`ParseBranchFeature`が存在しないため）

2.  **実装: cmd/common.go の引数解析変更**
    *   `ParseFeatureBranch` → `ParseBranchFeature` にリネーム、返り値と引数解析ロジック変更
    *   `HasFeature()` メソッド追加
    *   `InitContext` のエラーメッセージ変更、`ParseBranchFeature`を呼び出すように変更
    *   テスト実行 → パス

3.  **テスト作成: state/state_test.go**
    *   `TestStatePath` を `TestStatePath_WithFeature` にリネーム、期待パスを新構造に変更
    *   `TestStatePath_NoFeature` テスト追加
    *   テスト実行 → 失敗を確認

4.  **実装: state/state.go のパス変更**
    *   `StatePath` 関数を新パス構造に変更
    *   テスト実行 → パス

5.  **テスト作成: resolve/worktree_test.go**
    *   既存テストのパスを新構造に修正
    *   `TestResolveWorktree_NoFeature` テスト追加
    *   テスト実行 → 失敗を確認

6.  **実装: resolve/worktree.go のパス解決ロジック変更**
    *   `Worktree` 関数を新パス構造に変更（旧構造フォールバック付き）
    *   テスト実行 → パス

7.  **テスト作成: resolve/container_test.go**
    *   feature空文字テストケース追加
    *   テスト実行 → 失敗を確認

8.  **実装: resolve/container.go のfeature空チェック追加**
    *   `ContainerName`, `ImageName` にfeature空チェック追加
    *   テスト実行 → パス

9.  **テスト作成: worktree/worktree_test.go**
    *   既存テストのパスを新構造に修正
    *   `TestPath_NoFeature` テスト追加
    *   テスト実行 → 失敗を確認

10. **実装: worktree/worktree.go のパス構造変更**
    *   `Path`, `List` メソッドを新構造に変更
    *   テスト実行 → パス

11. **テスト修正: resolve/devcontainer_test.go**
    *   既存テストのパスを新構造に修正
    *   テスト実行 → 失敗を確認

12. **実装: resolve/devcontainer.go のパス検索変更**
    *   `LoadDevcontainerConfig` の検索パスを新構造に変更
    *   テスト実行 → パス

13. **ビルド検証**
    *   `./scripts/process/build.sh` を実行して全体ビルドと単体テスト確認

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **Log Verification**: 全テストがパスすること、ビルドエラーがないこと

## Documentation

変更なし（Part2完了後にまとめて更新する）

## 継続計画について

Part2ではCLIコマンド層（`cmd/*.go`）の変更とR9（`open --up`）の実装を行う。
