# 005-IntegrationTestGoMigration

> **Source Specification**: [004-IntegrationTestGoMigration.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/004-IntegrationTestGoMigration.md)

## Goal Description

`tests/integration-test/` のテストコードをPythonからGoに移行する。Pythonファイル（`conftest.py`, `test_*.py`）を削除し、同等のテスト内容をGo標準`testing`パッケージと`os/exec`で再実装する。独自の`go.mod`を配置し、devctlモジュールとは独立したテストモジュールとする。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| PythonコードをGoに置き換え | Proposed Changes > DELETE Python files + NEW Go files |
| 独自の `go.mod` を配置 | Proposed Changes > go.mod |
| Dockerfileビルド検証の維持 | Proposed Changes > docker_build_test.go |
| devctl up テストの維持 | Proposed Changes > devctl_up_test.go |
| devctl down テストの維持 | Proposed Changes > devctl_down_test.go |
| devctl status テストの維持 | Proposed Changes > devctl_status_test.go |
| `os/exec` でバイナリ呼び出し | Proposed Changes > helpers_test.go |
| `integration_test.sh` との互換性 | 既存Goテスト検出が機能（変更不要） |
| テストスキップ禁止 | 全テストで `t.Fatalf()` を使用 |

## Proposed Changes

### tests/integration-test/ (テストコード)

> TDD: テストファイルを先に記述する。

#### [DELETE] [conftest.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/conftest.py)
#### [DELETE] [test_docker_build.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_docker_build.py)
#### [DELETE] [test_devctl_up.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_up.py)
#### [DELETE] [test_devctl_down.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_down.py)
#### [DELETE] [test_devctl_status.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_status.py)

---

#### [NEW] [go.mod](file:///c:/Users/yamya/myprog/escape/tests/integration-test/go.mod)
*   **Description**: 独立したGoモジュール定義
*   **Technical Design**:
    ```go
    module github.com/escape-dev/integration-test
    go 1.24.0
    ```
    *   依存: `github.com/stretchr/testify` (assert用)

#### [NEW] [helpers_test.go](file:///c:/Users/yamya/myprog/escape/tests/integration-test/helpers_test.go)
*   **Description**: 共通ヘルパー関数。旧 `conftest.py` 相当
*   **Technical Design**:
    *   `package integration_test` (全ファイル共通)
    *   定数:
        *   `featureName = "integration-test"`
    *   `func projectRoot() string`:
        *   `runtime.Caller(0)` でこのファイルの絶対パスを取得
        *   2階層上 (`tests/integration-test/` → プロジェクトルート) を返す
    *   `func devctlBinary(t *testing.T) string`:
        *   `projectRoot() + "/bin/devctl"` (Windows: `+ ".exe"`)
        *   ファイル存在チェック、存在しない場合 `t.Fatalf("devctl binary not found at %s. Run ./scripts/process/build.sh first.", path)`
    *   `func requireDockerAvailable(t *testing.T)`:
        *   `exec.Command("docker", "info").CombinedOutput()` を実行
        *   失敗時 `t.Fatalf("Docker is not available: %v", err)`
    *   `func runDevctl(t *testing.T, args ...string) (stdout, stderr string, exitCode int)`:
        *   `exec.Command(devctlBinary(t), args...)` を実行
        *   `cmd.Dir = projectRoot()`
        *   stdout/stderr を bytes.Buffer でキャプチャ
        *   タイムアウト: `context.WithTimeout(context.Background(), 120*time.Second)`
        *   exitCode: `cmd.ProcessState.ExitCode()` または error時に -1
    *   `func dockerRun(args ...string) (string, error)`:
        *   `exec.Command("docker", args...)` の stdout を返す
    *   `func TestMain(m *testing.M)`:
        *   `m.Run()` でテスト実行
        *   テスト完了後にクリーンアップ:
            *   `docker ps -a --filter name=integration-test --format {{.Names}}` でコンテナ名取得
            *   各コンテナを `docker rm -f` で削除
            *   `docker rmi -f integration-test-verify devctl-verify` でイメージ削除
        *   `os.Exit(exitCode)`

