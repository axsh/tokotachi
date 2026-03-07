# 006-ContainerHealthCheck

> **Source Specification**: [005-ContainerHealthCheck.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/005-ContainerHealthCheck.md)

## Goal Description

`devctl up` で `docker run -d` 実行後、コンテナが実際に `running` 状態であることを確認するヘルスチェックを追加する。コンテナが即座に終了した場合はエラーを返し、エディタ起動を抑制する。併せて、devctl の Dockerfile に `CMD ["sleep", "infinity"]` を追加してコンテナの永続稼働を実現する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| コンテナ起動後のヘルスチェック | Proposed Changes > action/up.go |
| 待機時間2秒 | Proposed Changes > action/up.go (`containerStartGracePeriod`) |
| `docker inspect` で `running` 確認 | Proposed Changes > action/up.go (既存 `r.Status()` を再利用) |
| エディタ起動の抑制 | 既存動作で対応（`Up()` がエラーを返せば `cmd/up.go` L159でエディタ前にreturn） |
| `docker logs` 出力をエラーに含める | Proposed Changes > action/up.go |
| devctl Dockerfile に CMD 追加 | Proposed Changes > Dockerfile |

## Proposed Changes

### action パッケージ

#### [MODIFY] [up.go](file:///c:/Users/yamya/myprog/escape/features/devctl/internal/action/up.go)
*   **Description**: `docker run -d` 成功後にコンテナ生存確認を追加
*   **Technical Design**:
    *   定数追加:
        ```go
        const containerStartGracePeriod = 2 * time.Second
        ```
    *   `Up()` 関数の L96-98 を以下のロジックに置き換え:
        ```go
        // Step 5: Verify container is still running after grace period
        r.Logger.Debug("Waiting %s for container to stabilize...", containerStartGracePeriod)
        time.Sleep(containerStartGracePeriod)

        state := r.Status(opts.ContainerName, "")
        if state != StateContainerRunning {
            // Collect container logs for diagnosis
            logs, _ := r.DockerRunOutput("logs", "--tail", "20", opts.ContainerName)
            return fmt.Errorf(
                "container %s exited immediately after start. "+
                    "Ensure the Dockerfile has a CMD that keeps the process running "+
                    "(e.g. CMD [\"sleep\", \"infinity\"]).\n"+
                    "Container logs:\n%s", opts.ContainerName, logs,
            )
        }

        r.Logger.Info("Container %s started successfully", opts.ContainerName)
        ```
*   **Logic**:
    1. `docker run -d` が exit code 0 で完了（L94-96, 既存のまま）
    2. 2秒間 `time.Sleep` で待機（コンテナ初期化時間を確保）
    3. `r.Status(containerName, "")` を呼び出して `StateContainerRunning` か確認
    4. 非 Running の場合、`docker logs --tail 20` で最新ログを取得しエラーメッセージに含める
    5. Running の場合、"started successfully" を表示して正常return
*   **Import追加**: `"fmt"`, `"time"` を import に追加

---

### devctl Dockerfile

#### [MODIFY] [Dockerfile](file:///c:/Users/yamya/myprog/escape/features/devctl/.devcontainer/Dockerfile)
*   **Description**: コンテナ永続稼働のための `CMD` を追加
*   **Technical Design**:
    *   末尾に追加:
        ```dockerfile
        CMD ["sleep", "infinity"]
        ```
*   **Logic**: `sleep infinity` はシグナルを受け取るまで無期限に実行され続ける。`docker stop` で SIGTERM が送られると正常終了する。

---

### integration-test Dockerfile

#### [MODIFY] [Dockerfile](file:///c:/Users/yamya/myprog/escape/features/integration-test/.devcontainer/Dockerfile)
*   **Description**: integration-test のコンテナも永続稼働するように `CMD` を追加
*   **Technical Design**:
    *   末尾に追加:
        ```dockerfile
        CMD ["sleep", "infinity"]
        ```

## Step-by-Step Implementation Guide

1.  **devctl Dockerfile に CMD 追加**:
    *   `features/devctl/.devcontainer/Dockerfile` の末尾に `CMD ["sleep", "infinity"]` を追加

2.  **integration-test Dockerfile に CMD 追加**:
    *   `features/integration-test/.devcontainer/Dockerfile` の末尾に `CMD ["sleep", "infinity"]` を追加

3.  **action/up.go にヘルスチェック追加**:
    *   import に `"fmt"`, `"time"` を追加
    *   定数 `containerStartGracePeriod = 2 * time.Second` を追加
    *   `Up()` 関数の `docker run` 成功後（L96-98）にヘルスチェックロジックを挿入

4.  **ビルドと統合テスト実行**

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests**:
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```
    *   **Log Verification**:
        *   `TestDevctlUpStartsContainer` — コンテナが起動し、ヘルスチェック通過後に PASS
        *   `TestDevctlDockerfileBuild` — 更新後の Dockerfile がビルド成功
        *   `TestIntegrationTestDockerfileBuild` — 更新後の Dockerfile がビルド成功
        *   全8テストが PASS

## Documentation

#### [MODIFY] [features/devctl/README.md](file:///c:/Users/yamya/myprog/escape/features/devctl/README.md)
*   **更新内容**: コンテナ起動後のヘルスチェック機能についての記述を追加
