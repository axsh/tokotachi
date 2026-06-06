# 011 — Close操作のエラー耐性改善

## 背景 (Background)

`devctl close` は以下の4ステップを順次実行するが、各ステップのエラー処理方針が一貫していない:

| Step | 操作 | 現在のエラー処理 | 問題 |
|---|---|---|---|
| 1 | Container down | `[WARN]` → 続行 ✅ | なし |
| 2 | Worktree remove | `return err` → **即座に失敗** ❌ | Step 3-4が実行されない |
| 3 | Branch delete | `[WARN]` → 続行 ✅ | なし |
| 4 | State file remove | `[WARN]` → 続行 ✅ | なし |

**再現シナリオ:**
1. `devctl close devctl test-003` → worktreeフォルダがロックされて `Permission denied`
2. git worktreeメタデータは解除されるが、ディレクトリが残る
3. 再度 `devctl close devctl test-003` → `wm.Exists()` が `true`（ディレクトリ存在）→ `git worktree remove` → `'...' is not a working tree` で失敗
4. Step 3-4（ブランチ削除、stateファイル削除）が**実行されず**、リソースが残留する

## 要件 (Requirements)

### 必須要件

1. **`close.go` の worktree 削除失敗を許容エラーに変更**
   - `git worktree remove` が失敗しても、`[WARN]` を出力して Step 3-4 を続行する
   - worktree ディレクトリが残留した場合は、`os.RemoveAll` でフォールバック削除を試みる

2. **ログ出力の一貫性**
   - `close` の全4ステップで、エラーを「許容エラー（`[WARN]`）」として扱い、最終的に全ステップを実行する
   - 全ステップが成功した場合のみ `SUCCESS`、1つ以上失敗した場合は `PARTIAL`（ただし全ステップは実行する）

3. **`git worktree remove` のログレベル改善**
   - `worktree.go` の `Remove()` 内の `m.CmdRunner.Run()` を `RunWithOpts(ToleratedOpt())` に変更（`close` 文脈では失敗が許容されるため）

## 実現方針 (Implementation Approach)

### `close.go` の改修

```go
// Step 2: Remove worktree
if wm.Exists(opts.Feature, opts.Branch) {
    r.Logger.Info("Removing worktree work/%s/%s...", opts.Feature, opts.Branch)
    if err := wm.Remove(opts.Feature, opts.Branch, opts.Force); err != nil {
        r.Logger.Warn("Worktree remove failed: %v", err)
        // Fallback: try removing the directory directly
        wtPath := wm.Path(opts.Feature, opts.Branch)
        if removeErr := os.RemoveAll(wtPath); removeErr != nil {
            r.Logger.Warn("Directory cleanup also failed: %v", removeErr)
        } else {
            r.Logger.Info("Cleaned up worktree directory directly")
        }
    }
}
```

### `worktree.go` の改修（任意）

`Remove()` 内の `git worktree remove` コマンドを `ToleratedOpt()` に変更する（close 以外の文脈でも使われる可能性があるため、こちらは任意）。

## 検証シナリオ (Verification Scenarios)

1. 通常の `devctl close devctl test-xxx` が正常に完了する
2. worktree ディレクトリがロックされている場合（例: エディタが開いている）:
   - `git worktree remove` が失敗 → `[WARN]` 表示
   - `os.RemoveAll` でフォールバック削除を試行
   - Step 3-4（branch delete, state file remove）が**実行される**
   - 最終結果は `SUCCESS`（全リソースが最終的に削除された場合）
3. worktree ディレクトリが git worktree として認識されない場合（`is not a working tree`）:
   - `git worktree remove` が失敗 → `[WARN]` 表示
   - `os.RemoveAll` で直接削除
   - Step 3-4が実行される

## テスト項目 (Testing for the Requirements)

| 要件 | 検証コマンド |
|---|---|
| ビルド成功 | `./scripts/process/build.sh` |
| ユニットテスト | `./scripts/process/build.sh` (内包) |
| 統合テスト | `./scripts/process/integration_test.sh --categories "integration-test"` |