#### [NEW] [docker_build_test.go](file:///c:/Users/yamya/myprog/escape/tests/integration-test/docker_build_test.go)
*   **Description**: feature の Dockerfile が正常にビルドできるかを検証
*   **Technical Design**:
    *   `func TestIntegrationTestDockerfileBuild(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   buildContext: `filepath.Join(projectRoot(), "features", "integration-test", ".devcontainer")`
        *   `exec.Command("docker", "build", "-t", "integration-test-verify", buildContext)` を実行
        *   `assert.Equal(t, 0, exitCode, "Docker build failed: %s", stderr)`
        *   クリーンアップ: `dockerRun("rmi", "-f", "integration-test-verify")`
    *   `func TestDevctlDockerfileBuild(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   buildContext: `filepath.Join(projectRoot(), "features", "devctl", ".devcontainer")`
        *   `exec.Command("docker", "build", "-t", "devctl-verify", buildContext)` を実行
        *   `assert.Equal(t, 0, exitCode, "Docker build failed: %s", stderr)`
        *   クリーンアップ: `dockerRun("rmi", "-f", "devctl-verify")`

#### [NEW] [devctl_up_test.go](file:///c:/Users/yamya/myprog/escape/tests/integration-test/devctl_up_test.go)
*   **Description**: `devctl up` でコンテナが起動するかを検証
*   **Technical Design**:
    *   `func TestDevctlUpStartsContainer(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   クリーン状態確保: `runDevctl(t, "down", featureName)`
        *   `stdout, stderr, code := runDevctl(t, "up", featureName, "--verbose")`
        *   `assert.Equal(t, 0, code, "devctl up failed: stdout=%s stderr=%s", stdout, stderr)`
        *   検証: `dockerRun("ps", "--filter", "name="+featureName, "--format", "{{.Names}}")`
        *   `assert.Contains(t, output, featureName)`
        *   クリーンアップ: `runDevctl(t, "down", featureName)`
    *   `func TestDevctlUpIdempotent(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   `runDevctl(t, "down", featureName)` でクリーン確保
        *   1回目 up → exitCode == 0
        *   2回目 up → exitCode == 0 （エラーにならない）
        *   クリーンアップ

#### [NEW] [devctl_down_test.go](file:///c:/Users/yamya/myprog/escape/tests/integration-test/devctl_down_test.go)
*   **Description**: `devctl down` でコンテナが停止・削除されるかを検証
*   **Technical Design**:
    *   `func TestDevctlDownStopsContainer(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   前提: `runDevctl(t, "up", featureName)` → exitCode == 0
        *   `stdout, stderr, code := runDevctl(t, "down", featureName, "--verbose")`
        *   `assert.Equal(t, 0, code)`
        *   検証: `dockerRun("ps", "-a", "--filter", "name="+featureName, "--format", "{{.Names}}")`
        *   `assert.NotContains(t, output, featureName)`
    *   `func TestDevctlDownNoopWhenNotRunning(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   `runDevctl(t, "down", featureName)` でクリーン確保
        *   再度 down → exitCode == 0

#### [NEW] [devctl_status_test.go](file:///c:/Users/yamya/myprog/escape/tests/integration-test/devctl_status_test.go)
*   **Description**: `devctl status` が正しいステータスを返すかを検証
*   **Technical Design**:
    *   `func TestDevctlStatusWhenRunning(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   前提: `runDevctl(t, "up", featureName)` → exitCode == 0
        *   `stdout, stderr, code := runDevctl(t, "status", featureName)`
        *   `assert.Equal(t, 0, code)`
        *   `assert.Contains(t, strings.ToLower(stdout+stderr), "running")`
        *   クリーンアップ: `runDevctl(t, "down", featureName)`
    *   `func TestDevctlStatusWhenStopped(t *testing.T)`:
        *   `requireDockerAvailable(t)`
        *   `runDevctl(t, "down", featureName)` でクリーン確保
        *   `_, _, code := runDevctl(t, "status", featureName)`
        *   `assert.Equal(t, 0, code)`

## Step-by-Step Implementation Guide

1.  **Pythonテストファイルの削除**:
    *   `tests/integration-test/conftest.py` を削除
    *   `tests/integration-test/test_docker_build.py` を削除
    *   `tests/integration-test/test_devctl_up.py` を削除
    *   `tests/integration-test/test_devctl_down.py` を削除
    *   `tests/integration-test/test_devctl_status.py` を削除

2.  **Go モジュールの初期化**:
    *   `tests/integration-test/go.mod` を作成
    *   `go mod tidy` で依存解決

3.  **共通ヘルパーの作成**:
    *   `tests/integration-test/helpers_test.go` を作成

4.  **テストファイルの作成**:
    *   `tests/integration-test/docker_build_test.go` を作成
    *   `tests/integration-test/devctl_up_test.go` を作成
    *   `tests/integration-test/devctl_down_test.go` を作成
    *   `tests/integration-test/devctl_status_test.go` を作成

5.  **ビルドと統合テスト実行**:
    *   `./scripts/process/build.sh` を実行
    *   `./scripts/process/integration_test.sh --categories "integration-test"` を実行

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests (カテゴリ指定)**:
    Goテストカテゴリとして検出・実行されることを確認。
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```
    *   **Log Verification**:
        *   カテゴリ表示に `integration-test [go]` と表示されること
        *   `go test` 形式の出力（`=== RUN`, `--- PASS`）が含まれること
        *   テスト結果サマリーに passed カウントが表示されること

3.  **Integration Tests (特定テスト)**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDockerBuild"
    ```

## Documentation

#### [MODIFY] [004-IntegrationTest.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/plans/main/004-IntegrationTest.md)
*   **更新内容**: テストコードのGo化に関する注記を追記
