# Dev Container 内で Git Worktree が機能しない問題の修正

## 背景 (Background)

### 現在の構成

`devctl` は git worktree を利用して、ブランチごとに独立した作業ディレクトリ (`work/<feature>/<branch>`) を作成する。
git worktree では、作業ディレクトリ内の `.git` は**ディレクトリではなくファイル**であり、親リポジトリの `.git/worktrees/<name>` を参照する:

```
# work/devctl/fix-git/.git (ファイル・63バイト)
gitdir: C:/Users/yamya/myprog/tokotachi/.git/worktrees/fix-git
```

worktree 参照先のディレクトリ構造:

```
C:/Users/yamya/myprog/tokotachi/.git/worktrees/fix-git/
├── commondir   # "../.." → 親リポの .git を参照
├── gitdir      # worktree の .git ファイルへの逆参照
├── HEAD
├── index
├── FETCH_HEAD
├── ORIG_HEAD
└── logs/
```

### 問題

`devctl up` でコンテナを起動する際、[up.go](file:///c:/Users/yamya/myprog/tokotachi/work/devctl/fix-git/features/devctl/internal/action/up.go) は worktreePath のみを `/workspace` にマウントする:

```go
"-v", opts.WorktreePath + ":" + wsFolder,
```

VSCode/Cursor が Dev Container に接続後、`/workspace/.git` ファイルを読むと `gitdir: C:/Users/yamya/myprog/tokotachi/.git/worktrees/fix-git` が記載されているが、このホスト側パスはコンテナ内に存在しない。そのため:

1. **`git status`, `git commit` 等のGitコマンドが失敗する**
2. **VSCode/Cursor の Source Control パネルが機能しない**
3. **エディタからコミット・プッシュができない**

### 問題の構造図

```mermaid
graph LR
    subgraph ホスト
        A[work/devctl/fix-git/] -->|".git ファイル"| B[.git/worktrees/fix-git/]
        B -->|commondir: "../.."| C[.git/]
        C -->|objects, refs, etc.| D[Git データ]
    end
    subgraph コンテナ
        E["/workspace/"] -->|".git ファイル"| F["gitdir: C:/.../.git/worktrees/fix-git"]
        F -->|"❌ パス不在"| G["エラー: Git が動作しない"]
    end
    A -.->|"-v マウント"| E
```

## 要件 (Requirements)

### 必須要件

1. **R1**: Dev Container 起動時に、コンテナ内で `git status`, `git commit`, `git push` 等の基本的な Git コマンドが正常に動作すること
2. **R2**: git worktree 構成（`.git` がファイルの場合）と通常の git 構成（`.git` がディレクトリの場合）の両方に対応すること
3. **R3**: 既存の動作（monorepo 内の worktree マウント）を壊さないこと
4. **R4**: VSCode/Cursor の Source Control パネルでコミット・プッシュ操作が可能であること

### 任意要件

5. **R5**: devcontainer.json に追加の設定なしで自動的に対応すること（ゼロコンフィグ）

## 実現方針 (Implementation Approach)

### 方針: ホスト側の `.git` 関連ディレクトリをコンテナ内にマウントする

git worktree の場合、コンテナ内のGitが動作するために以下のマウントが必要:

1. **親リポの `.git/` ディレクトリ全体**: objects, refs, packed-refs 等のGitデータ本体
2. **worktree メタデータディレクトリ**: `.git/worktrees/<name>/` に含まれる HEAD, index 等

さらに、`.git` ファイル内のパスをコンテナ内で有効なパスに書き換える必要がある。

### 実装手順の概要

#### 1. git worktree 検出ロジックの追加

新規関数 `resolve.DetectGitWorktree(worktreePath string)` を作成:

- `.git` がファイルかどうかを確認
- ファイルの場合、`gitdir:` の値を読み取り、worktree メタデータディレクトリのパスを取得
- `commondir` ファイルを読み取り、親リポの `.git/` ディレクトリのパスを算出
- 検出結果を構造体として返す

```go
type GitWorktreeInfo struct {
    IsWorktree       bool   // worktree構成かどうか
    WorktreeGitDir   string // .git/worktrees/<name>/ の絶対パス
    MainGitDir       string // 親リポの .git/ の絶対パス
}
```

#### 2. `action.UpOptions` に Git マウント情報を追加

```go
type UpOptions struct {
    // ... 既存フィールド ...
    GitWorktree *resolve.GitWorktreeInfo // nil なら通常の git 構成
}
```

#### 3. `action.Up()` でのマウント追加

git worktree が検出された場合、以下のマウントを `docker run` に追加:

```
# 親リポの .git/ をコンテナ内にマウント (read-write)
-v <MainGitDir>:/repo-git:rw

# worktree メタデータをコンテナ内にマウント (read-write)
-v <WorktreeGitDir>:/worktree-git:rw
```

#### 4. コンテナ内の `.git` ファイル書き換え

コンテナ起動後、`docker exec` で `.git` ファイルの `gitdir:` パスをコンテナ内のパスに書き換える:

```bash
echo "gitdir: /worktree-git" > /workspace/.git
```

同様に、worktree メタデータ内の `commondir` ファイルもコンテナ内パスに書き換える:

```bash
echo "/repo-git" > /worktree-git/commondir
```

また、`gitdir` ファイル（逆参照）も更新:

```bash
echo "/workspace/.git" > /worktree-git/gitdir
```

### 変更対象ファイル

| ファイル | 変更内容 |
|---------|---------|
| `internal/resolve/gitworktree.go` | **[新規]** git worktree 検出ロジック |
| `internal/resolve/gitworktree_test.go` | **[新規]** 検出ロジックのテスト |
| `internal/action/up.go` | マウント追加 + コンテナ内 `.git` パス書き換え |
| `cmd/up.go` | worktree 検出結果を `UpOptions` に渡す |

### 代替案の検討と棄却理由

| 代替案 | 棄却理由 |
|-------|---------|
| リポ全体をマウント | worktree 構成の意味がなくなる、パフォーマンス劣化 |
| `--no-worktree` で通常 clone | ブランチ切り替えが必要になり UX 悪化 |
| ボリュームに git clone | ホストとの同期が複雑 |

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: worktree 構成での Git 動作確認

1. `devctl up devctl` でコンテナを起動する
2. `devctl exec devctl -- git status` でGitステータスが表示されることを確認
3. `devctl exec devctl -- git log --oneline -5` でコミット履歴が表示されることを確認
4. `devctl exec devctl -- touch /workspace/test-file && git add test-file && git status` でステージングが動作することを確認
5. テスト用ファイルをクリーンアップ: `devctl exec devctl -- rm /workspace/test-file`

### シナリオ 2: 通常の git 構成での後方互換確認

1. `.git` がディレクトリであるリポジトリでコンテナを起動
2. `git status` が正常に動作することを確認

### シナリオ 3: VSCode/Cursor での動作確認（手動）

1. `devctl up devctl --editor code` で VSCode を起動
2. Source Control パネルに変更が表示されることを確認
3. ファイルを変更し、Source Control パネルからコミットできることを確認

## テスト項目 (Testing for the Requirements)

### 自動テスト

| 要件 | テスト方法 | 検証コマンド |
|------|----------|------------|
| R1, R2 | `internal/resolve/gitworktree_test.go` で worktree 検出ロジックの単体テスト | `scripts/process/build.sh` |
| R2 | `.git` がディレクトリの場合のテスト（`IsWorktree=false` が返る） | `scripts/process/build.sh` |
| R3 | 既存テストケースが引き続きパスすること | `scripts/process/build.sh` |
| R1, R4 | 統合テスト: コンテナ起動後に `git status` が成功することを確認 | `scripts/process/integration_test.sh` |

### 手動テスト

| 要件 | テスト方法 |
|------|----------|
| R4 | VSCode/Cursor で Dev Container に接続し、Source Control パネルからコミット操作が可能か確認 |
