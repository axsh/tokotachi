# 003 — 統合テスト環境 (Integration Test Infrastructure)

## 背景 (Background)

`devctl` CLIツールは、feature単位の開発環境オーケストレータとして `up`, `down`, `open`, `status`, `shell`, `exec`, `pr`, `close`, `list` サブコマンドを提供している。しかし、現時点でdevctlの動作を検証する統合テストが存在しない。

### 現在の課題

1. **統合テストが存在しない**: `devctl`のサブコマンドが実際に正しく動作するか、エンドツーエンドで確認する手段がない
2. **Dockerビルドの失敗**: 既存の `features/devctl/.devcontainer/Dockerfile` が `golang:1.22-bookworm` ベースで `golangci-lint@latest` をインストールしようとするが、golangci-lint最新版は `go >= 1.23.0` を要求するためビルドが失敗する
3. **`integration_test.sh` がGoテスト専用**: 現行の `scripts/process/integration_test.sh` は Go テストファイル (`*_test.go`) のみを探索・実行する設計であり、Pythonテストプロジェクトに対応していない

## 要件 (Requirements)

### 必須要件

1. **`features/integration-test/` フォルダの作成**
   - `features/` 直下に `integration-test/` フォルダを新規作成する
   - Pythonベースの統合テストプロジェクトを配置する

2. **`.devcontainer` による Python 開発環境の整備**
   - `features/integration-test/.devcontainer/` に以下を配置:
     - `devcontainer.json`: Python開発用のdevcontainer定義
     - `Dockerfile`: Python 3.12+ ベースのイメージ、Docker CLI含む
   - コンテナ内でPythonテストが実行できる環境を構築する

3. **`tests/` フォルダへのテストコード配置**
   - `features/integration-test/tests/` にテストファイルを配置する
   - `devctl` のサブコマンドが正しく動作することを検証するテスト群を作成

4. **`integration_test.sh` の改修**
   - 既存のGoテスト実行ロジックを維持しつつ、Pythonテストカテゴリも実行できるように拡張する
   - Pythonテストの場合は `pytest` を使用して実行する
   - カテゴリの自動検出ロジックを拡張し、`*_test.go` に加えて `test_*.py` / `*_test.py` も検出可能にする

### 任意要件

- テスト結果をJUnit XML形式で出力し、CI連携に備える
- テストのタイムアウト制御

## 実現方針 (Implementation Approach)

### ディレクトリ構造

```
features/
├── devctl/              # (既存) devctl本体
├── integration-test/    # (新規) 統合テストプロジェクト
│   ├── .devcontainer/
│   │   ├── devcontainer.json
│   │   └── Dockerfile
│   ├── tests/
│   │   ├── conftest.py          # pytest共通フィクスチャ
│   │   ├── test_devctl_up.py    # devctl up テスト
│   │   ├── test_devctl_down.py  # devctl down テスト
│   │   ├── test_devctl_status.py# devctl status テスト
│   │   └── test_docker_build.py # Dockerビルドテスト
│   ├── requirements.txt         # pytest, その他依存
│   └── README.md
```

### `.devcontainer` 構成

#### Dockerfile

```dockerfile
FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y \
    docker.io \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
```

#### devcontainer.json

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
            "extensions": [
                "ms-python.python",
                "ms-python.debugpy"
            ]
        }
    },
    "mounts": [
        "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
    ],
    "remoteUser": "root"
}
```

### テスト設計

#### 前提条件

- `devctl` バイナリがビルド済みで `bin/devctl` に存在すること
- Docker Engine が利用可能であること（Docker Socket マウント）
- テスト実行用のtemporaryなfeature/worktreeを使用

#### テストケース

| テスト | 内容 | 検証ポイント |
|---|---|---|
| `test_devctl_up.py` | `devctl up` でコンテナが起動するか | コンテナ起動、Docker イメージビルド成功 |
| `test_devctl_down.py` | `devctl down` でコンテナが停止・削除されるか | コンテナ停止、コンテナ削除 |
| `test_devctl_status.py` | `devctl status` が正しいステータスを返すか | 起動前/起動中の状態表示 |
| `test_docker_build.py` | feature の Dockerfile が正常にビルドできるか | ビルド成功（特にGo版の修正後） |

#### conftest.py（共通フィクスチャ）

- `devctl_binary`: `bin/devctl` のパスを提供
- `project_root`: プロジェクトルートパスを提供
- `temp_feature`: テスト用一時featureディレクトリの作成・クリーンアップ
- `docker_cleanup`: テスト後のDockerコンテナ/イメージのクリーンアップ

### `integration_test.sh` の改修方針

現行スクリプトの構造（カテゴリベースの自動検出・実行）を維持しつつ、以下を追加:

1. **カテゴリの言語判定**: 各カテゴリディレクトリ内のファイルを検査し、Go (`*_test.go`) と Python (`test_*.py` / `conftest.py`) を自動判別
2. **Python実行パス**: Pythonカテゴリの場合は `go test` の代わりに `pytest` を実行
3. **`--specify` オプション**: Pythonの場合は `-k` オプションにマッピング

```
discover_categories() の拡張:
  Go: *_test.go が存在 → cat_name を出力
  Python: test_*.py が存在 → cat_name を出力

run_category_tests() の拡張:
  if Goテストファイルが存在 → go test
  elif Pythonテストファイルが存在 → pytest
```

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: 統合テストフレームワークの基本動作

1. `features/integration-test/` フォルダが存在することを確認
2. `.devcontainer/Dockerfile` が正常にビルドできることを確認
3. `features/integration-test/tests/` に `conftest.py` と `test_*.py` ファイルが配置されていることを確認

### シナリオ 2: `integration_test.sh` のPythonテスト対応

1. `scripts/process/integration_test.sh` を引数なしで実行
2. `features/integration-test/tests/` のPythonテストカテゴリが自動検出されること
3. Pythonテストが `pytest` で実行されること
4. テスト結果サマリーが正しく表示されること

### シナリオ 3: devctl動作の検証テスト

1. `devctl up integration-test` が成功すること（Dockerビルド + コンテナ起動）
2. `devctl status integration-test` が起動中ステータスを返すこと
3. `devctl down integration-test` がコンテナを停止・削除すること

## テスト項目 (Testing for the Requirements)

### 自動化テスト

| 要件 | 検証スクリプト | テストケース |
|---|---|---|
| 統合テストプロジェクト構造 | `scripts/process/integration_test.sh` | Pythonカテゴリの自動検出・実行 |
| devctl upの動作 | `scripts/process/integration_test.sh` | `test_devctl_up.py` |
| devctl downの動作 | `scripts/process/integration_test.sh` | `test_devctl_down.py` |
| devctl statusの動作 | `scripts/process/integration_test.sh` | `test_devctl_status.py` |
| Dockerfileビルド | `scripts/process/integration_test.sh` | `test_docker_build.py` |

### 検証コマンド

```bash
# ビルド（devctlバイナリの生成）
./scripts/process/build.sh

# 全統合テスト実行
./scripts/process/integration_test.sh

# 特定カテゴリのみ実行
./scripts/process/integration_test.sh --categories "integration-test"

# 特定テストのみ実行
./scripts/process/integration_test.sh --categories "integration-test" --specify "test_devctl_up"
```

> [!NOTE]
> 統合テストはDocker環境が必要です。Docker Desktopが起動していることを前提とします。
