# devctl close → up ライフサイクル修正

## 背景 (Background)

`devctl up <branch>` → `devctl pr` → `devctl close <branch>` → 再度 `devctl up <branch>` というサイクルを繰り返すと、以下の問題が発生する可能性がある:

1. **`close` 後に空ディレクトリが残存する**: `git worktree remove` が成功しても、Gitの既知の挙動として空のディレクトリが残る場合がある。`close` のフォールバック (`os.RemoveAll`) は `git worktree remove` が **失敗** した場合のみ実行されるため、成功後に空ディレクトリが残ったケースはカバーされない。

2. **`wm.Exists()` がゴーストディレクトリを正常と誤判定**: `Exists()` は `os.Stat().IsDir()` のみで判定しており、そのディレクトリが実際にGit worktreeとして正常かどうかを確認しない。空ディレクトリが残存していると `true` を返し、worktree作成がスキップされる。

3. **`resolve.Worktree()` も同様の問題**: ディレクトリの存在のみチェックしており、有効なworktreeかどうかを検証していない。

4. **結果として「成功」と報告されるが中身は空**: ユーザーには `SUCCESS` と報告されるが、実際にはworktreeディレクトリは空のまま。作業ファイルが存在しない状態でエディタが起動される。

### 現在の実装

| ファイル | 関数 | 問題 |
|---|---|---|
| [`worktree.go`](file://features/devctl/internal/worktree/worktree.go) | `Manager.Exists()` | `os.Stat().IsDir()` のみでチェック |
| [`worktree.go`](file://features/devctl/internal/worktree/worktree.go) | `Manager.Create()` | 呼び出し前に `Exists()` でスキップ判定されるため到達しない |
| [`worktree.go`](file://features/devctl/internal/worktree/worktree.go) | `Manager.Remove()` | 成功後に空ディレクトリが残った場合の後処理がない |
| [`close.go`](file://features/devctl/internal/action/close.go) | `Runner.Close()` | `Remove()` 成功後のディレクトリ残存チェックがない |
| [`worktree.go`](file://features/devctl/internal/resolve/worktree.go) | `resolve.Worktree()` | `os.Stat().IsDir()` のみでチェック |

---

## 要件 (Requirements)

### 必須要件

1. **`wm.Exists()` の強化**: ディレクトリの存在だけでなく、そのディレクトリが有効なGit worktreeであることを検証する。具体的には、ディレクトリ内に `.git` ファイル（通常ファイルとして存在し、worktreeのメタデータへのパスを含む）が存在することを確認する。

2. **`wm.Create()` の前処理**: 空ディレクトリや不正なworktreeディレクトリが存在する場合、自動的に削除してからworktreeを作成する。

3. **`close` 後のディレクトリ残存防止**: `wm.Remove()` 成功後に、ディレクトリがまだ存在する場合は明示的に削除する。

4. **`resolve.Worktree()` の検証強化**: ディレクトリの存在だけでなく、`.git` ファイルまたはディレクトリの存在を確認する。

### 任意要件

- なし（リモートブランチの削除については今回のスコープ外とする）

---

## 実現方針 (Implementation Approach)

### 1. `worktree.Manager` の修正

#### 1.1 `Exists()` に有効性チェックを追加

```go
// Exists checks if the worktree directory exists and is a valid git worktree.
func (m *Manager) Exists(branch string) bool {
    wtPath := m.Path(branch)
    info, err := os.Stat(wtPath)
    if err != nil || !info.IsDir() {
        return false
    }
    // Check for .git file (worktrees have a .git file, not a directory)
    gitPath := filepath.Join(wtPath, ".git")
    _, err = os.Stat(gitPath)
    return err == nil
}
```

#### 1.2 `Create()` にゴーストディレクトリの前処理を追加

`Create()` の先頭で、ディレクトリが存在するがGit worktreeとして無効な場合（`.git` ファイルがない場合）、そのディレクトリを削除する。

```go
func (m *Manager) Create(branch string) error {
    wtPath := m.Path(branch)
    
    // Clean up ghost directory: exists but not a valid worktree
    if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
        gitPath := filepath.Join(wtPath, ".git")
        if _, gitErr := os.Stat(gitPath); os.IsNotExist(gitErr) {
            // Ghost directory — remove before creating
            os.RemoveAll(wtPath)
        }
    }
    
    // ... existing logic ...
}
```

#### 1.3 `Remove()` の後処理強化

`Remove()` の最後で、ディレクトリがまだ残っている場合は `os.RemoveAll` で明示的に削除する。

```go
func (m *Manager) Remove(branch string, force bool) error {
    wtPath := m.Path(branch)
    // ... existing git worktree remove logic ...
    
    // Ensure directory is fully removed (handle empty directory remnants)
    if _, err := os.Stat(wtPath); err == nil {
        os.RemoveAll(wtPath)
    }
    return nil
}
```

### 2. `resolve.Worktree()` の修正

ディレクトリ内に `.git` ファイルまたは `.git` ディレクトリが存在することを追加チェックする。

```go
func Worktree(repoRoot, branch string) (string, error) {
    path := filepath.Join(repoRoot, "work", branch)
    if info, err := os.Stat(path); err == nil && info.IsDir() {
        // Validate: must have .git file (worktree) or .git directory
        gitPath := filepath.Join(path, ".git")
        if _, gitErr := os.Stat(gitPath); gitErr == nil {
            return path, nil
        }
        return "", fmt.Errorf("worktree for branch %q exists but is not a valid git worktree (ghost directory)", branch)
    }
    return "", fmt.Errorf("worktree for branch %q not found", branch)
}
```

### 3. `close.go` の修正

`wm.Remove()` の後処理強化が `worktree.go` 側に移るため、`close.go` 側のフォールバックロジックは現状維持で十分。`Remove()` の戻り値がエラーの場合のフォールバック `os.RemoveAll` は残す。

### 修正対象ファイル一覧

| ファイル | 変更内容 |
|---|---|
| `features/devctl/internal/worktree/worktree.go` | `Exists()`, `Create()`, `Remove()` の修正 |
| `features/devctl/internal/resolve/worktree.go` | `Worktree()` の検証強化 |
| `features/devctl/internal/worktree/worktree_test.go` | 新規テストケース追加 |
| `features/devctl/internal/resolve/worktree_test.go` | 新規テストケース追加 |

---

## 検証シナリオ (Verification Scenarios)

### シナリオ1: close後に空ディレクトリが残っている状態でのup

1. `devctl up test-branch` で初回worktree作成
2. `devctl close test-branch` でworktreeを削除
3. 手動で `work/test-branch/` 空ディレクトリを作成（ゴーストディレクトリの再現）
4. `devctl up test-branch` を再実行
5. **期待**: ゴーストディレクトリが自動削除され、新規worktreeが正常に作成される

### シナリオ2: close後にディレクトリが完全に削除された状態でのup

1. `devctl up test-branch` で初回worktree作成
2. `devctl close test-branch` でworktreeを削除
3. `work/test-branch/` ディレクトリが存在しないことを確認
4. `devctl up test-branch` を再実行
5. **期待**: 新規worktreeが正常に作成される（リグレッションなし）

### シナリオ3: Exists()がゴーストディレクトリをfalseと判定する

1. 空のディレクトリ `work/test-branch/` を作成（`.git` ファイルなし）
2. `wm.Exists("test-branch")` を呼び出す
3. **期待**: `false` を返す

### シナリオ4: Exists()が有効なworktreeをtrueと判定する

1. ディレクトリ `work/test-branch/` を作成
2. 内部に `.git` ファイルを作成
3. `wm.Exists("test-branch")` を呼び出す
4. **期待**: `true` を返す

### シナリオ5: resolve.Worktree()がゴーストディレクトリでエラーを返す

1. 空のディレクトリ `work/test-branch/` を作成（`.git` ファイルなし）
2. `resolve.Worktree(root, "test-branch")` を呼び出す
3. **期待**: ゴーストディレクトリを示すエラーメッセージを返す

### シナリオ6: Remove()後にディレクトリが完全削除される

1. `work/test-branch/` ディレクトリを作成
2. `wm.Remove("test-branch", false)` を呼び出す
3. **期待**: ディレクトリが完全に削除される（空ディレクトリが残らない）

---

## テスト項目 (Testing for the Requirements)

### 単体テスト

以下のテストケースを追加・修正する。すべて `scripts/process/build.sh` で検証可能。

#### `worktree_test.go` に追加するテスト

| テスト名 | 検証内容 |
|---|---|
| `TestExists_GhostDirectory` | `.git` ファイルがない空ディレクトリに対して `Exists()` が `false` を返す |
| `TestExists_ValidWorktree` | `.git` ファイルがあるディレクトリに対して `Exists()` が `true` を返す |
| `TestCreate_CleansGhostDirectory` | 空ディレクトリが存在する場合、`Create()` がそれを削除してからworktreeを作成する（dry-runでコマンド記録を確認） |
| `TestRemove_CleansRemainingDirectory` | `Remove()` 後にディレクトリが残っていた場合、自動削除される |

#### `resolve/worktree_test.go` に追加するテスト

| テスト名 | 検証内容 |
|---|---|
| `TestResolveWorktree_GhostDirectory` | `.git` ファイルがない空ディレクトリに対してエラーを返す |
| `TestResolveWorktree_ValidWithGitFile` | `.git` ファイルがあるディレクトリに対して正常にパスを返す |

### 検証コマンド

```bash
# 全体ビルド & 単体テスト
scripts/process/build.sh
```
