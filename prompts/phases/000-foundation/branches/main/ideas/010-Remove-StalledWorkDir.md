# 010: `features/devctl/work/` ディレクトリの削除

## 背景 (Background)

`features/devctl/work/` ディレクトリがGitリポジトリに追跡されている。このディレクトリには以下のファイルが含まれている:

```
features/devctl/work/devctl/test-001.state.yaml
```

### 調査結果

以下の調査により、このディレクトリは**不要**であると判断した。

| 調査項目 | 結果 |
|---|---|
| コード内の `features/devctl/work` への直接参照 | **なし** (0件) |
| コードが使用する `work/` パス | リポジトリルート直下の `work/<feature>/<branch>` |
| `.gitignore` のカバー範囲 | ルートの `work/*` のみ。`features/devctl/work/` は対象外 |
| Git追跡状況 | `features/devctl/work/devctl/test-001.state.yaml` が**追跡済み** |
| ファイル内容 | `test-001` のstate YAMLファイル（開発中のテストデータ） |

### 根本原因

`devctl` の state 管理コード (`internal/state/state.go`) は、state ファイルを `work/<feature>/<branch>.state.yaml` (リポジトリルート基準) に保存する:

```go
func StatePath(repoRoot, feature, branch string) string {
    return filepath.Join(repoRoot, "work", feature, branch+".state.yaml")
}
```

`features/devctl/work/` にファイルが存在するのは、開発中に `features/devctl/` をリポジトリルートと誤認した状態でコマンドを実行し、その結果が誤ってコミットされたものと推測される。

## 要件 (Requirements)

### 必須要件

1. **ディレクトリ削除**: `features/devctl/work/` ディレクトリとその中の全ファイルをGitリポジトリから削除する
2. **`.gitignore` の補強**: `features/devctl/work/` 配下が再び追跡されないよう、`features/devctl/.gitignore` に `work/` パターンを追加する

### 任意要件

3. **ルート `.gitignore` の最適化**: 現在の `work/*` パターンを `/work/` に変更し、ルートディレクトリ直下のみに限定する意図を明確にすることを検討する（既に動作上は問題ないため優先度低）

## 実現方針 (Implementation Approach)

### ステップ 1: Gitからの削除

```bash
git rm -r features/devctl/work/
```

### ステップ 2: `.gitignore` の追加

`features/devctl/.gitignore` を作成（または既存ファイルに追記）:

```gitignore
# Worktree work directories (created at runtime)
work/
```

### ステップ 3: コミット

```bash
git add features/devctl/.gitignore
git commit -m "chore: remove stale work directory from features/devctl"
```

## 検証シナリオ (Verification Scenarios)

1. `git rm -r features/devctl/work/` を実行する
2. `features/devctl/.gitignore` に `work/` を追加する
3. `git status` で削除とignore追加が反映されていることを確認する
4. コミットする
5. `features/devctl/` 配下に `work/test/dummy.txt` を手動作成し、`git status` で追跡されないことを確認する
6. 手動作成したテストファイルをクリーンアップする
7. `scripts/process/build.sh` でビルドが通ることを確認する

## テスト項目 (Testing for the Requirements)

| 要件 | 検証方法 | コマンド |
|---|---|---|
| ディレクトリ削除 | Git履歴からの削除確認 | `git ls-files features/devctl/work/` が空であること |
| `.gitignore` 補強 | 新規ファイルが追跡されないことの確認 | `mkdir -p features/devctl/work/test && touch features/devctl/work/test/dummy.txt && git status --porcelain features/devctl/work/ | grep -c "^" | xargs test 0 -eq` |
| ビルド非破壊 | ビルドスクリプトの実行 | `scripts/process/build.sh` |
