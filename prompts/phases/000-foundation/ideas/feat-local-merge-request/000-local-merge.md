# ローカルマージワークフローの検討

> [!NOTE]
> この文書は検討段階のメモです。実装の方向性が確定した段階で正式な仕様書に昇格します。

## 背景

### 現在のワークフロー

```
tt open <branch>  →  コード編集  →  git add/commit/push  →  tt pr  →  GitHub PR マージ  →  tt close  →  git pull  →  ビルド
```

- GitHub PR の作成・マージが**必須**のステップとなっている
- 個人開発や小規模な変更ではオーバーヘッドが大きい
- リモートへの push → PR → マージ → pull というラウンドトリップが発生
- ネットワーク不要な作業でもネットワーク依存になっている

### 理想のローカルワークフロー

```
tt open <branch>  →  コード編集  →  git add/commit  →  tt merge  →  tt close  →  ビルド
```

- push / PR / pull の3ステップが不要
- ローカルの git merge だけで完結
- ネットワーク不要

---

## 検討事項

### 1. 親ブランチ（BaseBranch）の記録

`tt open <branch>` でworktreeを作成する時点で、親ブランチを特定して `StateFile` に記録する。

#### 技術的実現方法

`git worktree add -b <branch>` は、実行時点のHEADブランチから新ブランチを作成する。この時、以下のgitコマンドで親ブランチを取得できる:

```bash
git rev-parse --abbrev-ref HEAD
```

これを `StateFile` に `BaseBranch` フィールドとして記録する:

```yaml
# work/<branch>.state.yaml
branch: feat-xxx
base_branch: main          # ← 新規追加
created_at: 2026-03-15T...
code_status:
  status: local
```

#### StateFile への変更

```go
type StateFile struct {
    Branch     string                  `yaml:"branch"`
    BaseBranch string                  `yaml:"base_branch,omitempty"` // ← 新規追加
    CreatedAt  time.Time               `yaml:"created_at"`
    Features   map[string]FeatureState `yaml:"features,omitempty"`
    CodeStatus *CodeStatus             `yaml:"code_status,omitempty"`
}
```

#### メリット

- `tt merge` 実行時にマージ先ブランチが自動的に分かる（引数不要）
- 関係のないブランチへの誤マージを防止できる
- 既存のリモートブランチから `tt open` した場合も、その時点のHEADが記録される

#### 考慮点

- 既に `tt open` 済みの既存 worktree には `BaseBranch` がない → `tt merge` 時にフォールバックで `main` or `git` コマンドで推定
- リモートブランチから fetch して作成した場合、親ブランチの意味が変わる可能性がある（ただし通常は問題ない）

---

### 2. 新コマンド `tt merge` の導入

現在の `tt pr` に対応するローカル版として `tt merge` コマンドを新設する案。

#### 処理フロー

```
tt merge <branch>
```

1. `StateFile` から `BaseBranch` を取得（マージ先の自動解決）
2. worktree 内の uncommitted changes をチェック
3. ルートリポジトリ（`BaseBranch` が checkout されている場所）で `git merge <branch>` を実行
4. ステート (`CodeStatus`) を更新
5. 成功メッセージ表示

#### 確定事項

| 項目 | 結論 |
|------|------|
| **マージ先ブランチ** | ✅ `StateFile.BaseBranch` から自動解決。未記録の場合は `main` にフォールバック |
| **マージ戦略** | ✅ `--ff-only` をデフォルト。`--no-ff` / `--ff` をオプションで選択可能 |
| **コンフリクト処理** | ✅ 中断してコンフリクト状況を表示し、ユーザーに手動解決を促す |
| **squash マージ** | 📋 Phase 2 以降で対応（個人開発では優先度低い） |
| **worktree 制約** | ✅ 問題なし。`tt merge` は親ブランチ（ルートリポジトリ）で実行するため、checkout 制約に該当しない |
| **関係のないブランチへのマージ** | ✅ `BaseBranch` で防止。意図的に別ブランチへのマージが必要な場合は `--target` オプション |

### 3. 関係のないブランチへのマージについて

**結論: 基本的に不要**

`tt` のユースケースでは、worktree は常に特定の親ブランチから派生して作業を行い、完了後にその親ブランチに戻す、という一方向のフローを想定しています。

- `tt open <branch>` で `main` → `<branch>` を作成
- 作業完了後に `tt merge` で `<branch>` → `main` に戻す

この「往復」以外のマージ（例: `feature-a` → `feature-b`）は `tt` が管理すべきスコープ外であり、ユーザーが直接 `git merge` を使えばよいです。

ただし、将来的に必要になった場合に備えて `--target <branch>` オプションを用意しておく設計にはしておきたいです。

