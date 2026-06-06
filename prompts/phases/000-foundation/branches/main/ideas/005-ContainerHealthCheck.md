# 005 — コンテナ起動検知とDockerfile修正

## 背景 (Background)

`devctl up` で `docker run -d` を実行後、コンテナが実際に稼働しているかを確認せずに "started successfully" と表示する。`CMD` が定義されていないDockerfileのコンテナは即座に `Exited (0)` で終了するが、`devctl` はこれを検知できず、さらにエディタのDevcontainer Attachまで実行してしまう。

実際の事象:
1. `devctl up devctl test-001 --verbose --editor cursor` を実行
2. "Container devctl-devctl started successfully" と表示
3. エディタが起動するがコンテナは既に停止済み → 開発環境が使えない

## 要件 (Requirements)

### 必須要件

1. **コンテナ起動後のヘルスチェック**
   - `docker run -d` 完了後、一定時間待機してからコンテナが `running` 状態であることを確認
   - 確認方法: `docker inspect --format {{.State.Running}} <container>` が `true` を返すこと
   - 待機時間: 2秒（コンテナが即座にexitするケースの検出に十分）
   - コンテナが停止している場合、エラーメッセージを表示してエラーを返す

2. **エディタ起動の抑制**
   - ヘルスチェックが失敗した場合、`Up()` がエラーを返し、呼び出し元（`cmd/up.go`）がエディタを起動しない

3. **devctl DockerfileにCMD追加**
   - `features/devctl/.devcontainer/Dockerfile` に `CMD ["sleep", "infinity"]` を追加
   - これによりコンテナが永続的に稼働する

### 任意要件
- エラーメッセージに `docker logs <container>` の出力を含めて原因の特定を支援

## 実現方針 (Implementation Approach)

### `action/up.go` の修正

`Up()` 関数の `docker run -d` 成功後（L94-98の間）に以下を追加:
1. `time.Sleep(2 * time.Second)` で短時間待機
2. `r.Status(opts.ContainerName, "")` で状態確認
3. `StateContainerRunning` でなければ、`docker logs` を取得してエラーを返す

### `features/devctl/.devcontainer/Dockerfile` の修正

末尾に `CMD ["sleep", "infinity"]` を追加。

## 検証シナリオ (Verification Scenarios)

1. CMDなしのDockerfileを持つfeatureで `devctl up` → エラーが表示され、エディタは起動しない
2. CMDありのDockerfileを持つfeatureで `devctl up` → コンテナが起動し、"started successfully" と表示
3. 修正後の devctl Dockerfile で `devctl up devctl test-001` → コンテナが稼働し続ける
4. 既存の統合テスト（`TestDevctlUpStartsContainer` 等）が引き続きPASS

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| ビルド成功 | `./scripts/process/build.sh` |
| 統合テスト（全カテゴリ） | `./scripts/process/integration_test.sh --categories "integration-test"` |
| コンテナ停止確認 | `devctl up devctl test-001 --verbose` 後に `docker ps` でコンテナが running であること |
