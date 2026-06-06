# 004 — 統合テストコードのGo化 (Integration Test Go Migration)

## 背景 (Background)

003-IntegrationTest 仕様に基づき `tests/integration-test/` に統合テストを配置したが、テストコードがPythonで記述されていた。Go言語のほうが開発環境に対する依存ライブラリの問題が少なくクリーンであるため、テストコードをGoに変更する。

`features/integration-test/` はPythonプロジェクトのまま維持する（Go以外のプロジェクトでもdevctlが動作するかの検証目的）。

## 要件 (Requirements)

### 必須要件

1. **`tests/integration-test/` のPythonコードをGoに置き換える**
   - `conftest.py`, `test_*.py` を削除
   - 同等のテスト内容を `*_test.go` ファイルで再実装
   - 独自の `go.mod` を配置（devctlモジュールとは独立）

2. **テストケースの維持**
   - Dockerfileビルド検証、devctl up/down/status のE2Eテストを維持
   - `os/exec` でdevctlバイナリとdockerコマンドを呼び出す方式

3. **`integration_test.sh` との互換性**
   - Goテストカテゴリとして検出・実行されること（`*_test.go` 検出で既に対応済み）
   - Pythonの `integration_test.sh` 拡張はそのまま残す（`features/integration-test/` のPython環境テストに将来利用可能）

### 任意要件
- テストスキップ禁止（testing-rules.md: `t.Fatalf()` を使用）

## 実現方針 (Implementation Approach)

### Go テストモジュール構造

```
tests/integration-test/
├── go.mod                           # 独立モジュール
├── go.sum
├── helpers_test.go                  # 共通ヘルパー（旧conftest.py相当）
├── docker_build_test.go             # Dockerfileビルド検証
├── devctl_up_test.go                # devctl up テスト
├── devctl_down_test.go              # devctl down テスト
└── devctl_status_test.go            # devctl status テスト
```

### helpers_test.go

- `projectRoot()`: テストファイルから2階層上のプロジェクトルート
- `devctlBinary()`: `bin/devctl` (or `.exe`) のパス、存在しない場合は `t.Fatalf()`
- `requireDockerAvailable(t)`: `docker info` 成功の確認、失敗時 `t.Fatalf()`
- `runDevctl(t, args...)`: devctlバイナリ呼び出しヘルパー
- `dockerRun(args...)`: docker CLI呼び出しヘルパー
- `TestMain(m)`: テスト終了後にコンテナ/イメージのクリーンアップ

## 検証シナリオ (Verification Scenarios)

1. `tests/integration-test/` に `*_test.go` ファイルのみが存在すること（Python不在）
2. `./scripts/process/build.sh` が成功すること
3. `./scripts/process/integration_test.sh --categories "integration-test"` がGoテストとして実行されること
4. テスト結果に `go test` 形式の出力が含まれること

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| ビルド成功 | `./scripts/process/build.sh` |
| 統合テスト実行 | `./scripts/process/integration_test.sh --categories "integration-test"` |
| 特定テスト実行 | `./scripts/process/integration_test.sh --categories "integration-test" --specify "TestDockerBuild"` |