---

### 4. worktree 環境での git merge の実行方法

**結論: ルートリポジトリ（親ブランチ）で直接マージ**

`tt merge` は親ブランチ（main 等）がcheckoutされているルートリポジトリで実行する。worktree 内からの実行ではないため、git の「同一ブランチを複数 worktree で checkout できない」制約には該当しない。

```bash
# ルートリポジトリ (main ブランチ) で実行
cd <repo-root>
tt merge <branch>    # 内部で git merge <branch> を実行
```

事前チェック:
- ルートリポジトリの作業ディレクトリが dirty でないことを確認
- worktree 内の uncommitted changes がないことを確認

### 5. `tt close` への影響

現在の `tt close` は:
- ブランチがマージ済みかどうかを `git branch -d`（マージ済みのみ削除可能）で暗黙的にチェック
- マージ前に close すると `--force` が必要

ローカルマージ後は:
- ブランチがマージ済みと認識されるので `git branch -d` が正常に動作する
- `--force` フラグ不要になる

### 6. `CodeStatus` への影響

現在の `CodeStatus` の状態遷移:

```
Local → Hosted → PR → Deleted
```

ローカルマージの場合の状態遷移:

```
Local → Merged(Local)
```

新しいステータス `CodeStatusMerged` の追加が必要か、それとも既存フローに収めるか。

### 7. ハイブリッドワークフローの対応

ユーザーによっては:
- ある変更はローカルマージで済ませたい
- 別の変更は従来通り PR を使いたい

両方の選択肢を維持する設計が望ましい。`tt pr` と `tt merge` を並立させる。

---

## メリット・デメリット

### メリット
- ✅ ワークフローの高速化（ネットワーク不要）
- ✅ 個人開発での摩擦軽減
- ✅ オフライン作業対応
- ✅ `tt close` 時に `--force` 不要
- ✅ `git pull` ステップ不要

### デメリット・リスク
- ⚠️ レビュープロセスのスキップ（チームリポジトリでは不適切な可能性）
- ⚠️ `CodeStatus` の状態遷移が複雑化
- ⚠️ ルートリポジトリが dirty な場合の対処が必要

---

## 推奨案

### 最小実装（Phase 1）

**パターン A（ルートリポジトリで直接マージ）** を採用し、以下の制約付きで実装:

1. **`StateFile` に `BaseBranch` フィールドを追加**
   - `tt open` 時に `git rev-parse --abbrev-ref HEAD` で親ブランチを取得・記録
   - 既存 worktree には `BaseBranch` が未設定 → マージ時にフォールバック処理

2. **`tt merge <branch>` コマンド新設**
   - `StateFile.BaseBranch` をマージ先として自動解決
   - ルートリポジトリ（親ブランチ）で `git merge <branch>` を実行
   - マージ戦略オプション:
     - デフォルト: `--ff-only`（履歴がきれい、main に他の変更がない場合に成功）
     - `--no-ff`: 常にマージコミットを作成（ブランチ作業の記録を残したい場合）
     - `--ff`: git 標準動作に任せる（fast-forward 可能ならする、不可能ならマージコミット）
   - コンフリクト時: エラーで中断し、コンフリクト状況を表示してユーザーに手動解決を促す

3. **事前チェック**
   - worktree 内の uncommitted changes の有無
   - ルートリポジトリの dirty 状態チェック

4. **`tt close` との連携**
   - マージ済みブランチは `--force` なしで close 可能（既存動作で対応済み）

5. **`CodeStatus` の拡張は後回し**
   - Phase 1 では `CodeStatus` を `Local` のまま維持
   - close 時にブランチ削除が成功すれば実質マージ済みと判断可能

### 将来拡張（Phase 2 以降）

- `CodeStatusMerged` ステータスの追加
- `--squash` オプション対応（複数コミットを1つにまとめてマージ）
- `--target <branch>` オプション（BaseBranch 以外へのマージ）
- `tt merge --push` でマージ後に push するオプション

---

## 未解決の疑問

1. ルートリポジトリが dirty な場合の対処（エラーにする？自動 stash？）
2. ~~マージ先となるデフォルトブランチ名の解決方法~~ → `BaseBranch` で解決
3. ~~worktree 制約~~ → ルートリポジトリで実行するため問題なし
4. ~~マージ戦略~~ → `--ff-only` デフォルト、`--no-ff` / `--ff` オプション
5. ~~コンフリクト処理~~ → 中断して状況表示、ユーザーに解決を促す
6. ~~squash マージ~~ → Phase 2 以降
7. `BaseBranch` が未記録の既存 worktree のフォールバック方法
8. ネストした worktree がある場合のマージの影響範囲
