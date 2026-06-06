# 004-Scaffold-RootDir-AutoDetect

> **Source Specification**: [004-Scaffold-RootDir-AutoDetect.md](file://prompts/phases/000-foundation/ideas/feat-devctl-scafford/004-Scaffold-RootDir-AutoDetect.md)

## Goal Description

`devctl scaffold` のテンプレート展開先ルートディレクトリを、`os.Getwd()` による CWD 固定から、`git rev-parse --show-toplevel` によるGitルート自動検出（ハイブリッドフォールバック）に変更する。`--cwd` フラグによりCWD強制使用も可能にする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| Gitルート自動検出 (`git rev-parse --show-toplevel`) | Proposed Changes > `cmd/scaffold.go` の `resolveRepoRoot` 関数 |
| ハイブリッドフォールバック (Git失敗時はCWD) | Proposed Changes > `cmd/scaffold.go` の `resolveRepoRoot` 関数 |
| `--cwd` フラグ | Proposed Changes > `cmd/scaffold.go` のフラグ追加 + `runScaffold` 内の分岐 |
| 既存統合テストの維持 | Verification Plan > 既存テスト実行 |

## Proposed Changes

### cmd パッケージ

#### テスト先行: `resolveRepoRoot` のロジック

`resolveRepoRoot` は `cmd` パッケージ内のパッケージプライベート関数となるため、`cmd` パッケージの単体テストとして検証する。ただし、`git rev-parse` は実際の Git リポジトリに依存するため、統合テスト側でのE2E検証を主軸とする。

#### [MODIFY] [scaffold.go](file://features/devctl/cmd/scaffold.go)

*   **Description**: `--cwd` フラグを追加し、`repoRoot` 決定ロジックをハイブリッド方式に変更する
*   **Technical Design**:
    *   `scaffoldFlagCwd` (bool) フラグを追加
    *   `resolveRepoRoot(useCwd bool) string` ヘルパー関数を追加
    *   `runScaffold` 内の `repoRoot` 決定部分を `resolveRepoRoot` 呼び出しに置換

    ```go
    var scaffoldFlagCwd bool

    // init() 内に追加:
    scaffoldCmd.Flags().BoolVar(&scaffoldFlagCwd, "cwd", false,
        "Use current working directory as root instead of auto-detecting Git root")

    // resolveRepoRoot determines the target root directory.
    // If useCwd is true, always uses os.Getwd().
    // Otherwise, tries "git rev-parse --show-toplevel" first,
    // falling back to os.Getwd() on failure.
    func resolveRepoRoot(useCwd bool) string
    ```

*   **Logic**:
    1. `useCwd` が `true` の場合:
        - `os.Getwd()` を返す（失敗時は `"."`）
    2. `useCwd` が `false` の場合:
        - `exec.Command("git", "rev-parse", "--show-toplevel")` を実行
        - 成功: 出力を `strings.TrimSpace` してパスを返す
        - 失敗: `os.Getwd()` にフォールバック（失敗時は `"."`）
    3. `runScaffold` 内の既存コード:
        ```go
        // Before:
        repoRoot, err := os.Getwd()
        if err != nil {
            repoRoot = "."
        }

        // After:
        repoRoot := resolveRepoRoot(scaffoldFlagCwd)
        ```

---

### 統合テスト

#### [MODIFY] [devctl_scaffold_test.go](file://tests/integration-test/devctl_scaffold_test.go)

*   **Description**: `--cwd` フラグの統合テストを追加する
*   **Technical Design**:
    ```go
    func TestScaffoldCwdFlag(t *testing.T)
    ```
*   **Logic**:
    1. `t.TempDir()` で一時ディレクトリを作成（Git リポジトリとして初期化**しない**）
    2. `runDevctlInDir(t, tmpDir, "scaffold", "--cwd", "--yes")` を実行
    3. テンプレートが tmpDir（CWD）に展開されたことを確認（`features/README.md` の存在チェック）
    4. Git リポジトリ外でも `--cwd` により正常動作することを検証

> [!NOTE]
> 既存の `TestScaffoldDefault` は `initGitRepo(t, tmpDir)` で Git リポジトリを初期化してから実行するため、`git rev-parse --show-toplevel` は tmpDir を返す。よって既存テストは **変更不要** でパスする。

## Step-by-Step Implementation Guide

1. [x] **`--cwd` フラグの追加**:
    - `features/devctl/cmd/scaffold.go` を編集
    - `scaffoldFlagCwd` 変数を `var` ブロックに追加
    - `init()` 内に `scaffoldCmd.Flags().BoolVar(...)` を追加

2. [x] **`resolveRepoRoot` 関数の実装**:
    - `features/devctl/cmd/scaffold.go` に `resolveRepoRoot(useCwd bool) string` を追加
    - ハイブリッドロジックを実装:
        - `useCwd == true` → `os.Getwd()` を返す
        - `useCwd == false` → `git rev-parse --show-toplevel` を試行 → 失敗時は `os.Getwd()` フォールバック

3. [x] **`runScaffold` の修正**:
    - `features/devctl/cmd/scaffold.go` 内の `repoRoot` 決定部分を `resolveRepoRoot(scaffoldFlagCwd)` に置換

4. [x] **ビルド検証**:
    - `./scripts/process/build.sh` を実行してコンパイルエラーがないことを確認

5. [x] **統合テスト `TestScaffoldCwdFlag` の追加**:
    - `tests/integration-test/devctl_scaffold_test.go` に新テスト関数を追加
    - Git リポジトリ外の tmpDir で `--cwd` 付き scaffold を実行し、展開を確認

6. [x] **統合テスト実行**:
    - 既存テスト + 新規テストを実行して全パスを確認

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2. **Integration Tests**:
    既存テスト + 新規テストを実行:
    ```bash
    ./scripts/process/integration_test.sh --categories "devctl" --specify "TestScaffoldDefault"
    ./scripts/process/integration_test.sh --categories "devctl" --specify "TestScaffoldCwdFlag"
    ```
    *   **Log Verification**:
        - `TestScaffoldDefault`: テストがパスすること（Git ルート自動検出が既存テストと互換）
        - `TestScaffoldCwdFlag`: `--cwd` 指定時に CWD へテンプレートが展開されること

## Documentation

#### [MODIFY] [000-Reference-Manual.md](file://prompts/phases/000-foundation/refs/tokotachi-scaffolds/000-Reference-Manual.md)
*   **更新内容**: `--cwd` フラグの説明を追加
