# 006 — ログレベルとエラー表現の改善

## 背景 (Background)

`devctl` のコマンド実行基盤（`cmdexec.Runner`）は、外部コマンド（docker, git等）の失敗を一律に `[ERROR] [FAIL]` としてログ出力する。しかし、呼び出し元のアクション（`up`, `down`, `close` 等）では、コマンドの失敗を正常フローの一部として扱うケースが多い。

**現在の問題例:**

```
$ devctl close devctl test-001 --verbose
[INFO] [CMD] docker inspect --format {{.State.Running}} devctl-devctl
Error: No such object: devctl-devctl
[ERROR] [FAIL] docker inspect --format {{.State.Running}} devctl-devctl (exit=1)
  ← ユーザーの視点: 「エラー？ 失敗してるの？」

[INFO] Removing worktree work/devctl/test-001...

[INFO] [CMD] git branch -d test-001
error: Cannot delete branch 'test-001' checked out at '...'
[ERROR] [FAIL] git branch -d test-001 (exit=1)
  ← ユーザーの視点: 「またエラー？」
[WARN] Branch delete failed: ...

## Result: **SUCCESS**
  ← ユーザーの視点: 「ERRORが2つもあるのにSUCCESS？バグ？」
```

ユーザーにとって、以下の区別がつかない:
1. **致命的エラー（Fatal）**: プロセス全体が失敗する真のエラー
2. **条件チェック（Check）**: エラー応答を期待して行う存在確認等（例: `docker inspect` でコンテナの有無確認）
3. **許容されるエラー（Tolerated）**: 失敗したがプロセスは続行する（例: ブランチ削除失敗）

## 要件 (Requirements)

### 必須要件

1. **コマンド実行時のログレベルの柔軟化**
   - `cmdexec.Runner` のコマンド実行メソッドに、呼び出し元が「失敗時のログレベル」を指定できる仕組みを導入する
   - 条件チェック用途のコマンドは `[DEBUG]` レベルで失敗をログ出力する
   - 許容されるエラーは `[WARN]` レベルで失敗をログ出力する
   - 未指定の場合は従来通り `[ERROR]` レベル（後方互換性）

2. **呼び出し元のアクションコードの改修**
   - 条件チェック用途のコマンド呼び出しを、新しいAPIに移行する
   - 具体例:
     - `docker inspect` によるコンテナ存在確認 → 失敗は `[DEBUG]`
     - `docker image inspect` によるイメージ存在確認 → 失敗は `[DEBUG]`
     - `docker stop` の失敗（既に停止している場合） → 失敗は `[WARN]`
     - `docker rm` の失敗（既に削除されている場合） → 失敗は `[WARN]`
     - `git branch -d` の失敗 → 失敗は `[WARN]`

3. **ログ出力形式の改善**
   - `[ERROR] [FAIL]` という冗長な二重タグを簡潔にする
   - 成功時は `[OK]`、失敗時はレベルに応じたタグで出力する
   - 例: 条件チェック失敗 → `[DEBUG] [SKIP] docker inspect ... (not found)` 
   - 例: 許容エラー → `[WARN] docker stop ... (exit=1, may already be stopped)`
   - 例: 致命的エラー → `[ERROR] docker build ... (exit=1)`

### 任意要件

- `--verbose` なしの通常表示では、条件チェック系のコマンド実行ログ自体を非表示にする（ユーザーにとってノイズになるため）

## 実現方針 (Implementation Approach)

### 方針: Run/RunInteractive にオプション引数を追加

`cmdexec.Runner` の `Run()` / `RunInteractive()` のシグネチャをそのまま維持しつつ、新しい「エラーレベルを指定できるバリアント」を追加する。

```go
// 既存（後方互換、err時は[ERROR]）
func (r *Runner) Run(name string, args ...string) (string, error)

// 新規: 失敗時のログレベルとラベルを指定可能
type RunOption struct {
    FailLevel log.Level   // 失敗時のログレベル (default: LevelError)
    FailLabel string      // 失敗時のラベル (default: "FAIL", 例: "SKIP")
}

func (r *Runner) RunWithOpts(opts RunOption, name string, args ...string) (string, error)
func (r *Runner) RunInteractiveWithOpts(opts RunOption, name string, args ...string) error
```

### 呼び出し元の改修パターン

```go
// Before:
out, err := r.DockerRunOutput("inspect", "--format", "{{.State.Running}}", containerName)

// After:
out, err := r.DockerRunOutputCheck("inspect", "--format", "{{.State.Running}}", containerName)
// DockerRunOutputCheck は失敗を [DEBUG] [SKIP] でログ
```

`action.Runner` にラッパーメソッドを追加:
- `DockerRunCheck()` — 条件チェック用（失敗は `[DEBUG]`）
- `DockerRunTolerated()` — 許容エラー用（失敗は `[WARN]`）

## 検証シナリオ (Verification Scenarios)

1. `devctl up devctl test-001 --verbose` 実行時:
   - `docker inspect` 失敗（コンテナ不存在）→ `[DEBUG]` レベルで表示
   - `docker image inspect` 失敗（イメージ不存在）→ `[DEBUG]` レベルで表示
   - 最終的に SUCCESS

2. `devctl down devctl test-001 --verbose` 実行時:
   - `docker stop` 失敗（既に停止）→ `[WARN]` レベルで表示
   - `docker rm` 失敗（既に削除）→ `[WARN]` レベルで表示

3. `devctl close devctl test-001 --verbose` 実行時:
   - `docker inspect` 失敗 → `[DEBUG]` レベル
   - `git branch -d` 失敗 → `[WARN]` レベル
   - 結果 SUCCESS — ユーザーに矛盾感を与えない

4. 致命的エラー発生時:
   - `docker build` 失敗 → `[ERROR]` レベルで表示
   - 結果 FAILED

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| ビルド成功 | `./scripts/process/build.sh` |
| ユニットテスト | `./scripts/process/build.sh` (内包) |
| 統合テスト | `./scripts/process/integration_test.sh --categories "integration-test"` |
