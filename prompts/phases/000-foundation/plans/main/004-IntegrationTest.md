# 004-IntegrationTest

> **Source Specification**: [003-IntegrationTest.md](file:///c:/Users/yamya/myprog/escape/prompts/phases/000-foundation/ideas/main/003-IntegrationTest.md)

## Goal Description

`devctl` CLIの動作を検証する統合テスト環境を構築する。`features/integration-test/` にPythonベースの統合テストプロジェクト（`.devcontainer`付き）を配置し、プロジェクトルート直下の `tests/integration-test/` にテストコードを配置する。`integration_test.sh` をPythonテスト（pytest）にも対応するよう改修する。

## User Review Required

> [!IMPORTANT]
> テストの配置場所について: 仕様書に基づき、テストコードは `$PROJECT_ROOT/tests/integration-test/` に配置します（既存の `integration_test.sh` のカテゴリ検出と一致させるため）。`features/integration-test/` はfeature定義（`.devcontainer`、`requirements.txt`、`README.md`）のみを保持します。

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `features/integration-test/` フォルダの作成 | Proposed Changes > integration-test feature |
| `.devcontainer` による Python 開発環境の整備 | Proposed Changes > devcontainer.json, Dockerfile |
| `tests/` フォルダへのテストコード配置 | Proposed Changes > tests/integration-test/ |
| `integration_test.sh` の改修（Python対応） | Proposed Changes > integration_test.sh |
| `conftest.py` 共通フィクスチャ | Proposed Changes > conftest.py |
| `test_devctl_up.py` テスト | Proposed Changes > test_devctl_up.py |
| `test_devctl_down.py` テスト | Proposed Changes > test_devctl_down.py |
| `test_devctl_status.py` テスト | Proposed Changes > test_devctl_status.py |
| `test_docker_build.py` テスト | Proposed Changes > test_docker_build.py |
| テストスキップ禁止（Testing Rules） | 全テストファイルで `pytest.skip()` を使用しない |

## Proposed Changes

### integration-test feature

#### [NEW] [Dockerfile](file:///c:/Users/yamya/myprog/escape/features/integration-test/.devcontainer/Dockerfile)
*   **Description**: Python 3.12 ベースの統合テスト実行環境
*   **Technical Design**:
    *   ベースイメージ: `python:3.12-slim-bookworm`
    *   追加パッケージ: `docker.io`, `git`, `curl`
    *   `requirements.txt` を `COPY` してから `pip install`
    *   `WORKDIR /workspace`

#### [NEW] [devcontainer.json](file:///c:/Users/yamya/myprog/escape/features/integration-test/.devcontainer/devcontainer.json)
*   **Description**: Python開発用のdevcontainer定義
*   **Technical Design**:
    ```json
    {
        "name": "integration-test",
        "build": {
            "dockerfile": "./Dockerfile",
            "context": ".."
        },
        "workspaceFolder": "/workspace",
        "customizations": {
            "vscode": {
                "extensions": ["ms-python.python", "ms-python.debugpy"]
            }
        },
        "mounts": [
            "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
        ],
        "remoteUser": "root"
    }
    ```

#### [NEW] [requirements.txt](file:///c:/Users/yamya/myprog/escape/features/integration-test/requirements.txt)
*   **Description**: Python依存パッケージ
*   **Technical Design**:
    *   `pytest>=8.0`
    *   `pytest-timeout>=2.0`

#### [NEW] [README.md](file:///c:/Users/yamya/myprog/escape/features/integration-test/README.md)
*   **Description**: 統合テストプロジェクトの説明ドキュメント

---

### tests/integration-test/ (テストコード)

> テストの記述順序: TDDに基づき、テストファイルを先に記述する。

#### [NEW] [conftest.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/conftest.py)
*   **Description**: pytest共通フィクスチャ。全テストファイルで使用される共有ヘルパーを提供
*   **Technical Design**:
    *   フィクスチャ `project_root` → `pathlib.Path`: プロジェクトルート（`tests/integration-test/` から2階層上）
    *   フィクスチャ `devctl_binary` → `pathlib.Path`: `bin/devctl` (`bin/devctl.exe` on Windows) のパス。存在しない場合は `pytest.fail()` で失敗（スキップ禁止ルール準拠）
    *   フィクスチャ `docker_available` → `bool`: `docker info` が成功するか確認。失敗時は `pytest.fail()`
    *   フィクスチャ `feature_name` → `str`: テスト対象featureの名前。固定値 `"integration-test"`
    *   ヘルパー関数 `run_devctl(args: list[str], cwd: Path, timeout: int = 120) -> subprocess.CompletedProcess`:
        *   `devctl_binary` を使って指定引数でサブプロセス実行
        *   stdout/stderr をキャプチャ
        *   タイムアウト付き
    *   フィクスチャ `cleanup_containers` (autouse, scope=session):
        *   セッション終了時に `docker rm -f` でテスト中に作成されたコンテナを削除
        *   `docker rmi` でテスト中に作成されたイメージを削除
        *   コンテナ名パターン: `*-integration-test`
*   **Logic**:
    *   テストスキップは一切行わない（testing-rules.md 遵守）
    *   前提条件未達の場合は `pytest.fail("reason")` でエラー終了

#### [NEW] [test_docker_build.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_docker_build.py)
*   **Description**: feature の Dockerfile が正常にビルドできるかを検証
*   **Technical Design**:
    *   `test_integration_test_dockerfile_builds`:
        *   `docker build -t integration-test-verify features/integration-test/.devcontainer` を実行
        *   exit code = 0 を assert
        *   完了後イメージを削除
    *   `test_devctl_dockerfile_builds`:
        *   `docker build -t devctl-verify features/devctl/.devcontainer` を実行
        *   exit code = 0 を assert
        *   **注意**: 現在のdevctl Dockerfileは golangci-lint のGoバージョン不整合でビルド失敗するため、Dockerfileの修正が前提
        *   完了後イメージを削除

#### [NEW] [test_devctl_up.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_up.py)
*   **Description**: `devctl up` でコンテナが起動するかを検証
*   **Technical Design**:
    *   `test_devctl_up_starts_container`:
        *   `run_devctl(["up", "integration-test", "--verbose"])` を実行
        *   exit code = 0 を assert
        *   `docker ps --filter name=*-integration-test --format '{{.Names}}'` でコンテナが起動していることを assert
    *   `test_devctl_up_idempotent`:
        *   `run_devctl(["up", "integration-test"])` を2回実行
        *   2回目も exit code = 0 を assert（既に起動中のためスキップされる挙動を確認）
    *   テスト後 `run_devctl(["down", "integration-test"])` でクリーンアップ

#### [NEW] [test_devctl_down.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_down.py)
*   **Description**: `devctl down` でコンテナが停止・削除されるかを検証
*   **Technical Design**:
    *   `test_devctl_down_stops_container`:
        *   前提: `run_devctl(["up", "integration-test"])` でコンテナを起動
        *   `run_devctl(["down", "integration-test", "--verbose"])` を実行
        *   exit code = 0 を assert
        *   `docker ps -a --filter name=*-integration-test` でコンテナが存在しないことを assert
    *   `test_devctl_down_noop_when_not_running`:
        *   コンテナが起動していない状態で `run_devctl(["down", "integration-test"])` を実行
        *   exit code = 0 を assert（エラーにならないことを確認）

#### [NEW] [test_devctl_status.py](file:///c:/Users/yamya/myprog/escape/tests/integration-test/test_devctl_status.py)
*   **Description**: `devctl status` が正しいステータスを返すかを検証
*   **Technical Design**:
    *   `test_devctl_status_when_running`:
        *   前提: `run_devctl(["up", "integration-test"])` でコンテナを起動
        *   `run_devctl(["status", "integration-test"])` を実行
        *   exit code = 0 を assert
        *   stdout に "running" 相当の表示があることを assert
        *   クリーンアップ: `run_devctl(["down", "integration-test"])`
    *   `test_devctl_status_when_stopped`:
        *   コンテナが起動していない状態で `run_devctl(["status", "integration-test"])` を実行
        *   exit code = 0 を assert

---

### integration_test.sh 改修

#### [MODIFY] [integration_test.sh](file:///c:/Users/yamya/myprog/escape/scripts/process/integration_test.sh)
*   **Description**: Pythonテスト（pytest）対応を追加。既存のGoテストロジックは維持
*   **Technical Design**:

    **1. `discover_categories()` の拡張 (L133-L152)**:
    *   現在: `*_test.go` が存在するサブディレクトリのみ検出
    *   変更後: `*_test.go` **または** `test_*.py` / `conftest.py` が存在するサブディレクトリを検出
    *   擬似コード:
        ```bash
        # Check for Go test files OR Python test files
        if ls "$dir"*_test.go 1>/dev/null 2>&1 || \
           ls "$dir"test_*.py 1>/dev/null 2>&1 || \
           ls "$dir"conftest.py 1>/dev/null 2>&1; then
            echo "$cat_name"
        fi
        ```

    **2. カテゴリの言語判定関数 `detect_category_lang()` を新規追加**:
    *   引数: カテゴリディレクトリパス
    *   戻り値: `"go"` or `"python"` を echo
    *   判定ロジック:
        *   `*_test.go` が存在 → `"go"`
        *   `test_*.py` が存在 → `"python"`
        *   両方存在 → `"go"`（Goを優先）

    **3. `run_category_tests()` の拡張 (L198-L237)**:
    *   言語判定を追加し、Go/Pythonで分岐:

    ```bash
    run_category_tests() {
        local category="$1"
        local test_dir="$PROJECT_ROOT/tests/$category"
        # ... (ディレクトリ存在チェックは既存のまま)

        local lang
        lang=$(detect_category_lang "$test_dir")

        if [[ "$lang" == "go" ]]; then
            run_go_tests "$category" "$test_dir"
        elif [[ "$lang" == "python" ]]; then
            run_python_tests "$category" "$test_dir"
        else
            warn "No test files in tests/$category — skipping"
            return 0
        fi
    }
    ```

    **4. `run_go_tests()` 関数の抽出**:
    *   既存の `run_category_tests()` 内のGoテスト実行ロジックをそのまま移動
    *   `go test -v -count=1 [-run "$SPECIFY"] <pkg_path>` を実行

    **5. `run_python_tests()` 関数の新規追加**:
    *   `pytest` コマンドでテストを実行
    *   `--specify` オプションは pytest の `-k` オプションにマッピング
    *   擬似コード:
        ```bash
        run_python_tests() {
            local category="$1"
            local test_dir="$2"

            step "Running Python integration tests: $category"

            local pytest_args=("-v" "--tb=short")
            if [[ -n "$SPECIFY" ]]; then
                pytest_args+=("-k" "$SPECIFY")
            fi
            pytest_args+=("$test_dir")

            if python -m pytest "${pytest_args[@]}"; then
                success "Category '$category' — all tests passed."
                return 0
            else
                fail "Category '$category' — tests FAILED."
                return 1
            fi
        }
        ```

    **6. `main()` のpre-flightチェック拡張 (L264-L269)**:
    *   現在: `go.mod` が無い場合にスキップ
    *   変更後: `go.mod` チェックを削除し、`tests/` ディレクトリの存在のみをチェック
    *   （Pythonテストのみの場合も実行可能にするため）

    **7. ヘルプテキスト更新 (L48-L83)**:
    *   `--specify` の説明を更新:「`go test -run` or `pytest -k`」
    *   Python対応の旨を追記

---

### devctl Dockerfile 修正

#### [MODIFY] [Dockerfile](file:///c:/Users/yamya/myprog/escape/features/devctl/.devcontainer/Dockerfile)
*   **Description**: golangci-lint のGoバージョン不整合を修正
*   **Technical Design**:
    *   変更: `FROM golang:1.22-bookworm` → `FROM golang:1.23-bookworm`
    *   これにより `golangci-lint@latest` (go >= 1.23.0 要求) のインストールが成功する

## Step-by-Step Implementation Guide

1.  **devctl Dockerfile のGoバージョン修正**:
    *   `features/devctl/.devcontainer/Dockerfile` を編集
    *   `FROM golang:1.22-bookworm` → `FROM golang:1.23-bookworm` に変更

2.  **integration-test feature ディレクトリ構造の作成**:
    *   `features/integration-test/.devcontainer/Dockerfile` を新規作成
    *   `features/integration-test/.devcontainer/devcontainer.json` を新規作成
    *   `features/integration-test/requirements.txt` を新規作成
    *   `features/integration-test/README.md` を新規作成

3.  **テストコードの作成（TDD: テスト先行）**:
    *   `tests/integration-test/conftest.py` を新規作成
    *   `tests/integration-test/test_docker_build.py` を新規作成
    *   `tests/integration-test/test_devctl_up.py` を新規作成
    *   `tests/integration-test/test_devctl_down.py` を新規作成
    *   `tests/integration-test/test_devctl_status.py` を新規作成

4.  **`integration_test.sh` の改修**:
    *   `detect_category_lang()` 関数を追加
    *   `discover_categories()` を拡張（Python検出対応）
    *   `run_category_tests()` を Go/Python 分岐に改修
    *   `run_go_tests()` を既存ロジックから抽出
    *   `run_python_tests()` を新規追加
    *   `main()` の pre-flight チェック緩和
    *   ヘルプテキスト更新

5.  **ビルドと統合テスト実行**:
    *   `build.sh` を実行してdevctlバイナリを生成
    *   `integration_test.sh` を実行して統合テストを検証

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    devctlバイナリの生成と単体テストの実行。
    ```bash
    ./scripts/process/build.sh
    ```

2.  **Integration Tests (全カテゴリ)**:
    Pythonテストカテゴリが自動検出され、pytest で実行されることを確認。
    ```bash
    ./scripts/process/integration_test.sh
    ```

3.  **Integration Tests (カテゴリ指定)**:
    特定カテゴリのみ実行できることを確認。
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test"
    ```

4.  **Integration Tests (テスト名指定)**:
    `--specify` で特定テストのみ実行できることを確認（`-k` にマッピング）。
    ```bash
    ./scripts/process/integration_test.sh --categories "integration-test" --specify "test_docker_build"
    ```

*   **Log Verification**:
    *   カテゴリ検出ログに `integration-test` が表示されること
    *   Python テストの実行ログに `pytest` のフォーマット出力が表示されること
    *   テスト結果サマリーに passed/failed カウントが正しく表示されること

## Documentation

#### [MODIFY] [features/README.md](file:///c:/Users/yamya/myprog/escape/features/README.md)
*   **更新内容**: `integration-test` featureの説明を追加
